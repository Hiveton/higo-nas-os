package sharedfolders

import (
	"context"
	"path/filepath"
	"testing"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/protocols"
	"higoos/server-go/internal/storage"
)

type fakeStorage struct{ spaces []storage.StorageSpace }

func (f fakeStorage) Spaces(context.Context) ([]storage.StorageSpace, error) {
	return f.spaces, nil
}

func newTestService(t *testing.T) (*Service, *accounts.Service, *protocols.Service, string, storage.StorageSpace) {
	t.Helper()
	nasRoot := t.TempDir()
	space := storage.StorageSpace{ID: "sp1", Name: "主存储池", MountPath: filepath.Join(nasRoot, "pool")}
	acc := accounts.NewService()
	proto := protocols.NewService(nil) // dev adapter on macOS — host writes are no-ops
	svc := NewService(Deps{
		Accounts:   acc,
		Storage:    fakeStorage{spaces: []storage.StorageSpace{space}},
		Protocols:  proto,
		NASRoot:    nasRoot,
		ServerHost: "10.211.55.3",
	})
	return svc, acc, proto, nasRoot, space
}

func TestSharedFolderUnifiesPermissionsAndSmb(t *testing.T) {
	ctx := context.Background()
	svc, acc, proto, _, space := newTestService(t)

	user, err := acc.CreateUser(ctx, accounts.CreateUserRequest{Username: "hiveton", DisplayName: "Hiveton", Password: "123qwe"})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// Create a shared folder at the space root.
	view, err := svc.Create(ctx, CreateRequest{SpaceID: space.ID, Name: "hiveton-share", Actor: "admin"})
	if err != nil {
		t.Fatalf("create folder: %v", err)
	}
	if view.DirKey != "pool" || view.AbsPath != filepath.Join(space.MountPath) {
		t.Fatalf("unexpected folder view: %#v", view)
	}

	// Grant hiveton read-write via the permission table.
	view, err = svc.SetPermissions(ctx, view.ID, SetPermissionsRequest{
		Entries: []PermissionEntry{{SubjectType: accounts.SubjectUser, SubjectID: user.ID, Access: AccessReadWrite}},
	})
	if err != nil {
		t.Fatalf("set permissions: %v", err)
	}
	if len(view.Permissions) != 1 || view.Permissions[0].SubjectID != user.ID || view.Permissions[0].Access != AccessReadWrite {
		t.Fatalf("permission table not reflected: %#v", view.Permissions)
	}
	// The accounts grant must target the folder DirKey (so setfacl ran on it).
	summary, _ := acc.Summary(ctx)
	foundGrant := false
	for _, g := range summary.Grants {
		if g.SpaceID == "pool" && g.SubjectID == user.ID && g.Access == accounts.AccessReadWrite {
			foundGrant = true
		}
	}
	if !foundGrant {
		t.Fatalf("expected a read_write grant on dirKey 'pool': %#v", summary.Grants)
	}

	// Enabling SMB derives a share whose valid users / write list come from the table.
	view, err = svc.SetService(ctx, view.ID, SetServiceRequest{Protocol: "smb", Enabled: true})
	if err != nil {
		t.Fatalf("set service: %v", err)
	}
	shares, _ := proto.Shares(ctx, protocols.ProtocolSMB)
	var derived *protocols.Share
	for i := range shares {
		if filepath.Clean(shares[i].Path) == filepath.Clean(view.AbsPath) {
			derived = &shares[i]
		}
	}
	if derived == nil {
		t.Fatalf("expected a derived SMB share at %s; shares=%#v", view.AbsPath, shares)
	}
	if !contains(derived.AllowedUsers, "hiveton") || !contains(derived.WriteUsers, "hiveton") {
		t.Fatalf("derived share should grant hiveton write: %#v", derived)
	}

	// Demote to read-only → hiveton stays in valid users but leaves write list.
	if _, err := svc.SetPermissions(ctx, view.ID, SetPermissionsRequest{
		Entries: []PermissionEntry{{SubjectType: accounts.SubjectUser, SubjectID: user.ID, Access: AccessRead}},
	}); err != nil {
		t.Fatalf("demote: %v", err)
	}
	shares, _ = proto.Shares(ctx, protocols.ProtocolSMB)
	for i := range shares {
		if filepath.Clean(shares[i].Path) == filepath.Clean(view.AbsPath) {
			if !contains(shares[i].AllowedUsers, "hiveton") || contains(shares[i].WriteUsers, "hiveton") {
				t.Fatalf("after demote hiveton should be read-only: %#v", shares[i])
			}
		}
	}

	// Delete the folder → grants and shares are torn down.
	preview, err := svc.PreviewDelete(ctx, view.ID, "admin")
	if err != nil {
		t.Fatalf("preview delete: %v", err)
	}
	if err := svc.ConfirmDelete(ctx, ConfirmDeleteRequest{ConfirmationID: preview.ConfirmationID}); err != nil {
		t.Fatalf("confirm delete: %v", err)
	}
	list, _ := svc.List(ctx)
	if len(list) != 0 {
		t.Fatalf("folder should be gone: %#v", list)
	}
	summary, _ = acc.Summary(ctx)
	for _, g := range summary.Grants {
		if g.SpaceID == "pool" {
			t.Fatalf("grants should be cleared on delete: %#v", g)
		}
	}
	shares, _ = proto.Shares(ctx, protocols.ProtocolSMB)
	for _, sh := range shares {
		if filepath.Clean(sh.Path) == filepath.Clean(view.AbsPath) {
			t.Fatalf("derived share should be removed on delete: %#v", sh)
		}
	}
}

func contains(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}
