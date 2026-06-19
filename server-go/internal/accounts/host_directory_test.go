package accounts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recordedCall struct {
	stdin string
	name  string
	args  []string
}

// fakeHost builds a HostDirectory wired to a scripted runner so the Linux
// shell-out paths can be exercised on any OS.
func fakeHost(t *testing.T, existing map[string]bool, calls *[]recordedCall) *HostDirectory {
	t.Helper()
	d := newHostDirectory(directoryConfig{Group: "higoos", AdminGroup: "higoos-admins", UIDBase: 3000})
	d.runner = func(ctx context.Context, stdin string, name string, args ...string) ([]byte, error) {
		*calls = append(*calls, recordedCall{stdin: stdin, name: name, args: args})
		switch name {
		case "getent":
			if len(args) >= 2 && args[0] == "passwd" && len(args) == 2 {
				if existing[args[1]] {
					return []byte(args[1] + ":x:3000:3000::/home/" + args[1] + ":/usr/sbin/nologin\n"), nil
				}
				return nil, fmt.Errorf("not found")
			}
			if len(args) >= 2 && args[0] == "group" {
				return nil, fmt.Errorf("no group")
			}
			if len(args) == 1 && args[0] == "passwd" {
				return []byte("root:x:0:0::/root:/bin/bash\n"), nil
			}
		}
		return nil, nil
	}
	return d
}

func TestHostDirectoryEnsureUserProvisionsSystemAccount(t *testing.T) {
	var calls []recordedCall
	d := fakeHost(t, map[string]bool{}, &calls)
	if err := d.EnsureUser(context.Background(), IdentityRef{Username: "lin", DisplayName: "Lin", Admin: false}); err != nil {
		t.Fatalf("ensure user: %v", err)
	}
	var sawGroupadd, sawUseradd, sawNologin, sawPrimaryGroup bool
	for _, c := range calls {
		if c.name == "groupadd" {
			sawGroupadd = true
		}
		if c.name == "useradd" {
			sawUseradd = true
			joined := strings.Join(c.args, " ")
			sawNologin = strings.Contains(joined, "nologin")
			sawPrimaryGroup = strings.Contains(joined, "higoos")
		}
	}
	if !sawGroupadd || !sawUseradd || !sawNologin || !sawPrimaryGroup {
		t.Fatalf("expected groupadd+useradd with nologin shell and higoos group; calls=%+v", calls)
	}
}

func TestHostDirectorySetPasswordForcesSHA512(t *testing.T) {
	var calls []recordedCall
	d := fakeHost(t, map[string]bool{"lin": true}, &calls)
	if err := d.SetPassword(context.Background(), "lin", "Passw0rd1"); err != nil {
		t.Fatalf("set password: %v", err)
	}
	found := false
	for _, c := range calls {
		if c.name == "chpasswd" {
			found = true
			if strings.Join(c.args, " ") != "-c SHA512" {
				t.Fatalf("expected chpasswd -c SHA512, got %v", c.args)
			}
			if c.stdin != "lin:Passw0rd1\n" {
				t.Fatalf("unexpected chpasswd stdin: %q", c.stdin)
			}
		}
	}
	if !found {
		t.Fatalf("expected chpasswd call; calls=%+v", calls)
	}
}

func TestHostDirectoryVerifyPasswordAgainstShadow(t *testing.T) {
	var calls []recordedCall
	d := fakeHost(t, map[string]bool{"lin": true}, &calls)

	hash, err := generateCryptHash("Passw0rd1")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	dir := t.TempDir()
	shadow := filepath.Join(dir, "shadow")
	content := "root:!:19000::::::\nlin:" + hash + ":19000:0:99999:7:::\n"
	if err := os.WriteFile(shadow, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	d.shadowPath = shadow

	ok, err := d.VerifyPassword(context.Background(), "lin", "Passw0rd1")
	if err != nil || !ok {
		t.Fatalf("expected valid password, ok=%v err=%v", ok, err)
	}
	bad, err := d.VerifyPassword(context.Background(), "lin", "wrong")
	if err != nil || bad {
		t.Fatalf("expected wrong password rejected, bad=%v err=%v", bad, err)
	}
	// A locked (!-prefixed) hash never verifies.
	missing, _ := d.VerifyPassword(context.Background(), "root", "anything")
	if missing {
		t.Fatal("locked account must not verify")
	}
}

func TestHostDirectorySetLocked(t *testing.T) {
	var calls []recordedCall
	d := fakeHost(t, map[string]bool{"lin": true}, &calls)
	if err := d.SetLocked(context.Background(), "lin", true); err != nil {
		t.Fatalf("lock: %v", err)
	}
	last := calls[len(calls)-1]
	if last.name != "usermod" || last.args[0] != "-L" {
		t.Fatalf("expected usermod -L, got %+v", last)
	}
}
