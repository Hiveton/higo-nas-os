package accounts

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GehirnInc/crypt/sha512_crypt"

	"higoos/server-go/internal/state"
)

// DevDirectory is the non-Linux identity backend. It keeps SHA-512 crypt
// credentials in memory and, when a state dir is configured, persists them to
// credentials.json (atomic write, same as the rest of HiGoOS state). It never
// touches host users, so it is safe on developer machines.
type DevDirectory struct {
	mu        sync.RWMutex
	creds     map[string]credentialEntry // username -> credential
	locked    map[string]bool
	statePath string
}

type credentialEntry struct {
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	Locked       bool   `json:"locked"`
}

type credentialsSnapshot struct {
	Credentials []credentialEntry `json:"credentials"`
}

func newDevDirectory(stateDir string) (*DevDirectory, error) {
	d := &DevDirectory{
		creds:  make(map[string]credentialEntry),
		locked: make(map[string]bool),
	}
	if strings.TrimSpace(stateDir) == "" {
		return d, nil
	}
	d.statePath = filepath.Join(stateDir, "credentials.json")
	var snap credentialsSnapshot
	if err := state.LoadJSON(d.statePath, &snap); err != nil {
		return nil, err
	}
	for _, c := range snap.Credentials {
		d.creds[strings.ToLower(c.Username)] = c
		d.locked[strings.ToLower(c.Username)] = c.Locked
	}
	return d, nil
}

func (d *DevDirectory) EnsureUser(ctx context.Context, ref IdentityRef) error {
	// Dev identities exist purely as credential records; nothing to provision.
	return ctx.Err()
}

func (d *DevDirectory) RemoveUser(ctx context.Context, username string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	key := strings.ToLower(username)
	delete(d.creds, key)
	delete(d.locked, key)
	return d.saveLocked()
}

func (d *DevDirectory) SetPassword(ctx context.Context, username, plaintext string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	hash, err := generateCryptHash(plaintext)
	if err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	key := strings.ToLower(username)
	entry := d.creds[key]
	entry.Username = username
	entry.PasswordHash = hash
	d.creds[key] = entry
	return d.saveLocked()
}

func (d *DevDirectory) VerifyPassword(ctx context.Context, username, plaintext string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	d.mu.RLock()
	entry, ok := d.creds[strings.ToLower(username)]
	locked := d.locked[strings.ToLower(username)]
	d.mu.RUnlock()
	if !ok || locked {
		return false, nil
	}
	return verifyCryptHash(entry.PasswordHash, plaintext)
}

func (d *DevDirectory) SetLocked(ctx context.Context, username string, locked bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	key := strings.ToLower(username)
	d.locked[key] = locked
	if entry, ok := d.creds[key]; ok {
		entry.Locked = locked
		d.creds[key] = entry
	}
	return d.saveLocked()
}

// List returns no OS identities — the dev backend is sidecar-authoritative, so
// the accounts service keeps using its own user list on non-Linux hosts.
func (d *DevDirectory) List(ctx context.Context) ([]SystemIdentity, error) {
	return nil, ctx.Err()
}

// Lookup never resolves OS identities on the dev backend.
func (d *DevDirectory) Lookup(ctx context.Context, username string) (SystemIdentity, bool, error) {
	return SystemIdentity{}, false, ctx.Err()
}

// EnsureGroup / SetGroupMembers are no-ops on the dev backend.
func (d *DevDirectory) EnsureGroup(ctx context.Context, _ string) error { return ctx.Err() }
func (d *DevDirectory) SetGroupMembers(ctx context.Context, _ string, _ []string) error {
	return ctx.Err()
}

func (d *DevDirectory) SambaPresent(ctx context.Context) (map[string]bool, error) {
	return map[string]bool{}, ctx.Err()
}

func (d *DevDirectory) HasCredential(ctx context.Context, username string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	entry, ok := d.creds[strings.ToLower(username)]
	return ok && entry.PasswordHash != ""
}

func (d *DevDirectory) saveLocked() error {
	if d.statePath == "" {
		return nil
	}
	snap := credentialsSnapshot{Credentials: make([]credentialEntry, 0, len(d.creds))}
	for _, c := range d.creds {
		snap.Credentials = append(snap.Credentials, c)
	}
	return state.SaveJSON(d.statePath, snap)
}

// generateCryptHash produces a SHA-512 crypt ($6$) hash with a random salt —
// the same scheme HiGoOS provisions on Linux, so verification is uniform.
func generateCryptHash(plaintext string) (string, error) {
	salt := randomSalt(16)
	magic := fmt.Sprintf("$6$%s", salt)
	hash, err := sha512_crypt.New().Generate([]byte(plaintext), []byte(magic))
	if err != nil {
		return "", fmt.Errorf("accounts: hash password: %w", err)
	}
	return hash, nil
}

func randomSalt(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		// rand.Read essentially never fails; fall back to a fixed-length string.
		return strings.Repeat("0", n)
	}
	enc := base64.RawStdEncoding.EncodeToString(buf)
	enc = strings.NewReplacer("+", ".", "/", ".").Replace(enc)
	if len(enc) > n {
		enc = enc[:n]
	}
	return enc
}
