package security

import (
	"context"
	"testing"

	"higoos/server-go/internal/audit"
)

func TestServiceUpdatesIdentityPermissions(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	identity, err := service.UpdateIdentityPermissions(ctx, "guest", IdentityPermissions{
		MFA:      true,
		FileACL:  true,
		AppAdmin: false,
		AITools:  false,
	})
	if err != nil {
		t.Fatalf("update identity permissions: %v", err)
	}

	if !identity.MFA || !identity.FileACL || identity.AppAdmin || identity.AITools {
		t.Fatalf("unexpected identity permissions: %#v", identity)
	}

	identities, err := service.Identities(ctx)
	if err != nil {
		t.Fatalf("list identities: %v", err)
	}
	if identities[2].ID != "guest" || !identities[2].MFA || !identities[2].FileACL {
		t.Fatalf("identity list was not updated: %#v", identities[2])
	}

	entries, err := service.Audit(ctx)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(entries) != 4 || entries[0].Event != "调整访客文件夹 ACL" {
		t.Fatalf("expected permission audit entry to be prepended, got %#v", entries)
	}
}

func TestServiceUpdatesAiPolicy(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	policy, err := service.UpdateAiPolicy(ctx, "finance", AiPolicyUpdate{
		Indexed:    boolPtr(true),
		CloudModel: boolPtr(false),
		Sensitive:  "票据索引仅本地可见",
	})
	if err != nil {
		t.Fatalf("update ai policy: %v", err)
	}

	if !policy.Indexed || policy.CloudModel || policy.Sensitive != "票据索引仅本地可见" {
		t.Fatalf("unexpected policy: %#v", policy)
	}

	entries, err := service.Audit(ctx)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(entries) != 4 || entries[0].Actor != "AI 数据层" || entries[0].Risk != audit.RiskMedium {
		t.Fatalf("expected ai policy audit entry, got %#v", entries[0])
	}
}

func TestServiceConfirmsAndBlocksRiskActionsWithAudit(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	confirmed, err := service.ConfirmRiskAction(ctx, "r1", "Hiveton")
	if err != nil {
		t.Fatalf("confirm risk action: %v", err)
	}
	if confirmed.State != RiskActionStateConfirmed || !confirmed.Confirmed {
		t.Fatalf("unexpected confirmed action: %#v", confirmed)
	}

	blocked, err := service.BlockRiskAction(ctx, "r2", "安全中心", "外链包含敏感合同")
	if err != nil {
		t.Fatalf("block risk action: %v", err)
	}
	if blocked.State != RiskActionStateBlocked || blocked.Confirmed {
		t.Fatalf("unexpected blocked action: %#v", blocked)
	}

	entries, err := service.Audit(ctx)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("expected two new audit entries, got %d", len(entries))
	}
	if entries[0].Event != "阻止执行：公开分享合同扫描件" {
		t.Fatalf("expected block audit first, got %#v", entries[0])
	}
	if entries[1].Event != "确认执行：Agent 申请批量重命名照片" {
		t.Fatalf("expected confirm audit second, got %#v", entries[1])
	}
}

func TestServiceDeletesShareAndAddsAudit(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	share, err := service.DeleteShare(ctx, "s2", "分享链接安全检查")
	if err != nil {
		t.Fatalf("delete share: %v", err)
	}
	if share.Active {
		t.Fatalf("expected share to be revoked: %#v", share)
	}

	shares, err := service.Shares(ctx)
	if err != nil {
		t.Fatalf("list shares: %v", err)
	}
	if shares[1].Active {
		t.Fatalf("share list was not updated: %#v", shares[1])
	}

	entries, err := service.Audit(ctx)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(entries) != 4 || entries[0].Event != "撤销公开链接：合同扫描件外链" || entries[0].Risk != audit.RiskHigh {
		t.Fatalf("expected share revoke audit entry, got %#v", entries[0])
	}
}

func TestServiceRollsBackAuditEntry(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	entry, err := service.RollbackAudit(ctx, "a1", "Hiveton")
	if err != nil {
		t.Fatalf("rollback audit: %v", err)
	}
	if !entry.Reverted {
		t.Fatalf("expected entry to be marked reverted: %#v", entry)
	}

	entries, err := service.Audit(ctx)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	found := false
	for _, candidate := range entries {
		if candidate.ID == "a1" {
			found = true
			if !candidate.Reverted {
				t.Fatalf("expected listed audit entry to be reverted: %#v", candidate)
			}
		}
	}
	if !found {
		t.Fatal("expected rolled back audit entry to remain listed")
	}
}

func boolPtr(value bool) *bool {
	return &value
}
