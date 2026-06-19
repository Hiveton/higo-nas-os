package auth

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// SessionStore is the persistent HiGoOS web-session backend. Sessions are
// keyed by an unguessable random token (the higo_session cookie value) and
// stored as JSON (atomic write, same as the rest of HiGoOS state). Linux does
// not model web sessions, so HiGoOS owns this even when identities live in the
// system user database.
type SessionStore struct {
	mu        sync.RWMutex
	sessions  map[string]Session
	statePath string
	now       func() time.Time
	ttl       time.Duration
}

type sessionsSnapshot struct {
	Sessions []Session `json:"sessions"`
}

// NewSessionStore returns an in-memory session store (no persistence).
func NewSessionStore(ttl time.Duration) *SessionStore {
	if ttl <= 0 {
		ttl = 720 * time.Hour
	}
	return &SessionStore{
		sessions: make(map[string]Session),
		now:      func() time.Time { return time.Now().UTC() },
		ttl:      ttl,
	}
}

// NewSessionStoreWithStateDir loads (and persists) sessions under stateDir.
func NewSessionStoreWithStateDir(stateDir string, ttl time.Duration) (*SessionStore, error) {
	store := NewSessionStore(ttl)
	if strings.TrimSpace(stateDir) == "" {
		return store, nil
	}
	store.statePath = filepath.Join(stateDir, "sessions.json")
	var snap sessionsSnapshot
	if err := state.LoadJSON(store.statePath, &snap); err != nil {
		return nil, err
	}
	now := store.now()
	for _, s := range snap.Sessions {
		// Drop expired/revoked sessions on load so the file self-prunes.
		if s.RevokedAt != nil || !now.Before(s.ExpiresAt) {
			continue
		}
		store.sessions[s.ID] = s
	}
	return store, nil
}

// Issue creates and persists a new session for userID.
func (s *SessionStore) Issue(userID, deviceID, sourceIP, userAgent string) (Session, error) {
	now := s.now()
	session := Session{
		ID:         randomToken(32),
		UserID:     userID,
		DeviceID:   deviceID,
		SourceIP:   sourceIP,
		UserAgent:  userAgent,
		CSRFToken:  randomToken(32),
		CreatedAt:  now,
		ExpiresAt:  now.Add(s.ttl),
		LastSeenAt: now,
	}
	s.mu.Lock()
	s.sessions[session.ID] = session
	err := s.saveLocked()
	s.mu.Unlock()
	return session, err
}

// Validate returns the live session for id, refreshing LastSeenAt.
func (s *SessionStore) Validate(id string, at time.Time) (Session, bool) {
	if at.IsZero() {
		at = s.now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[id]
	if !ok {
		return Session{}, false
	}
	if session.RevokedAt != nil || !at.Before(session.ExpiresAt) {
		delete(s.sessions, id)
		_ = s.saveLocked()
		return Session{}, false
	}
	// Throttle persistence: only rewrite when LastSeen moved by >1 minute.
	if at.Sub(session.LastSeenAt) > time.Minute {
		session.LastSeenAt = at
		s.sessions[id] = session
		_ = s.saveLocked()
	}
	return session, true
}

// Revoke invalidates a single session.
func (s *SessionStore) Revoke(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[id]; !ok {
		return false
	}
	delete(s.sessions, id)
	_ = s.saveLocked()
	return true
}

// RevokeAllForUser invalidates every session for userID except keepID, and
// returns the count revoked. Pass keepID="" to revoke all.
func (s *SessionStore) RevokeAllForUser(userID, keepID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	for id, session := range s.sessions {
		if session.UserID == userID && id != keepID {
			delete(s.sessions, id)
			removed++
		}
	}
	if removed > 0 {
		_ = s.saveLocked()
	}
	return removed
}

// ListByUser returns a user's active sessions, newest first.
func (s *SessionStore) ListByUser(userID string) []Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Session, 0)
	now := s.now()
	for _, session := range s.sessions {
		if session.UserID != userID {
			continue
		}
		if session.RevokedAt != nil || !now.Before(session.ExpiresAt) {
			continue
		}
		out = append(out, session)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

func (s *SessionStore) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	snap := sessionsSnapshot{Sessions: make([]Session, 0, len(s.sessions))}
	for _, session := range s.sessions {
		snap.Sessions = append(snap.Sessions, session)
	}
	return state.SaveJSON(s.statePath, snap)
}

func randomToken(n int) string {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strings.Repeat("0", n*2)
	}
	return hex.EncodeToString(buf)
}
