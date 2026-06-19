package accounts

import (
	"context"
	"crypto/subtle"
	"fmt"
	"runtime"
	"strings"

	"github.com/GehirnInc/crypt"
	// Register the crypt algorithms used to verify HiGoOS-managed credentials.
	// HiGoOS always provisions passwords as SHA-512 crypt ($6$), but we register
	// the legacy schemes too so pre-existing shadow hashes can be read.
	_ "github.com/GehirnInc/crypt/md5_crypt"
	_ "github.com/GehirnInc/crypt/sha256_crypt"
	_ "github.com/GehirnInc/crypt/sha512_crypt"
	yescrypt "github.com/openwall/yescrypt-go"
)

// IdentityRef is the host-facing projection of a HiGoOS user. The directory
// mirrors this to the operating system (real system users on Linux) — only the
// fields the OS understands. App-level metadata (role/quota/grants) stays in the
// accounts service sidecar.
type IdentityRef struct {
	Username    string
	DisplayName string
	// Admin reflects whether the user should belong to the admin system group.
	Admin bool
}

// SystemIdentity is a real OS account the directory can enumerate. HiGoOS
// reconciles these into its user list so pre-existing system users (e.g. the
// installer-created sudo user) appear and can authenticate — the OS is the
// authoritative source of identities, not just HiGoOS-created ones.
type SystemIdentity struct {
	Username    string
	UID         int
	DisplayName string
	Admin       bool // member of the admin/sudo/wheel group
}

// Directory is the identity + credential backend. HostDirectory drives real
// Linux system users (useradd/usermod/userdel/chpasswd) and authenticates
// against /etc/shadow; DevDirectory keeps an in-memory/JSON store so Mac
// development and unit tests work without touching the host.
//
// The accounts Service remains the source of truth for app-level metadata and
// calls the directory to keep the host identity store in sync and to own the
// (never-stored-in-plaintext) credentials.
type Directory interface {
	// EnsureUser creates or updates the host identity for ref.
	EnsureUser(ctx context.Context, ref IdentityRef) error
	// RemoveUser deletes the host identity (and home, on Linux).
	RemoveUser(ctx context.Context, username string) error
	// SetPassword sets the credential for username.
	SetPassword(ctx context.Context, username, plaintext string) error
	// VerifyPassword reports whether plaintext matches the stored credential.
	VerifyPassword(ctx context.Context, username, plaintext string) (bool, error)
	// SetLocked locks or unlocks the account (maps to usermod -L/-U).
	SetLocked(ctx context.Context, username string, locked bool) error
	// HasCredential reports whether a credential exists for username.
	HasCredential(ctx context.Context, username string) bool
	// List enumerates real OS login accounts (empty on the dev backend, which
	// is sidecar-authoritative).
	List(ctx context.Context) ([]SystemIdentity, error)
	// Lookup resolves a single OS account by username.
	Lookup(ctx context.Context, username string) (SystemIdentity, bool, error)
	// EnsureGroup creates the named system group if absent.
	EnsureGroup(ctx context.Context, name string) error
	// SetGroupMembers replaces a system group's membership.
	SetGroupMembers(ctx context.Context, name string, usernames []string) error
}

// directoryConfig carries the host-tuning knobs the system directory needs.
type directoryConfig struct {
	Backend    string // "system", "devstub", or "" (auto)
	StateDir   string
	UIDBase    int
	Group      string
	AdminGroup string
}

// newDirectory selects the directory backend. "system" (or auto on Linux) uses
// real system users; everything else uses the dev store.
func newDirectory(cfg directoryConfig) (Directory, error) {
	backend := cfg.Backend
	if backend == "" {
		if runtime.GOOS == "linux" {
			backend = "system"
		} else {
			backend = "devstub"
		}
	}
	switch backend {
	case "system":
		return newHostDirectory(cfg), nil
	case "devstub":
		return newDevDirectory(cfg.StateDir)
	default:
		return nil, fmt.Errorf("accounts: unknown backend %q", backend)
	}
}

// verifyCryptHash reports whether plaintext matches a crypt(3)-style hash
// (e.g. "$6$salt$..."). Used by both directories so the verification path is
// identical on Linux and dev.
func verifyCryptHash(hash, plaintext string) (bool, error) {
	if hash == "" {
		return false, nil
	}
	// yescrypt ($y$ / $gy$) is Ubuntu 24.04's default shadow scheme — verify it
	// by recomputing the hash from the stored setting and comparing.
	if strings.HasPrefix(hash, "$y$") || strings.HasPrefix(hash, "$gy$") {
		computed, err := yescrypt.Hash([]byte(plaintext), []byte(hash))
		if err != nil {
			return false, fmt.Errorf("accounts: yescrypt verify: %w", err)
		}
		return subtle.ConstantTimeCompare(computed, []byte(hash)) == 1, nil
	}
	// The remaining crypt(3) schemes are handled by GehirnInc/crypt. Reject
	// anything else up front so crypt.NewFromHash — which panics on unknown
	// prefixes — is never reached.
	switch {
	case strings.HasPrefix(hash, "$1$"),
		strings.HasPrefix(hash, "$5$"),
		strings.HasPrefix(hash, "$6$"):
	default:
		return false, fmt.Errorf("accounts: unsupported password hash scheme")
	}
	crypter := crypt.NewFromHash(hash)
	if err := crypter.Verify(hash, []byte(plaintext)); err != nil {
		if err == crypt.ErrKeyMismatch {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
