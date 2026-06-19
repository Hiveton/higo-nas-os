package accounts

import (
	"context"
	"testing"
)

func TestEffectiveAccessAndCanWrite(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Create a non-admin user and a group.
	u, err := svc.CreateUser(ctx, CreateUserRequest{Username: "lin", Password: "Passw0rd1", Role: RoleUser})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Uncontrolled space: open to everyone.
	if !svc.CanWriteSpace(ctx, u.ID, "space-open") {
		t.Fatal("uncontrolled space should be writable")
	}

	// Controlled space with read-only grant: blocked for write.
	if _, err := svc.GrantSpace(ctx, SpaceGrantRequest{SubjectType: SubjectUser, SubjectID: u.ID, SpaceID: "space-x", Access: AccessReadOnly}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if svc.CanWriteSpace(ctx, u.ID, "space-x") {
		t.Fatal("read-only grant must not allow writes")
	}

	// Upgrade to read_write: now allowed.
	if _, err := svc.GrantSpace(ctx, SpaceGrantRequest{SubjectType: SubjectUser, SubjectID: u.ID, SpaceID: "space-x", Access: AccessReadWrite}); err != nil {
		t.Fatalf("grant rw: %v", err)
	}
	if !svc.CanWriteSpace(ctx, u.ID, "space-x") {
		t.Fatal("read_write grant should allow writes")
	}

	// A controlled space the user has no grant on: blocked.
	if _, err := svc.GrantSpace(ctx, SpaceGrantRequest{SubjectType: SubjectUser, SubjectID: "someone-else", SpaceID: "space-y", Access: AccessManage}); err != nil {
		t.Fatalf("grant other: %v", err)
	}
	if svc.CanWriteSpace(ctx, u.ID, "space-y") {
		t.Fatal("user without grant on a controlled space must be blocked")
	}

	// Admin bypasses everything.
	if !svc.CanWriteSpace(ctx, "admin", "space-y") {
		t.Fatal("admin should bypass space ACL")
	}
}
