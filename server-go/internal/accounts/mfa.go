package accounts

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// mfaEntry holds a user's TOTP secret. Pending becomes Enabled once the user
// proves possession by entering a valid code.
type mfaEntry struct {
	UserID  string `json:"userId"`
	Secret  string `json:"secret"` // base32, no padding
	Enabled bool   `json:"enabled"`
}

type mfaSnapshot struct {
	Entries []mfaEntry `json:"entries"`
}

// mfaStore keeps per-user TOTP secrets, persisted to mfa.json.
type mfaStore struct {
	mu        sync.RWMutex
	entries   map[string]mfaEntry // userID -> entry
	statePath string
}

func newMFAStore(stateDir string) (*mfaStore, error) {
	store := &mfaStore{entries: make(map[string]mfaEntry)}
	if strings.TrimSpace(stateDir) == "" {
		return store, nil
	}
	store.statePath = filepath.Join(stateDir, "mfa.json")
	var snap mfaSnapshot
	if err := state.LoadJSON(store.statePath, &snap); err != nil {
		return nil, err
	}
	for _, e := range snap.Entries {
		store.entries[e.UserID] = e
	}
	return store, nil
}

func (m *mfaStore) saveLocked() error {
	if m.statePath == "" {
		return nil
	}
	snap := mfaSnapshot{Entries: make([]mfaEntry, 0, len(m.entries))}
	for _, e := range m.entries {
		snap.Entries = append(snap.Entries, e)
	}
	return state.SaveJSON(m.statePath, snap)
}

// MFAEnabled reports whether a user has confirmed TOTP enrollment.
func (s *Service) MFAEnabled(userID string) bool {
	s.mfa.mu.RLock()
	defer s.mfa.mu.RUnlock()
	e, ok := s.mfa.entries[userID]
	return ok && e.Enabled
}

// MFASetup generates a fresh (pending) secret for a user and returns the secret
// plus an otpauth:// provisioning URI for authenticator apps. It overwrites any
// not-yet-enabled enrollment.
func (s *Service) MFASetup(ctx context.Context, userID, username string) (secret, uri string, err error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	secret = randomBase32Secret(20)
	s.mfa.mu.Lock()
	existing := s.mfa.entries[userID]
	if existing.Enabled {
		s.mfa.mu.Unlock()
		return "", "", fmt.Errorf("mfa already enabled")
	}
	s.mfa.entries[userID] = mfaEntry{UserID: userID, Secret: secret, Enabled: false}
	err = s.mfa.saveLocked()
	s.mfa.mu.Unlock()
	if err != nil {
		return "", "", err
	}
	label := url.PathEscape("HiGoOS:" + username)
	uri = fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=HiGoOS&algorithm=SHA1&digits=6&period=30", label, secret)
	return secret, uri, nil
}

// MFAEnable confirms enrollment if code matches the pending secret.
func (s *Service) MFAEnable(ctx context.Context, userID, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mfa.mu.Lock()
	defer s.mfa.mu.Unlock()
	entry, ok := s.mfa.entries[userID]
	if !ok || entry.Secret == "" {
		return fmt.Errorf("no pending MFA enrollment")
	}
	if !verifyTOTP(entry.Secret, code, time.Now()) {
		return ErrInvalidMFACode
	}
	entry.Enabled = true
	s.mfa.entries[userID] = entry
	return s.mfa.saveLocked()
}

// MFADisable turns off TOTP for a user.
func (s *Service) MFADisable(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mfa.mu.Lock()
	defer s.mfa.mu.Unlock()
	delete(s.mfa.entries, userID)
	return s.mfa.saveLocked()
}

// MFAVerify checks a login-time TOTP code for an enrolled user.
func (s *Service) MFAVerify(userID, code string) bool {
	s.mfa.mu.RLock()
	entry, ok := s.mfa.entries[userID]
	s.mfa.mu.RUnlock()
	if !ok || !entry.Enabled {
		return false
	}
	return verifyTOTP(entry.Secret, code, time.Now())
}

// --- TOTP (RFC 6238) -------------------------------------------------------

var totpEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func randomBase32Secret(nBytes int) string {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		for i := range buf {
			buf[i] = byte(i)
		}
	}
	return totpEncoding.EncodeToString(buf)
}

// verifyTOTP validates a 6-digit code against the secret with a ±1 step window
// (30s period) to tolerate clock skew.
func verifyTOTP(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	key, err := totpEncoding.DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return false
	}
	counter := uint64(now.Unix() / 30)
	for _, delta := range []int64{0, -1, 1} {
		if hotp(key, uint64(int64(counter)+delta)) == code {
			return true
		}
	}
	return false
}

func hotp(key []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (uint32(sum[offset]&0x7f) << 24) |
		(uint32(sum[offset+1]) << 16) |
		(uint32(sum[offset+2]) << 8) |
		uint32(sum[offset+3])
	return fmt.Sprintf("%06d", value%1_000_000)
}
