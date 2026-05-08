package security

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"higoos/server-go/internal/audit"
	"higoos/server-go/internal/iam"
	"higoos/server-go/internal/state"
)

type Service struct {
	mu         sync.RWMutex
	seq        int
	identities []IdentityPolicy
	aiPolicies []AiPolicy
	shares     []ShareLinkRisk
	risks      []SecurityRiskAction
	audit      []SecurityAuditEntry
	statePath  string
}

type snapshot struct {
	Seq        int                  `json:"seq"`
	Identities []IdentityPolicy     `json:"identities"`
	AIPolicies []AiPolicy           `json:"aiPolicies"`
	Shares     []ShareLinkRisk      `json:"shares"`
	Risks      []SecurityRiskAction `json:"risks"`
	Audit      []SecurityAuditEntry `json:"audit"`
}

func NewService() *Service {
	service := &Service{}
	service.seed()
	return service
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "security.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Identities) > 0 || len(persisted.AIPolicies) > 0 || len(persisted.Shares) > 0 || len(persisted.Risks) > 0 || len(persisted.Audit) > 0 {
		service.seq = persisted.Seq
		service.identities = append([]IdentityPolicy(nil), persisted.Identities...)
		service.aiPolicies = append([]AiPolicy(nil), persisted.AIPolicies...)
		service.shares = append([]ShareLinkRisk(nil), persisted.Shares...)
		service.risks = append([]SecurityRiskAction(nil), persisted.Risks...)
		service.audit = append([]SecurityAuditEntry(nil), persisted.Audit...)
		if service.seq < len(service.audit) {
			service.seq = len(service.audit)
		}
	}
	return service, nil
}

func (s *Service) Identities(ctx context.Context) ([]IdentityPolicy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]IdentityPolicy(nil), s.identities...), nil
}

func (s *Service) UpdateIdentityPermissions(ctx context.Context, id string, permissions IdentityPermissions) (IdentityPolicy, error) {
	if err := ctx.Err(); err != nil {
		return IdentityPolicy{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := findIdentity(s.identities, id)
	if !ok {
		return IdentityPolicy{}, fmt.Errorf("security identity not found: %s", id)
	}

	identity := &s.identities[index]
	identity.MFA = permissions.MFA
	identity.FileACL = permissions.FileACL
	identity.AppAdmin = permissions.AppAdmin
	identity.AITools = permissions.AITools
	s.prependAuditLocked(SecurityAuditEntry{
		Event:    fmt.Sprintf("调整%s文件夹 ACL", identity.Role),
		Actor:    "权限中心",
		Risk:     audit.RiskMedium,
		Reverted: false,
		Rollback: "恢复原 ACL",
		Result:   audit.ResultAllowed,
	})
	return *identity, s.saveLocked()
}

func (s *Service) AiPolicies(ctx context.Context) ([]AiPolicy, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]AiPolicy(nil), s.aiPolicies...), nil
}

func (s *Service) UpdateAiPolicy(ctx context.Context, id string, update AiPolicyUpdate) (AiPolicy, error) {
	if err := ctx.Err(); err != nil {
		return AiPolicy{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := findAiPolicy(s.aiPolicies, id)
	if !ok {
		return AiPolicy{}, fmt.Errorf("security ai policy not found: %s", id)
	}

	policy := &s.aiPolicies[index]
	if update.Indexed != nil {
		policy.Indexed = *update.Indexed
	}
	if update.CloudModel != nil {
		policy.CloudModel = *update.CloudModel
	}
	if update.Sensitive != "" {
		policy.Sensitive = update.Sensitive
	}
	s.prependAuditLocked(SecurityAuditEntry{
		Event:    fmt.Sprintf("%s 使用云模型摘要", policy.Space),
		Actor:    "AI 数据层",
		Risk:     audit.RiskMedium,
		Reverted: false,
		Rollback: "删除模型调用记录",
		Result:   audit.ResultAllowed,
	})
	return *policy, s.saveLocked()
}

func (s *Service) RiskActions(ctx context.Context) ([]SecurityRiskAction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SecurityRiskAction(nil), s.risks...), nil
}

func (s *Service) ConfirmRiskAction(ctx context.Context, id, actor string) (SecurityRiskAction, error) {
	if err := ctx.Err(); err != nil {
		return SecurityRiskAction{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := findRiskAction(s.risks, id)
	if !ok {
		return SecurityRiskAction{}, fmt.Errorf("security risk action not found: %s", id)
	}
	action := &s.risks[index]
	if action.State != RiskActionStatePending {
		return SecurityRiskAction{}, fmt.Errorf("security risk action is not pending: %s", id)
	}
	action.State = RiskActionStateConfirmed
	action.Confirmed = true
	action.AuditStatus = audit.RiskActionConfirmed
	s.prependAuditLocked(SecurityAuditEntry{
		Event:    fmt.Sprintf("确认执行：%s", action.Title),
		Actor:    actorOr(actor, action.Actor),
		Risk:     action.Level,
		Reverted: false,
		Rollback: action.Rollback,
		Result:   audit.ResultConfirmed,
	})
	return *action, s.saveLocked()
}

func (s *Service) BlockRiskAction(ctx context.Context, id, actor, reason string) (SecurityRiskAction, error) {
	if err := ctx.Err(); err != nil {
		return SecurityRiskAction{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := findRiskAction(s.risks, id)
	if !ok {
		return SecurityRiskAction{}, fmt.Errorf("security risk action not found: %s", id)
	}
	action := &s.risks[index]
	if action.State != RiskActionStatePending {
		return SecurityRiskAction{}, fmt.Errorf("security risk action is not pending: %s", id)
	}
	action.State = RiskActionStateBlocked
	action.Confirmed = false
	action.AuditStatus = audit.RiskActionBlocked
	rollback := "解除阻止并重新进入确认队列"
	if reason != "" {
		rollback = rollback + "：" + reason
	}
	s.prependAuditLocked(SecurityAuditEntry{
		Event:    fmt.Sprintf("阻止执行：%s", action.Title),
		Actor:    actorOr(actor, "安全中心"),
		Risk:     action.Level,
		Reverted: false,
		Rollback: rollback,
		Result:   audit.ResultBlocked,
	})
	return *action, s.saveLocked()
}

func (s *Service) Audit(ctx context.Context) ([]SecurityAuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]SecurityAuditEntry(nil), s.audit...), nil
}

func (s *Service) RollbackAudit(ctx context.Context, id, actor string) (SecurityAuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return SecurityAuditEntry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := findAuditEntry(s.audit, id)
	if !ok {
		return SecurityAuditEntry{}, fmt.Errorf("security audit entry not found: %s", id)
	}
	entry := &s.audit[index]
	if entry.Reverted {
		return SecurityAuditEntry{}, fmt.Errorf("security audit entry already rolled back: %s", id)
	}
	entry.Reverted = true
	entry.Result = audit.ResultRolledBack
	return *entry, s.saveLocked()
}

func (s *Service) Shares(ctx context.Context) ([]ShareLinkRisk, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]ShareLinkRisk(nil), s.shares...), nil
}

func (s *Service) DeleteShare(ctx context.Context, id, actor string) (ShareLinkRisk, error) {
	if err := ctx.Err(); err != nil {
		return ShareLinkRisk{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	index, ok := findShare(s.shares, id)
	if !ok {
		return ShareLinkRisk{}, fmt.Errorf("security share link not found: %s", id)
	}
	share := &s.shares[index]
	if !share.Active {
		return ShareLinkRisk{}, fmt.Errorf("security share link already revoked: %s", id)
	}
	share.Active = false
	s.prependAuditLocked(SecurityAuditEntry{
		Event:    fmt.Sprintf("撤销公开链接：%s", share.Name),
		Actor:    actorOr(actor, "分享链接安全检查"),
		Risk:     share.Risk,
		Reverted: false,
		Rollback: "恢复外链与访问密码",
		Result:   audit.ResultAllowed,
	})
	return *share, s.saveLocked()
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Seq:        s.seq,
		Identities: append([]IdentityPolicy(nil), s.identities...),
		AIPolicies: append([]AiPolicy(nil), s.aiPolicies...),
		Shares:     append([]ShareLinkRisk(nil), s.shares...),
		Risks:      append([]SecurityRiskAction(nil), s.risks...),
		Audit:      append([]SecurityAuditEntry(nil), s.audit...),
	})
}

func (s *Service) seed() {
	s.identities = []IdentityPolicy{
		{ID: "admin", Role: "管理员", IAMRole: iam.RoleAdmin, Name: "Hiveton", MFA: true, FileACL: true, AppAdmin: true, AITools: true},
		{ID: "family", Role: "家庭成员", IAMRole: iam.RoleFamilyMember, Name: "家人空间", MFA: true, FileACL: true, AppAdmin: false, AITools: true},
		{ID: "guest", Role: "访客", IAMRole: iam.RoleGuest, Name: "临时分享用户", MFA: false, FileACL: false, AppAdmin: false, AITools: false},
	}
	s.aiPolicies = []AiPolicy{
		{ID: "family-photos", Space: "家庭相册", SpaceID: "family-photos", Indexed: true, CloudModel: false, Sensitive: "人脸与定位仅本地索引"},
		{ID: "finance", Space: "财务票据", SpaceID: "finance", Indexed: false, CloudModel: false, Sensitive: "禁止进入 AI 分析"},
		{ID: "project-docs", Space: "项目资料", SpaceID: "project-docs", Indexed: true, CloudModel: true, Sensitive: "仅团队成员可问答"},
	}
	s.shares = []ShareLinkRisk{
		{ID: "s1", Name: "家庭相册春节精选", Target: "/家庭空间/相册/春节", Access: "密码 + 7 天", Downloads: 18, Risk: audit.RiskMedium, RiskLabel: riskLabel(audit.RiskMedium), Active: true},
		{ID: "s2", Name: "合同扫描件外链", Target: "/财务票据/合同", Access: "公开访问", Downloads: 4, Risk: audit.RiskHigh, RiskLabel: riskLabel(audit.RiskHigh), Active: true},
		{ID: "s3", Name: "安装包临时分发", Target: "/项目资料/release", Access: "团队可见", Downloads: 27, Risk: audit.RiskLow, RiskLabel: riskLabel(audit.RiskLow), Active: true},
	}
	s.risks = []SecurityRiskAction{
		{
			ID: "r1", Title: "Agent 申请批量重命名照片", Level: audit.RiskMedium, LevelLabel: riskLabel(audit.RiskMedium),
			Scope: "家庭相册 / 268 个文件", Actor: "相册整理 Agent", State: RiskActionStatePending, Confirmed: false,
			Rollback: "恢复原文件名快照", RequiredPermission: "files.write", AffectedItemCount: 268, AuditStatus: audit.RiskActionPending,
		},
		{
			ID: "r2", Title: "公开分享合同扫描件", Level: audit.RiskHigh, LevelLabel: riskLabel(audit.RiskHigh),
			Scope: "财务票据 / 合同扫描件", Actor: "外链分享", State: RiskActionStatePending, Confirmed: false,
			Rollback: "撤销链接并恢复 ACL", RequiredPermission: "share.public", AffectedItemCount: 1, AuditStatus: audit.RiskActionPending,
		},
		{
			ID: "r3", Title: "AI 摘要读取项目资料", Level: audit.RiskLow, LevelLabel: riskLabel(audit.RiskLow),
			Scope: "项目资料 / 只读摘要", Actor: "知识问答 Agent", State: RiskActionStatePending, Confirmed: false,
			Rollback: "清除本次摘要缓存", RequiredPermission: "ai.read", AffectedItemCount: 1, AuditStatus: audit.RiskActionPending,
		},
	}
	seedTime := time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC)
	s.audit = []SecurityAuditEntry{
		{ID: "a1", Event: "撤销公开链接：旧版报价单", Actor: "管理员", Risk: audit.RiskHigh, RiskLabel: riskLabel(audit.RiskHigh), Reverted: false, Rollback: "恢复链接撤销前状态", Result: audit.ResultAllowed, Time: seedTime.Add(2 * time.Minute)},
		{ID: "a2", Event: "调整访客文件夹 ACL", Actor: "权限中心", Risk: audit.RiskMedium, RiskLabel: riskLabel(audit.RiskMedium), Reverted: false, Rollback: "恢复原 ACL", Result: audit.ResultAllowed, Time: seedTime.Add(time.Minute)},
		{ID: "a3", Event: "项目资料使用云模型摘要", Actor: "AI 数据层", Risk: audit.RiskLow, RiskLabel: riskLabel(audit.RiskLow), Reverted: false, Rollback: "删除模型调用记录", Result: audit.ResultAllowed, Time: seedTime},
	}
	s.seq = len(s.audit)
}

func (s *Service) prependAuditLocked(entry SecurityAuditEntry) {
	s.seq++
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("security-audit-%03d", s.seq)
	}
	if entry.Time.IsZero() {
		entry.Time = time.Now().UTC()
	}
	if entry.RiskLabel == "" {
		entry.RiskLabel = riskLabel(entry.Risk)
	}
	s.audit = append([]SecurityAuditEntry{entry}, s.audit...)
}

func findIdentity(identities []IdentityPolicy, id string) (int, bool) {
	for index, identity := range identities {
		if identity.ID == id {
			return index, true
		}
	}
	return -1, false
}

func findAiPolicy(policies []AiPolicy, id string) (int, bool) {
	for index, policy := range policies {
		if policy.ID == id || policy.SpaceID == id {
			return index, true
		}
	}
	return -1, false
}

func findRiskAction(actions []SecurityRiskAction, id string) (int, bool) {
	for index, action := range actions {
		if action.ID == id {
			return index, true
		}
	}
	return -1, false
}

func findAuditEntry(entries []SecurityAuditEntry, id string) (int, bool) {
	for index, entry := range entries {
		if entry.ID == id {
			return index, true
		}
	}
	return -1, false
}

func findShare(shares []ShareLinkRisk, id string) (int, bool) {
	for index, share := range shares {
		if share.ID == id {
			return index, true
		}
	}
	return -1, false
}

func riskLabel(level audit.RiskLevel) string {
	switch level {
	case audit.RiskLow:
		return "低风险"
	case audit.RiskMedium:
		return "中风险"
	case audit.RiskHigh:
		return "高风险"
	default:
		return string(level)
	}
}

func actorOr(actor, fallback string) string {
	if actor != "" {
		return actor
	}
	return fallback
}
