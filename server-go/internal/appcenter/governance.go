package appcenter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"higoos/server-go/internal/audit"
	"higoos/server-go/internal/state"
)

// Supported governed actions.
const (
	ActionInstall   = "install"
	ActionUpdate    = "update"
	ActionStart     = "start"
	ActionStop      = "stop"
	ActionUninstall = "uninstall"
)

// pendingConfirm is a previewed-but-not-yet-executed action. Side effects only
// happen once the matching confirmationId is presented to ConfirmAction.
type pendingConfirm struct {
	Action     string
	AppID      string
	Config     map[string]string
	Risk       audit.RiskLevel
	Impact     string
	RollbackID string
	CreatedAt  time.Time
}

// appSnapshot captures the reversible install state of an app before a governed
// action executes, so Rollback can restore it.
type appSnapshot struct {
	Installed   bool              `json:"installed"`
	Running     bool              `json:"running"`
	Version     string            `json:"version"`
	Status      string            `json:"status"`
	ContainerID string            `json:"containerId,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
}

// AuditRecord is the app-center-scoped audit entry surfaced to the UI. It also
// carries the pre-action snapshot needed to reverse the action.
type AuditRecord struct {
	ID             string       `json:"id"`
	AppID          string       `json:"appId"`
	AppName        string       `json:"appName"`
	Action         string       `json:"action"`
	Actor          string       `json:"actor"`
	Risk           string       `json:"risk"`
	Result         string       `json:"result"` // confirmed|rolled_back
	Message        string       `json:"message"`
	ConfirmationID string       `json:"confirmationId"`
	RollbackID     string       `json:"rollbackId"`
	Rollbackable   bool         `json:"rollbackable"`
	CreatedAt      time.Time    `json:"createdAt"`
	RolledBackAt   *time.Time   `json:"rolledBackAt,omitempty"`
	Snapshot       *appSnapshot `json:"snapshot,omitempty"`
}

// ActionPreview is returned by PreviewAction: the impact summary plus the
// confirmationId the caller must echo back to ConfirmAction.
type ActionPreview struct {
	AppID                string `json:"appId"`
	Action               string `json:"action"`
	ConfirmationID       string `json:"confirmationId"`
	RollbackID           string `json:"rollbackId"`
	Risk                 string `json:"risk"`
	Impact               string `json:"impact"`
	RequiresConfirmation bool   `json:"requiresConfirmation"`
}

// ActionResult is returned by ConfirmAction.
type ActionResult struct {
	App     App    `json:"app"`
	AuditID string `json:"auditId"`
	Result  string `json:"result"`
	Message string `json:"message"`
}

// PreviewAction validates the action, computes its risk + impact summary, and
// registers a pending confirmation. It has no side effects.
func (s *Service) PreviewAction(ctx context.Context, id, action string, config map[string]string) (ActionPreview, error) {
	if err := ctx.Err(); err != nil {
		return ActionPreview{}, err
	}
	app, ok := s.Get(id)
	if !ok {
		if _, inCatalog := s.catalog.Lookup(id); !inCatalog {
			return ActionPreview{}, fmt.Errorf("app center item not found: %s", id)
		}
	}
	if !isGovernedAction(action) {
		return ActionPreview{}, fmt.Errorf("unsupported action: %s", action)
	}
	risk := s.actionRisk(id, action)
	impact := s.impactSummary(id, action, app)

	s.mu.Lock()
	s.nextConfirm++
	confirmationID := fmt.Sprintf("appcenter-confirm-%03d", s.nextConfirm)
	rollbackID := fmt.Sprintf("appcenter-rollback-%03d", s.nextConfirm)
	s.confirms[confirmationID] = pendingConfirm{
		Action:     action,
		AppID:      id,
		Config:     cloneStrMap(config),
		Risk:       risk,
		Impact:     impact,
		RollbackID: rollbackID,
		CreatedAt:  time.Now().UTC(),
	}
	s.mu.Unlock()

	return ActionPreview{
		AppID:                id,
		Action:               action,
		ConfirmationID:       confirmationID,
		RollbackID:           rollbackID,
		Risk:                 string(risk),
		Impact:               impact,
		RequiresConfirmation: true,
	}, nil
}

// ConfirmAction validates the confirmationId, snapshots the pre-action state,
// executes the lifecycle operation, records an audit entry (locally + in the
// shared store) and returns the resulting app.
func (s *Service) ConfirmAction(ctx context.Context, id, action, confirmationID, actor string) (ActionResult, error) {
	s.mu.Lock()
	pending, ok := s.confirms[confirmationID]
	if ok {
		delete(s.confirms, confirmationID)
	}
	s.mu.Unlock()
	if !ok || pending.AppID != id || pending.Action != action {
		return ActionResult{}, fmt.Errorf("invalid or expired confirmation: %s", confirmationID)
	}
	if actor == "" {
		actor = "operator"
	}

	before := s.snapshotOf(id)
	app, err := s.execute(ctx, id, action, pending.Config)
	result := audit.ResultConfirmed
	message := actionMessage(action, app)
	if err != nil {
		result = audit.ResultFailed
		message = err.Error()
	}

	record := s.recordAudit(id, app, action, actor, pending, result, message, before)
	if err != nil {
		return ActionResult{}, err
	}
	return ActionResult{App: app, AuditID: record.ID, Result: string(result), Message: message}, nil
}

// execute dispatches a governed action to the corresponding lifecycle method.
func (s *Service) execute(ctx context.Context, id, action string, config map[string]string) (App, error) {
	switch action {
	case ActionInstall:
		return s.InstallWithConfig(ctx, id, config)
	case ActionUpdate:
		return s.Update(ctx, id)
	case ActionStart:
		return s.Start(ctx, id)
	case ActionStop:
		return s.Stop(ctx, id)
	case ActionUninstall:
		removeVolumes := config["removeVolumes"] == "true"
		return s.Uninstall(ctx, id, removeVolumes)
	default:
		return App{}, fmt.Errorf("unsupported action: %s", action)
	}
}

// Rollback reverses a previously-confirmed action by restoring the captured
// snapshot (re-installing, uninstalling, or flipping run state as needed).
func (s *Service) Rollback(ctx context.Context, auditID, actor string) (AuditRecord, error) {
	s.mu.Lock()
	idx := -1
	for i := range s.auditLog {
		if s.auditLog[i].ID == auditID {
			idx = i
			break
		}
	}
	if idx == -1 {
		s.mu.Unlock()
		return AuditRecord{}, fmt.Errorf("audit entry not found: %s", auditID)
	}
	rec := s.auditLog[idx]
	s.mu.Unlock()

	if !rec.Rollbackable || rec.Snapshot == nil {
		return AuditRecord{}, fmt.Errorf("audit entry %s is not rollbackable", auditID)
	}
	if rec.RolledBackAt != nil {
		return AuditRecord{}, fmt.Errorf("audit entry %s is already rolled back", auditID)
	}

	if err := s.restoreSnapshot(ctx, rec.AppID, *rec.Snapshot); err != nil {
		return AuditRecord{}, err
	}

	now := time.Now().UTC()
	s.mu.Lock()
	s.auditLog[idx].RolledBackAt = &now
	s.auditLog[idx].Result = string(audit.ResultRolledBack)
	s.auditLog[idx].Rollbackable = false
	out := s.auditLog[idx]
	_ = s.saveAuditLocked()
	s.mu.Unlock()

	if s.audit != nil && rec.RollbackID != "" {
		s.audit.MarkRollback(rec.RollbackID, audit.RollbackSucceeded, "appcenter rollback")
		s.audit.Append(audit.AuditEvent{
			Domain: "app-center", Action: "appcenter." + rec.Action + ".rollback",
			ActorID: actor, Risk: audit.RiskLevel(rec.Risk), Result: audit.ResultRolledBack,
			TargetIDs: []string{rec.AppID},
		})
	}
	return out, nil
}

// AuditLog returns the app-center audit entries (newest first).
func (s *Service) AuditLog(ctx context.Context) ([]AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AuditRecord, 0, len(s.auditLog))
	for i := len(s.auditLog) - 1; i >= 0; i-- {
		r := s.auditLog[i]
		r.Snapshot = nil // never leak the internal snapshot/config to clients
		out = append(out, r)
	}
	return out, nil
}

// ----- internal helpers -----

func (s *Service) restoreSnapshot(ctx context.Context, id string, snap appSnapshot) error {
	switch {
	case snap.Installed && snap.Running:
		_, err := s.InstallWithConfig(ctx, id, snap.Config)
		return err
	case snap.Installed && !snap.Running:
		if _, err := s.InstallWithConfig(ctx, id, snap.Config); err != nil {
			return err
		}
		_, err := s.Stop(ctx, id)
		return err
	default:
		_, err := s.Uninstall(ctx, id, false)
		return err
	}
}

func (s *Service) snapshotOf(id string) appSnapshot {
	p, ok := s.find(id)
	if !ok {
		return appSnapshot{}
	}
	return appSnapshot{
		Installed:   p.Installed,
		Running:     p.Running,
		Version:     p.Version,
		Status:      p.Status,
		ContainerID: p.ContainerID,
		Config:      cloneStrMap(p.ConfigValues),
	}
}

func (s *Service) recordAudit(id string, app App, action, actor string, pending pendingConfirm, result audit.AuditResult, message string, before appSnapshot) AuditRecord {
	name := app.Name
	if name == "" {
		if p, ok := s.find(id); ok {
			name = p.Name
		}
	}
	rollbackable := result == audit.ResultConfirmed
	rec := AuditRecord{
		AppID:          id,
		AppName:        name,
		Action:         action,
		Actor:          actor,
		Risk:           string(pending.Risk),
		Result:         string(result),
		Message:        message,
		ConfirmationID: pending.RollbackID, // surfaced rollback handle is the rollbackId
		RollbackID:     pending.RollbackID,
		Rollbackable:   rollbackable,
		CreatedAt:      time.Now().UTC(),
	}
	if rollbackable {
		snap := before
		rec.Snapshot = &snap
	}

	s.mu.Lock()
	rec.ID = fmt.Sprintf("appcenter-audit-%03d", len(s.auditLog)+1)
	s.auditLog = append(s.auditLog, rec)
	_ = s.saveAuditLocked()
	s.mu.Unlock()

	if s.audit != nil {
		var rollback *audit.RollbackOperation
		if rollbackable {
			rollback = &audit.RollbackOperation{
				ID:       pending.RollbackID,
				Type:     "appcenter." + action,
				TargetID: id,
				Before:   before.Status,
				After:    app.Status,
			}
		}
		s.audit.CreateRiskAction(audit.RiskAction{
			ActorID:       actor,
			Action:        "appcenter." + action,
			Risk:          pending.Risk,
			ImpactSummary: pending.Impact,
			Status:        audit.RiskActionConfirmed,
			ConfirmedBy:   actor,
			ConfirmedAt:   time.Now().UTC(),
			Rollback:      rollback,
		})
		s.audit.Append(audit.AuditEvent{
			Domain: "app-center", Action: "appcenter." + action,
			ActorID: actor, ConfirmationID: pending.RollbackID,
			Risk: pending.Risk, Result: result, TargetIDs: []string{id},
			BeforeSummary: before.Status, AfterSummary: app.Status,
		})
	}
	return rec
}

func (s *Service) saveAuditLocked() error {
	if s.auditPath == "" {
		return nil
	}
	return state.SaveJSON(s.auditPath, s.auditLog)
}

func (s *Service) actionRisk(id, action string) audit.RiskLevel {
	switch action {
	case ActionUninstall:
		return audit.RiskHigh
	case ActionStart, ActionStop:
		return audit.RiskLow
	default:
		if m, ok := s.manifestFor(id); ok {
			return m.RiskLevel()
		}
		return audit.RiskMedium
	}
}

func (s *Service) impactSummary(id, action string, app App) string {
	m, ok := s.manifestFor(id)
	name := app.Name
	if name == "" && ok {
		name = m.Name
	}
	switch action {
	case ActionInstall:
		if !ok {
			return fmt.Sprintf("安装 %s", name)
		}
		parts := []string{fmt.Sprintf("拉取并启动镜像 %s", imagesOf(m))}
		if ports := m.portStrings(); len(ports) > 0 {
			parts = append(parts, "发布端口 "+strings.Join(ports, "、"))
		}
		if len(m.Permissions) > 0 {
			parts = append(parts, "授予权限 "+strings.Join(m.Permissions, "、"))
		}
		parts = append(parts, "资源 "+resourceText(m.totalResources()))
		return fmt.Sprintf("安装 %s：%s。", name, strings.Join(parts, "；"))
	case ActionUpdate:
		return fmt.Sprintf("将 %s 更新到 %s 并重建容器。", name, app.LatestVersion)
	case ActionUninstall:
		return fmt.Sprintf("卸载 %s：停止并销毁其容器（数据卷默认保留）。", name)
	case ActionStart:
		return fmt.Sprintf("启动 %s 的容器。", name)
	case ActionStop:
		return fmt.Sprintf("停止 %s 的容器。", name)
	default:
		return action
	}
}

func actionMessage(action string, app App) string {
	switch action {
	case ActionInstall:
		return fmt.Sprintf("%s 已安装并启动。", app.Name)
	case ActionUpdate:
		return fmt.Sprintf("%s 已更新到 %s。", app.Name, app.Version)
	case ActionStart:
		return fmt.Sprintf("%s 已启动。", app.Name)
	case ActionStop:
		return fmt.Sprintf("%s 已停止。", app.Name)
	case ActionUninstall:
		return fmt.Sprintf("%s 已卸载。", app.Name)
	default:
		return action
	}
}

func isGovernedAction(action string) bool {
	switch action {
	case ActionInstall, ActionUpdate, ActionStart, ActionStop, ActionUninstall:
		return true
	default:
		return false
	}
}
