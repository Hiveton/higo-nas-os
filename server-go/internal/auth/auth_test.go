package auth

import (
	"testing"
	"time"
)

func TestDevSessionStoreValidatesCreatedSession(t *testing.T) {
	now := time.Date(2026, 5, 6, 10, 0, 0, 0, time.UTC)
	store := NewDevSessionStore()
	user := User{ID: "user-admin", Username: "admin", DisplayName: "Admin"}

	output, err := store.CreateSession(user, PasswordLoginInput{
		Username:      "admin",
		Password:      "secret",
		DeviceID:      "device-1",
		TrustDevice:   true,
		SessionTTL:    time.Hour,
		Authenticated: now,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session, ok := store.ValidateSession(output.Session.ID, now.Add(10*time.Minute))
	if !ok {
		t.Fatal("expected created session to validate")
	}
	if session.UserID != user.ID || session.DeviceID != "device-1" {
		t.Fatalf("unexpected session: %#v", session)
	}
	if output.TrustedDevice == nil || output.TrustedDevice.UserID != user.ID {
		t.Fatalf("expected trusted device output, got %#v", output.TrustedDevice)
	}
}

func TestDevSessionStoreRejectsRevokedSession(t *testing.T) {
	now := time.Date(2026, 5, 6, 11, 0, 0, 0, time.UTC)
	store := NewDevSessionStore()

	output, err := store.CreateSession(User{ID: "user-1", Username: "member"}, PasswordLoginInput{
		Username:      "member",
		Password:      "secret",
		DeviceID:      "device-2",
		SessionTTL:    time.Hour,
		Authenticated: now,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if !store.RevokeSession(output.Session.ID, now.Add(5*time.Minute)) {
		t.Fatal("expected revoke to report success")
	}
	if _, ok := store.ValidateSession(output.Session.ID, now.Add(6*time.Minute)); ok {
		t.Fatal("expected revoked session to be rejected")
	}
}
