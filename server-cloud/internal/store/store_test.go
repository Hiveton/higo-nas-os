package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/store"
)

// runSuite exercises the core Store contract. Both Memory and Postgres run it, so
// the two backends are guaranteed to behave identically.
func runSuite(t *testing.T, s store.Store) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)

	uid := platform.NewID("cu")
	if err := s.CreateUser(ctx, store.User{ID: uid, DisplayName: "Tester", Status: "active", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if _, err := s.GetUser(ctx, uid); err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if _, err := s.GetUser(ctx, "missing"); err != store.ErrNotFound {
		t.Fatalf("GetUser(missing) = %v, want ErrNotFound", err)
	}

	// Identity uniqueness on (type, principal).
	id := store.Identity{ID: platform.NewID("ci"), UserID: uid, Type: store.IdentityPhone, Principal: uid + "-13800000000", CreatedAt: now}
	if err := s.CreateIdentity(ctx, id); err != nil {
		t.Fatalf("CreateIdentity: %v", err)
	}
	if err := s.CreateIdentity(ctx, store.Identity{ID: platform.NewID("ci"), UserID: uid, Type: store.IdentityPhone, Principal: id.Principal, CreatedAt: now}); err == nil {
		t.Fatal("duplicate identity should conflict")
	}
	if got, err := s.GetIdentity(ctx, store.IdentityPhone, id.Principal); err != nil || got.UserID != uid {
		t.Fatalf("GetIdentity: %v / %+v", err, got)
	}

	// Refresh token rotate/revoke.
	rt := store.RefreshToken{ID: platform.NewID("rt"), TokenHash: platform.NewID("h"), UserID: uid, ExpiresAt: now.Add(time.Hour), CreatedAt: now}
	if err := s.CreateRefreshToken(ctx, rt); err != nil {
		t.Fatalf("CreateRefreshToken: %v", err)
	}
	if err := s.RevokeRefreshToken(ctx, rt.ID, now); err != nil {
		t.Fatalf("RevokeRefreshToken: %v", err)
	}
	got, _ := s.GetRefreshToken(ctx, rt.TokenHash)
	if got.RevokedAt == nil {
		t.Fatal("refresh token should be revoked")
	}

	// Device + binding + pairing.
	did := platform.NewID("hg")
	dev := store.Device{ID: did, Serial: "S-" + did[:8], SecretHash: "h", Model: "X", CreatedAt: now, LastSeenAt: now}
	if err := s.CreateDevice(ctx, dev); err != nil {
		t.Fatalf("CreateDevice: %v", err)
	}
	if _, err := s.GetDeviceBySerial(ctx, dev.Serial); err != nil {
		t.Fatalf("GetDeviceBySerial: %v", err)
	}

	ch := store.PairingChallenge{Secret: "123456-" + did[:6], DeviceID: did, Kind: store.PairingCode, ExpiresAt: now.Add(time.Minute)}
	if err := s.PutPairingChallenge(ctx, ch); err != nil {
		t.Fatalf("PutPairingChallenge: %v", err)
	}
	if err := s.MarkPairingClaimed(ctx, store.PairingCode, ch.Secret); err != nil {
		t.Fatalf("MarkPairingClaimed: %v", err)
	}
	if got, _ := s.GetPairingChallenge(ctx, store.PairingCode, ch.Secret); !got.Claimed {
		t.Fatal("challenge should be claimed")
	}

	b := store.Binding{ID: platform.NewID("bind"), UserID: uid, DeviceID: did, Role: "admin", LocalUserID: "luid", CreatedAt: now}
	if err := s.CreateBinding(ctx, b); err != nil {
		t.Fatalf("CreateBinding: %v", err)
	}
	if err := s.CreateBinding(ctx, b); err == nil {
		t.Fatal("duplicate binding should conflict")
	}
	if list, _ := s.ListBindingsByUser(ctx, uid); len(list) != 1 {
		t.Fatalf("ListBindingsByUser = %d, want 1", len(list))
	}
	if list, _ := s.ListBindingsByDevice(ctx, did); len(list) != 1 {
		t.Fatalf("ListBindingsByDevice = %d, want 1", len(list))
	}
	if err := s.DeleteBinding(ctx, uid, did); err != nil {
		t.Fatalf("DeleteBinding: %v", err)
	}
	if err := s.DeleteBinding(ctx, uid, did); err != store.ErrNotFound {
		t.Fatalf("DeleteBinding(again) = %v, want ErrNotFound", err)
	}

	// Push + audit.
	if err := s.UpsertPushToken(ctx, store.PushToken{ID: platform.NewID("pt"), UserID: uid, Token: "apns-tok", Platform: "ios", CreatedAt: now}); err != nil {
		t.Fatalf("UpsertPushToken: %v", err)
	}
	if err := s.UpsertPushToken(ctx, store.PushToken{ID: platform.NewID("pt"), UserID: uid, Token: "apns-tok", Platform: "ios", CreatedAt: now}); err != nil {
		t.Fatalf("UpsertPushToken(again): %v", err) // upsert, not conflict
	}
	if list, _ := s.ListPushTokensByUser(ctx, uid); len(list) != 1 {
		t.Fatalf("push tokens = %d, want 1", len(list))
	}
	if err := s.AppendAudit(ctx, store.AuditEntry{ID: platform.NewID("aud"), UserID: uid, Action: "device.bind", CreatedAt: now}); err != nil {
		t.Fatalf("AppendAudit: %v", err)
	}
	if list, _ := s.ListAuditByUser(ctx, uid, 10); len(list) != 1 {
		t.Fatalf("audit = %d, want 1", len(list))
	}
}

func TestMemory(t *testing.T) {
	runSuite(t, store.NewMemory())
}

func TestPostgres(t *testing.T) {
	dsn := os.Getenv("HIGO_CLOUD_TEST_DSN")
	if dsn == "" {
		t.Skip("set HIGO_CLOUD_TEST_DSN to run the Postgres conformance suite")
	}
	ctx := context.Background()
	pg, err := store.ConnectPostgres(ctx, dsn)
	if err != nil {
		t.Fatalf("ConnectPostgres: %v", err)
	}
	defer pg.Close()
	runSuite(t, pg)
}
