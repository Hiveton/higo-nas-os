package accounts

import (
	"context"
	"fmt"
	"testing"
)

// TestYescryptVerify checks the yescrypt verification path against the
// openwall/yescrypt-go reference vector (Ubuntu 24.04's default shadow scheme).
func TestYescryptVerify(t *testing.T) {
	hash := "$y$j9T$AAt9R641xPvCI9nXw1HHW/$cuQRBMN3N/f8IcmVN.4YrZ1bHMOiLOoz9/XQMKV/v0A"
	ok, err := verifyCryptHash(hash, "openwall")
	if err != nil || !ok {
		t.Fatalf("expected yescrypt vector to verify, ok=%v err=%v", ok, err)
	}
	bad, err := verifyCryptHash(hash, "wrong")
	if err != nil || bad {
		t.Fatalf("expected wrong password to fail, bad=%v err=%v", bad, err)
	}
}

// fakeHostWithPasswd builds a HostDirectory whose runner serves a synthetic
// getent passwd/group database, so List/Lookup can be tested off-host.
func fakeHostWithPasswd(t *testing.T) *HostDirectory {
	t.Helper()
	passwd := "root:x:0:0:root:/root:/bin/bash\n" +
		"daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin\n" +
		"hiveton:x:1000:1000:Hiveton Owner:/home/hiveton:/bin/bash\n" +
		"admin:x:3000:1001:管理员:/home/admin:/usr/sbin/nologin\n" +
		"nobody:x:65534:65534:nobody:/nonexistent:/usr/sbin/nologin\n"
	d := newHostDirectory(directoryConfig{Group: "higoos", AdminGroup: "higoos-admins", UIDBase: 3000})
	d.runner = func(ctx context.Context, stdin string, name string, args ...string) ([]byte, error) {
		if name == "getent" && len(args) >= 1 && args[0] == "passwd" {
			if len(args) == 1 {
				return []byte(passwd), nil
			}
			// getent passwd <user>
			for _, line := range []string{
				"hiveton:x:1000:1000:Hiveton Owner:/home/hiveton:/bin/bash",
				"admin:x:3000:1001:管理员:/home/admin:/usr/sbin/nologin",
			} {
				if line[:len(args[1])] == args[1] {
					return []byte(line + "\n"), nil
				}
			}
			return nil, fmt.Errorf("not found")
		}
		if name == "getent" && len(args) == 2 && args[0] == "group" {
			switch args[1] {
			case "sudo":
				return []byte("sudo:x:27:hiveton\n"), nil
			case "higoos-admins":
				return []byte("higoos-admins:x:1002:admin\n"), nil
			}
			return nil, fmt.Errorf("no group")
		}
		return nil, nil
	}
	return d
}

func TestHostDirectoryListEnumeratesSystemUsers(t *testing.T) {
	d := fakeHostWithPasswd(t)
	ids, err := d.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	byName := map[string]SystemIdentity{}
	for _, id := range ids {
		byName[id.Username] = id
	}
	// Daemons, root and nobody are excluded; human users are included.
	if _, ok := byName["root"]; ok {
		t.Fatal("root must be excluded")
	}
	if _, ok := byName["daemon"]; ok {
		t.Fatal("daemon must be excluded")
	}
	if _, ok := byName["nobody"]; ok {
		t.Fatal("nobody must be excluded")
	}
	hiveton, ok := byName["hiveton"]
	if !ok {
		t.Fatal("hiveton (uid 1000) must be listed")
	}
	if !hiveton.Admin {
		t.Fatal("hiveton is in sudo → should be admin")
	}
	if hiveton.DisplayName != "Hiveton Owner" {
		t.Fatalf("expected GECOS display name, got %q", hiveton.DisplayName)
	}
	if admin := byName["admin"]; !admin.Admin {
		t.Fatal("admin is in higoos-admins → should be admin")
	}
}

func TestReconcileMaterializesSystemUsers(t *testing.T) {
	svc := NewService()
	svc.dir = fakeHostWithPasswd(t) // swap dev dir for the fake host
	svc.reconcileSystemUsers(context.Background())

	summary, _ := svc.Summary(context.Background())
	found := map[string]User{}
	for _, u := range summary.Users {
		found[u.Username] = u
	}
	if _, ok := found["hiveton"]; !ok {
		t.Fatal("hiveton should be reconciled into the user list")
	}
	if found["hiveton"].Role != RoleAdmin {
		t.Fatalf("hiveton (sudo) should map to admin role, got %q", found["hiveton"].Role)
	}
}
