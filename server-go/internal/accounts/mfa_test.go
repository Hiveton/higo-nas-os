package accounts

import (
	"context"
	"testing"
	"time"
)

// TestTOTPKnownVector checks against the RFC 6238 SHA-1 test vector: the base32
// secret "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" (ASCII "12345678901234567890") at
// Unix time 59 (counter 1) yields 287082.
func TestTOTPKnownVector(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if !verifyTOTP(secret, "287082", time.Unix(59, 0)) {
		t.Fatal("expected RFC 6238 vector 287082 to verify at t=59")
	}
	if verifyTOTP(secret, "000000", time.Unix(59, 0)) {
		t.Fatal("expected a wrong code to fail")
	}
}

func TestMFAEnrollAndVerify(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	uid := "admin"

	if svc.MFAEnabled(uid) {
		t.Fatal("MFA should start disabled")
	}
	secret, uri, err := svc.MFASetup(ctx, uid, "admin")
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if secret == "" || uri == "" {
		t.Fatal("expected secret and otpauth uri")
	}
	// Not enabled until confirmed.
	if svc.MFAEnabled(uid) {
		t.Fatal("MFA must not be enabled before confirmation")
	}
	code := hotp(decodeForTest(t, secret), uint64(time.Now().Unix()/30))
	if err := svc.MFAEnable(ctx, uid, code); err != nil {
		t.Fatalf("enable: %v", err)
	}
	if !svc.MFAEnabled(uid) {
		t.Fatal("MFA should be enabled")
	}
	if !svc.MFAVerify(uid, code) {
		t.Fatal("expected valid code to verify")
	}
	if svc.MFAVerify(uid, "111111") {
		t.Fatal("expected wrong code to fail")
	}
	if err := svc.MFADisable(ctx, uid); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if svc.MFAEnabled(uid) {
		t.Fatal("MFA should be disabled")
	}
}

func decodeForTest(t *testing.T, secret string) []byte {
	t.Helper()
	key, err := totpEncoding.DecodeString(secret)
	if err != nil {
		t.Fatalf("decode secret: %v", err)
	}
	return key
}
