package auth

import (
	"fmt"
	"sync"
	"time"
)

type User struct {
	ID          string
	Username    string
	DisplayName string
	Role        string
	Disabled    bool
	CreatedAt   time.Time
}

type Session struct {
	ID        string
	UserID    string
	DeviceID  string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

type TrustedDevice struct {
	ID          string
	UserID      string
	DeviceID    string
	Label       string
	Fingerprint string
	TrustedAt   time.Time
	LastSeenAt  time.Time
	RevokedAt   *time.Time
}

type PasswordLoginInput struct {
	Username      string
	Password      string
	DeviceID      string
	DeviceLabel   string
	Fingerprint   string
	TrustDevice   bool
	SessionTTL    time.Duration
	Authenticated time.Time
}

type PasswordLoginOutput struct {
	User          User
	Session       Session
	TrustedDevice *TrustedDevice
}

type DevSessionStore struct {
	mu             sync.RWMutex
	nextSessionID  int
	nextDeviceID   int
	sessions       map[string]Session
	trustedDevices map[string]TrustedDevice
}

func NewDevSessionStore() *DevSessionStore {
	return &DevSessionStore{
		sessions:       make(map[string]Session),
		trustedDevices: make(map[string]TrustedDevice),
	}
}

func (s *DevSessionStore) CreateSession(user User, input PasswordLoginInput) (PasswordLoginOutput, error) {
	if user.ID == "" {
		return PasswordLoginOutput{}, fmt.Errorf("auth: user id is required")
	}
	if input.Username == "" {
		input.Username = user.Username
	}
	if input.Authenticated.IsZero() {
		input.Authenticated = time.Now().UTC()
	}
	if input.SessionTTL <= 0 {
		input.SessionTTL = 24 * time.Hour
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextSessionID++
	session := Session{
		ID:        fmt.Sprintf("dev-session-%d", s.nextSessionID),
		UserID:    user.ID,
		DeviceID:  input.DeviceID,
		CreatedAt: input.Authenticated,
		ExpiresAt: input.Authenticated.Add(input.SessionTTL),
	}
	s.sessions[session.ID] = session

	var trusted *TrustedDevice
	if input.TrustDevice {
		s.nextDeviceID++
		device := TrustedDevice{
			ID:          fmt.Sprintf("trusted-device-%d", s.nextDeviceID),
			UserID:      user.ID,
			DeviceID:    input.DeviceID,
			Label:       input.DeviceLabel,
			Fingerprint: input.Fingerprint,
			TrustedAt:   input.Authenticated,
			LastSeenAt:  input.Authenticated,
		}
		s.trustedDevices[device.ID] = device
		trusted = &device
	}

	return PasswordLoginOutput{
		User:          user,
		Session:       session,
		TrustedDevice: trusted,
	}, nil
}

func (s *DevSessionStore) ValidateSession(sessionID string, at time.Time) (Session, bool) {
	if at.IsZero() {
		at = time.Now().UTC()
	}

	s.mu.RLock()
	session, ok := s.sessions[sessionID]
	s.mu.RUnlock()
	if !ok {
		return Session{}, false
	}
	if session.RevokedAt != nil || !at.Before(session.ExpiresAt) {
		return Session{}, false
	}
	return session, true
}

func (s *DevSessionStore) RevokeSession(sessionID string, at time.Time) bool {
	if at.IsZero() {
		at = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.sessions[sessionID]
	if !ok || session.RevokedAt != nil {
		return false
	}
	session.RevokedAt = &at
	s.sessions[sessionID] = session
	return true
}
