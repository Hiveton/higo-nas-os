package security

import (
	"context"
	"fmt"
	"strings"

	"higoos/server-go/internal/audit"
)

// Inspect runs an AI security sweep over current shares and identities, turning
// findings into pending risk actions (deduped by a stable id) that flow through
// the existing confirm/block/rollback machinery. Returns the full risk-action
// list afterwards.
func (s *Service) Inspect(ctx context.Context) ([]SecurityRiskAction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing := make(map[string]bool, len(s.risks))
	for _, r := range s.risks {
		existing[r.ID] = true
	}

	added := 0
	add := func(a SecurityRiskAction) {
		if existing[a.ID] {
			return
		}
		a.State = RiskActionStatePending
		a.AuditStatus = audit.RiskActionPending
		a.LevelLabel = riskLabel(a.Level)
		s.risks = append(s.risks, a)
		existing[a.ID] = true
		added++
	}

	for _, share := range s.shares {
		if share.Active && isPublicAccess(share.Access) {
			add(SecurityRiskAction{
				ID:                 "inspect-share-" + share.ID,
				Title:              "公开分享链接可被任何人访问：" + share.Name,
				Level:              audit.RiskHigh,
				Scope:              share.Target,
				Actor:              "安全巡检",
				Rollback:           "撤销该公开链接并恢复 ACL",
				RequiredPermission: "share.public",
				AffectedItemCount:  1,
			})
		}
	}

	for _, id := range s.identities {
		if id.AITools && !id.MFA {
			add(SecurityRiskAction{
				ID:                 "inspect-mfa-" + id.ID,
				Title:              fmt.Sprintf("%s 启用了 AI 工具但未开启 MFA", id.Name),
				Level:              audit.RiskMedium,
				Scope:              "身份与权限 / " + id.Role,
				Actor:              "安全巡检",
				Rollback:           "为该身份启用多因素认证",
				RequiredPermission: "iam.mfa",
				AffectedItemCount:  1,
			})
		}
	}

	if added > 0 {
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
	}
	return append([]SecurityRiskAction(nil), s.risks...), nil
}

func isPublicAccess(access string) bool {
	a := strings.ToLower(access)
	return strings.Contains(access, "公开") || strings.Contains(access, "任何") ||
		strings.Contains(a, "public") || strings.Contains(a, "anyone") || strings.Contains(a, "link")
}
