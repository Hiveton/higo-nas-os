package steward

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

type Service struct {
	mu          sync.RWMutex
	now         func() time.Time
	nextPreview int
	nextAudit   int
	suggestions map[string]Suggestion
	order       []string
	previews    map[string]SuggestionPreview
	audit       []AuditEntry
	statePath   string
}

type snapshot struct {
	NextPreview int                          `json:"nextPreview"`
	NextAudit   int                          `json:"nextAudit"`
	Suggestions map[string]Suggestion        `json:"suggestions"`
	Order       []string                     `json:"order"`
	Previews    map[string]SuggestionPreview `json:"previews"`
	Audit       []AuditEntry                 `json:"audit"`
}

func NewService() *Service {
	service := &Service{
		now:         time.Now,
		suggestions: make(map[string]Suggestion),
		previews:    make(map[string]SuggestionPreview),
	}
	service.seed()
	return service
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "steward.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Suggestions) > 0 {
		service.nextPreview = persisted.NextPreview
		service.nextAudit = persisted.NextAudit
		service.suggestions = cloneSuggestionMap(persisted.Suggestions)
		service.order = append([]string(nil), persisted.Order...)
		service.previews = clonePreviewMap(persisted.Previews)
		service.audit = cloneAudit(persisted.Audit)
	}
	return service, nil
}

func (s *Service) ListSuggestions(ctx context.Context) []Suggestion {
	if ctx.Err() != nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Suggestion, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.suggestions[id])
	}
	return out
}

func (s *Service) Preview(ctx context.Context, id string, request PreviewRequest) (SuggestionPreview, error) {
	if err := ctx.Err(); err != nil {
		return SuggestionPreview{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	suggestion, ok := s.suggestions[id]
	if !ok {
		return SuggestionPreview{}, fmt.Errorf("steward suggestion not found: %s", id)
	}
	s.nextPreview++
	preview := SuggestionPreview{
		SuggestionID: id,
		Impact:       impactForSuggestion(suggestion),
		Risk:         suggestion.Risk,
	}
	if requiresConfirmation(suggestion.Risk) {
		preview.RequiresConfirmation = true
		preview.ConfirmationID = fmt.Sprintf("steward-confirm-%03d", s.nextPreview)
		preview.RollbackID = fmt.Sprintf("steward-rollback-%03d", s.nextPreview)
	}
	s.previews[id] = preview
	return preview, s.saveLocked()
}

func (s *Service) Confirm(ctx context.Context, id string, request ConfirmRequest) (ConfirmResult, error) {
	if err := ctx.Err(); err != nil {
		return ConfirmResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	suggestion, ok := s.suggestions[id]
	if !ok {
		return ConfirmResult{}, fmt.Errorf("steward suggestion not found: %s", id)
	}
	if suggestion.Status != SuggestionPending {
		return ConfirmResult{}, fmt.Errorf("steward suggestion is not pending: %s", id)
	}
	preview := s.previews[id]
	if requiresConfirmation(suggestion.Risk) {
		if preview.ConfirmationID == "" {
			return ConfirmResult{}, fmt.Errorf("steward suggestion requires preview before confirmation")
		}
		if request.ConfirmationID != preview.ConfirmationID {
			return ConfirmResult{}, fmt.Errorf("steward confirmation mismatch")
		}
	}
	suggestion.Status = SuggestionConfirmed
	suggestion.UpdateAt = s.now().UTC()
	s.suggestions[id] = suggestion
	audit := s.appendAuditLocked(AuditEntry{
		SuggestionID:   id,
		Message:        fmt.Sprintf("已确认执行：%s", suggestion.Title),
		ActorID:        request.ActorID,
		Risk:           suggestion.Risk,
		Result:         AuditConfirmed,
		ConfirmationID: preview.ConfirmationID,
		RollbackID:     preview.RollbackID,
	})
	return ConfirmResult{Suggestion: suggestion, AuditEntry: audit}, s.saveLocked()
}

func (s *Service) Dismiss(ctx context.Context, id string, request DismissRequest) (Suggestion, error) {
	if err := ctx.Err(); err != nil {
		return Suggestion{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	suggestion, ok := s.suggestions[id]
	if !ok {
		return Suggestion{}, fmt.Errorf("steward suggestion not found: %s", id)
	}
	if suggestion.Status != SuggestionPending {
		return Suggestion{}, fmt.Errorf("steward suggestion is not pending: %s", id)
	}
	suggestion.Status = SuggestionDismissed
	suggestion.UpdateAt = s.now().UTC()
	s.suggestions[id] = suggestion
	message := fmt.Sprintf("已忽略建议：%s", suggestion.Title)
	if strings.TrimSpace(request.Reason) != "" {
		message = fmt.Sprintf("%s，原因：%s", message, request.Reason)
	}
	s.appendAuditLocked(AuditEntry{
		SuggestionID: id,
		Message:      message,
		ActorID:      request.ActorID,
		Risk:         suggestion.Risk,
		Result:       AuditDismissed,
	})
	return suggestion, s.saveLocked()
}

func (s *Service) Audit(ctx context.Context) []AuditEntry {
	if ctx.Err() != nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAudit(s.audit)
}

func (s *Service) Rollback(ctx context.Context, auditID string, request RollbackRequest) (AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return AuditEntry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.audit {
		if s.audit[i].ID != auditID {
			continue
		}
		if s.audit[i].RollbackID == "" {
			return AuditEntry{}, fmt.Errorf("steward audit entry has no rollback: %s", auditID)
		}
		s.audit[i].Result = AuditRolledBack
		s.audit[i].RolledBackAt = s.now().UTC()
		if suggestion, ok := s.suggestions[s.audit[i].SuggestionID]; ok {
			suggestion.Status = SuggestionPending
			suggestion.UpdateAt = s.audit[i].RolledBackAt
			s.suggestions[suggestion.ID] = suggestion
		}
		rolledBack := s.audit[i]
		s.appendAuditLocked(AuditEntry{
			SuggestionID: s.audit[i].SuggestionID,
			Message:      fmt.Sprintf("已回滚：%s", s.audit[i].Message),
			ActorID:      request.ActorID,
			Risk:         s.audit[i].Risk,
			Result:       AuditRolledBack,
			RollbackID:   s.audit[i].RollbackID,
		})
		return rolledBack, s.saveLocked()
	}
	return AuditEntry{}, fmt.Errorf("steward audit entry not found: %s", auditID)
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		NextPreview: s.nextPreview,
		NextAudit:   s.nextAudit,
		Suggestions: cloneSuggestionMap(s.suggestions),
		Order:       append([]string(nil), s.order...),
		Previews:    clonePreviewMap(s.previews),
		Audit:       cloneAudit(s.audit),
	})
}

func (s *Service) seed() {
	now := s.now().UTC()
	seeds := []Suggestion{
		{ID: "download-cleanup", Title: "下载目录智能整理", Detail: "31 张发票、12 个安装包和 4 个重复压缩包可按规则归档。", Count: "47 项", Risk: RiskMedium, Action: "预览整理", Status: SuggestionPending, UpdateAt: now},
		{ID: "expired-share-links", Title: "过期分享链接", Detail: "发现 3 个公开链接仍可访问，包含团队空间资料。", Count: "3 个", Risk: RiskHigh, Action: "查看权限", Status: SuggestionPending, UpdateAt: now},
		{ID: "similar-photo-cleanup", Title: "相似照片清理", Detail: "五一旅行相册中有 86 张连拍相似照片，可保留清晰版本。", Count: "1.6 GB", Risk: RiskLow, Action: "智能筛选", Status: SuggestionPending, UpdateAt: now},
	}
	for _, suggestion := range seeds {
		s.suggestions[suggestion.ID] = suggestion
		s.order = append(s.order, suggestion.ID)
	}
	for _, message := range []string{
		"09:41 文件管家读取 /下载/票据，仅生成建议，未移动文件",
		"09:22 Agent 创建家庭保修提醒，等待管理员确认",
		"昨天 18:36 撤销 12 个文件重命名，已恢复原路径",
	} {
		s.appendAuditLocked(AuditEntry{Message: message, Risk: RiskLow, Result: AuditAllowed})
	}
}

func (s *Service) appendAuditLocked(entry AuditEntry) AuditEntry {
	s.nextAudit++
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("steward-audit-%03d", s.nextAudit)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = s.now().UTC()
	}
	s.audit = append([]AuditEntry{entry}, s.audit...)
	return entry
}

func impactForSuggestion(suggestion Suggestion) string {
	switch suggestion.ID {
	case "download-cleanup":
		return "将预览移动、重命名和重复项处理计划，确认前不移动文件。"
	case "expired-share-links":
		return "将列出公开链接和影响范围，确认前不修改权限。"
	case "similar-photo-cleanup":
		return "仅生成相似照片筛选清单，默认不删除原图。"
	default:
		return fmt.Sprintf("将预览 %s 的影响范围。", suggestion.Title)
	}
}

func requiresConfirmation(risk RiskLevel) bool {
	return risk == RiskMedium || risk == RiskHigh
}

func cloneAudit(entries []AuditEntry) []AuditEntry {
	out := make([]AuditEntry, len(entries))
	copy(out, entries)
	return out
}

func cloneSuggestionMap(suggestions map[string]Suggestion) map[string]Suggestion {
	out := make(map[string]Suggestion, len(suggestions))
	for key, suggestion := range suggestions {
		out[key] = suggestion
	}
	return out
}

func clonePreviewMap(previews map[string]SuggestionPreview) map[string]SuggestionPreview {
	out := make(map[string]SuggestionPreview, len(previews))
	for key, preview := range previews {
		out[key] = preview
	}
	return out
}
