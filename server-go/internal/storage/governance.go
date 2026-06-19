package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"higoos/server-go/internal/audit"
)

// confirmationTTL bounds how long a delete confirmation token stays valid. A
// short window keeps single-use tokens from lingering after a previewed delete.
const confirmationTTL = 5 * time.Minute

// pendingDelete is a previewed-but-not-yet-executed space deletion. It carries
// no side effect until the matching ConfirmDeleteSpace call runs.
type pendingDelete struct {
	SpaceID        string          `json:"spaceId"`
	Name           string          `json:"name"`
	ConfirmationID string          `json:"confirmationId"`
	Actor          string          `json:"actor"`
	Risk           audit.RiskLevel `json:"risk"`
	Impact         string          `json:"impact"`
	CreatedAt      time.Time       `json:"createdAt"`
}

// StorageDeletePreview is returned by the preview endpoint: it summarizes the
// impact and hands back a single-use confirmationId, with no side effect.
type StorageDeletePreview struct {
	SpaceID              string    `json:"spaceId"`
	Name                 string    `json:"name"`
	Mode                 string    `json:"mode"`
	Total                string    `json:"total"`
	DiskSlots            []string  `json:"diskSlots"`
	Impact               string    `json:"impact"`
	Risk                 string    `json:"risk"`
	RiskLabel            string    `json:"riskLabel"`
	ConfirmationID       string    `json:"confirmationId"`
	RequiresConfirmation bool      `json:"requiresConfirmation"`
	ExpiresAt            time.Time `json:"expiresAt"`
}

// ConfirmDeleteRequest applies a previewed deletion.
type ConfirmDeleteRequest struct {
	ConfirmationID string `json:"confirmationId"`
	Actor          string `json:"actor"`
}

// StorageAuditEntry is an append-only governance record for high-risk storage
// actions (newest first).
type StorageAuditEntry struct {
	ID             string    `json:"id"`
	Event          string    `json:"event"`
	Actor          string    `json:"actor"`
	Result         string    `json:"result"`
	Impact         string    `json:"impact"`
	Risk           string    `json:"risk"`
	RiskLabel      string    `json:"riskLabel"`
	SpaceID        string    `json:"spaceId"`
	ConfirmationID string    `json:"confirmationId"`
	Time           time.Time `json:"time"`
}

func storageRiskLabel(level audit.RiskLevel) string {
	switch level {
	case audit.RiskHigh:
		return "高风险"
	case audit.RiskMedium:
		return "中风险"
	default:
		return "低风险"
	}
}

// PreviewDeleteSpace registers a single-use, high-risk confirmation for deleting
// a storage space and returns the impact summary. No state is removed here.
func (s *Service) PreviewDeleteSpace(ctx context.Context, id, actor string) (StorageDeletePreview, error) {
	if err := ctx.Err(); err != nil {
		return StorageDeletePreview{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	var space *StorageSpace
	for i := range s.spaces {
		if s.spaces[i].ID == id {
			space = &s.spaces[i]
			break
		}
	}
	if space == nil {
		return StorageDeletePreview{}, fmt.Errorf("storage space not found: %s", id)
	}

	disks := "无"
	if len(space.DiskSlots) > 0 {
		disks = strings.Join(space.DiskSlots, "、")
	}
	impact := fmt.Sprintf("将永久删除存储空间「%s」(模式 %s,容量 %s,磁盘 %s)。空间内数据不可恢复,请先确认已备份或建立快照。",
		space.Name, space.Mode, space.Total, disks)

	now := s.now().UTC()
	s.confirmSeq++
	confirmationID := fmt.Sprintf("storage-confirm-%03d", s.confirmSeq)
	if s.pending == nil {
		s.pending = map[string]pendingDelete{}
	}
	s.pending[confirmationID] = pendingDelete{
		SpaceID:        space.ID,
		Name:           space.Name,
		ConfirmationID: confirmationID,
		Actor:          actor,
		Risk:           audit.RiskHigh,
		Impact:         impact,
		CreatedAt:      now,
	}
	if err := s.saveLocked(); err != nil {
		return StorageDeletePreview{}, err
	}

	return StorageDeletePreview{
		SpaceID:              space.ID,
		Name:                 space.Name,
		Mode:                 string(space.Mode),
		Total:                space.Total,
		DiskSlots:            append([]string(nil), space.DiskSlots...),
		Impact:               impact,
		Risk:                 string(audit.RiskHigh),
		RiskLabel:            storageRiskLabel(audit.RiskHigh),
		ConfirmationID:       confirmationID,
		RequiresConfirmation: true,
		ExpiresAt:            now.Add(confirmationTTL),
	}, nil
}

// ConfirmDeleteSpace validates the single-use token (presence + TTL), removes
// the space, and records an audit entry. The token is consumed on any outcome.
func (s *Service) ConfirmDeleteSpace(ctx context.Context, req ConfirmDeleteRequest) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	pending, ok := s.pending[req.ConfirmationID]
	if !ok {
		return StorageTask{}, fmt.Errorf("no pending deletion for confirmation: %s", req.ConfirmationID)
	}
	// Single-use: consume the token regardless of what happens below.
	delete(s.pending, req.ConfirmationID)

	if s.now().UTC().Sub(pending.CreatedAt) > confirmationTTL {
		_ = s.saveLocked()
		return StorageTask{}, fmt.Errorf("confirmation %s has expired", req.ConfirmationID)
	}

	index := -1
	for i, space := range s.spaces {
		if space.ID == pending.SpaceID {
			index = i
			break
		}
	}
	if index < 0 {
		_ = s.saveLocked()
		return StorageTask{}, fmt.Errorf("storage space not found: %s", pending.SpaceID)
	}
	s.spaces = append(s.spaces[:index], s.spaces[index+1:]...)

	actor := pending.Actor
	if strings.TrimSpace(req.Actor) != "" {
		actor = req.Actor
	}
	s.appendAuditLocked(StorageAuditEntry{
		ID:             fmt.Sprintf("storage-audit-%03d", s.confirmSeq),
		Event:          fmt.Sprintf("删除存储空间「%s」", pending.Name),
		Actor:          actor,
		Result:         string(audit.ResultConfirmed),
		Impact:         pending.Impact,
		Risk:           string(pending.Risk),
		RiskLabel:      storageRiskLabel(pending.Risk),
		SpaceID:        pending.SpaceID,
		ConfirmationID: pending.ConfirmationID,
		Time:           s.now().UTC(),
	})

	// completedTaskLocked persists the whole snapshot (spaces, pending, audit).
	return s.completedTaskLocked(TaskKindDeleteSpace, TaskTarget{TargetPool: pending.SpaceID}, "存储空间已删除")
}

// Audit returns the high-risk storage audit trail, newest first.
func (s *Service) Audit() []StorageAuditEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]StorageAuditEntry(nil), s.audit...)
}

// appendAuditLocked prepends an entry and caps the trail length. Caller holds the lock.
func (s *Service) appendAuditLocked(entry StorageAuditEntry) {
	s.audit = append([]StorageAuditEntry{entry}, s.audit...)
	const maxAudit = 200
	if len(s.audit) > maxAudit {
		s.audit = s.audit[:maxAudit]
	}
}

func clonePending(src map[string]pendingDelete) map[string]pendingDelete {
	if src == nil {
		return map[string]pendingDelete{}
	}
	out := make(map[string]pendingDelete, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}
