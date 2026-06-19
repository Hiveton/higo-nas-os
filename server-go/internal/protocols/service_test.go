package protocols

import (
	"context"
	"testing"

	"higoos/server-go/internal/audit"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewServiceWithStateDir(NewDevAdapter(), t.TempDir())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func TestDefaultServiceSelectsDevAdapterOffLinux(t *testing.T) {
	// On the dev host (darwin) the default adapter must be the devstub.
	svc := NewService(nil)
	if _, ok := svc.adapter.(*DevAdapter); !ok {
		t.Skipf("default adapter is %T (expected DevAdapter off-Linux)", svc.adapter)
	}
}

func TestPreviewPublicShareIsHighRisk(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	preview, err := svc.PreviewCreateShare(ctx, ProtocolSMB, CreateShareRequest{
		Name: "外发", Path: "/srv/外发", AccessLevel: AccessPublic, Actor: "tester",
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Risk != audit.RiskHigh || !preview.RequiresConfirmation || preview.ConfirmationID == "" {
		t.Fatalf("public share should be high-risk requiring confirmation: %#v", preview)
	}
}

func TestConfirmCreateShareAppliesAndAudits(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	preview, err := svc.PreviewCreateShare(ctx, ProtocolSMB, CreateShareRequest{
		Name: "团队盘", Path: "/srv/team", AccessLevel: AccessAccount, AllowedUsers: []string{"alice"},
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	res, err := svc.Confirm(ctx, ConfirmRequest{ConfirmationID: preview.ConfirmationID, Actor: "tester"})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if res.Share == nil || res.Share.Name != "团队盘" {
		t.Fatalf("confirm should return the created share: %#v", res)
	}
	if res.Audit.Result != audit.ResultConfirmed || res.Audit.Kind != changeShareCreate {
		t.Fatalf("audit entry wrong: %#v", res.Audit)
	}
	shares, _ := svc.Shares(ctx, ProtocolSMB)
	found := false
	for _, s := range shares {
		if s.ID == res.Share.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("created share not persisted in list")
	}
}

func TestConfirmMismatchedIDFails(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	if _, err := svc.PreviewEnable(ctx, ProtocolNFS, "tester"); err != nil {
		t.Fatalf("preview: %v", err)
	}
	if _, err := svc.Confirm(ctx, ConfirmRequest{ConfirmationID: "protocols-confirm-999"}); err == nil {
		t.Fatalf("expected error for unknown confirmation id")
	}
}

func TestEnableConfirmThenRollback(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	preview, err := svc.PreviewEnable(ctx, ProtocolNFS, "tester")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	res, err := svc.Confirm(ctx, ConfirmRequest{ConfirmationID: preview.ConfirmationID})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if res.Protocol == nil || !res.Protocol.Enabled {
		t.Fatalf("protocol should be enabled after confirm: %#v", res)
	}
	rolled, err := svc.Rollback(ctx, res.Audit.ID, RollbackRequest{Actor: "tester"})
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if !rolled.Reverted || rolled.Result != audit.ResultRolledBack {
		t.Fatalf("audit entry should be reverted: %#v", rolled)
	}
	proto, _ := svc.Get(ctx, ProtocolNFS)
	if proto.Enabled {
		t.Fatalf("protocol should be disabled after rollback: %#v", proto)
	}
}

func TestUpdateConfigAppliesAndRollsBack(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()
	updated, err := svc.UpdateConfig(ctx, ProtocolSMB, ConfigUpdateRequest{
		Config: ProtocolConfig{ServerName: "客厅NAS", Workgroup: "HOME", MinProtocol: "SMB3", GuestAccess: true},
		Actor:  "tester",
	})
	if err != nil {
		t.Fatalf("update config: %v", err)
	}
	if updated.Config.Workgroup != "HOME" || !updated.Config.GuestAccess {
		t.Fatalf("config not applied: %#v", updated.Config)
	}
	entries, _ := svc.Audit(ctx)
	var auditID string
	for _, e := range entries {
		if e.Kind == changeConfigUpdate {
			auditID = e.ID
			break
		}
	}
	if auditID == "" {
		t.Fatalf("no config-update audit entry recorded")
	}
	if _, err := svc.Rollback(ctx, auditID, RollbackRequest{Actor: "tester"}); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	p, _ := svc.Get(ctx, ProtocolSMB)
	if p.Config.Workgroup != "WORKGROUP" || p.Config.GuestAccess {
		t.Fatalf("config not restored to seed defaults after rollback: %#v", p.Config)
	}
}

func TestSharePathMustBeUnderNASRoot(t *testing.T) {
	t.Setenv("HIGO_NAS_ROOT", t.TempDir())
	svc := newTestService(t)
	ctx := context.Background()
	if _, err := svc.PreviewCreateShare(ctx, ProtocolSMB, CreateShareRequest{
		Name: "越界", Path: "/etc/passwd", AccessLevel: AccessReadOnly,
	}); err == nil {
		t.Fatalf("expected rejection of path outside NAS root")
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	svc, err := NewServiceWithStateDir(NewDevAdapter(), dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	preview, err := svc.PreviewCreateShare(ctx, ProtocolWebDAV, CreateShareRequest{Name: "网盘", Path: "/srv/dav", AccessLevel: AccessAccount})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Confirm(ctx, ConfirmRequest{ConfirmationID: preview.ConfirmationID}); err != nil {
		t.Fatal(err)
	}
	// Reconstruct from the same state dir.
	reloaded, err := NewServiceWithStateDir(NewDevAdapter(), dir)
	if err != nil {
		t.Fatal(err)
	}
	shares, _ := reloaded.Shares(ctx, ProtocolWebDAV)
	found := false
	for _, s := range shares {
		if s.Name == "网盘" {
			found = true
		}
	}
	if !found {
		t.Fatalf("share did not survive reload: %#v", shares)
	}
}
