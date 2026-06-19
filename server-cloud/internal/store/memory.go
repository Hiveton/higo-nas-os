package store

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Memory is the zero-dependency Store used in dev/test. Everything lives in maps
// behind a single RWMutex — the same coarse-grained model server-go uses for its
// per-domain JSON state. Data is lost on restart, which is exactly what dev
// wants.
type Memory struct {
	mu sync.RWMutex

	users      map[string]User
	identities map[string]Identity // keyed by ID
	refresh    map[string]RefreshToken
	devices    map[string]Device
	bindings   map[string]Binding // keyed by userID+"|"+deviceID
	pairing    map[string]PairingChallenge
	push       map[string]PushToken // keyed by userID+"|"+token
	audit      []AuditEntry
}

// NewMemory returns an empty in-memory store.
func NewMemory() *Memory {
	return &Memory{
		users:      map[string]User{},
		identities: map[string]Identity{},
		refresh:    map[string]RefreshToken{},
		devices:    map[string]Device{},
		bindings:   map[string]Binding{},
		pairing:    map[string]PairingChallenge{},
		push:       map[string]PushToken{},
	}
}

func bindKey(userID, deviceID string) string { return userID + "|" + deviceID }
func pairKey(kind PairingKind, secret string) string {
	return string(kind) + "|" + secret
}

// --- Users & identities ------------------------------------------------------

func (m *Memory) CreateUser(_ context.Context, u User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[u.ID]; ok {
		return ErrConflict
	}
	m.users[u.ID] = u
	return nil
}

func (m *Memory) GetUser(_ context.Context, id string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (m *Memory) UpdateUser(_ context.Context, u User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[u.ID]; !ok {
		return ErrNotFound
	}
	m.users[u.ID] = u
	return nil
}

func (m *Memory) CreateIdentity(_ context.Context, i Identity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.identities {
		if existing.Type == i.Type && existing.Principal == i.Principal {
			return ErrConflict
		}
	}
	m.identities[i.ID] = i
	return nil
}

func (m *Memory) GetIdentity(_ context.Context, t IdentityType, principal string) (Identity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, i := range m.identities {
		if i.Type == t && i.Principal == principal {
			return i, nil
		}
	}
	return Identity{}, ErrNotFound
}

func (m *Memory) ListIdentitiesByUser(_ context.Context, userID string) ([]Identity, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Identity
	for _, i := range m.identities {
		if i.UserID == userID {
			out = append(out, i)
		}
	}
	sort.Slice(out, func(a, b int) bool { return out[a].CreatedAt.Before(out[b].CreatedAt) })
	return out, nil
}

func (m *Memory) UpdateIdentity(_ context.Context, i Identity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.identities[i.ID]; !ok {
		return ErrNotFound
	}
	m.identities[i.ID] = i
	return nil
}

// --- Refresh tokens ----------------------------------------------------------

func (m *Memory) CreateRefreshToken(_ context.Context, t RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refresh[t.TokenHash] = t
	return nil
}

func (m *Memory) GetRefreshToken(_ context.Context, tokenHash string) (RefreshToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.refresh[tokenHash]
	if !ok {
		return RefreshToken{}, ErrNotFound
	}
	return t, nil
}

func (m *Memory) RevokeRefreshToken(_ context.Context, id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for hash, t := range m.refresh {
		if t.ID == id {
			t.RevokedAt = &at
			m.refresh[hash] = t
			return nil
		}
	}
	return ErrNotFound
}

func (m *Memory) RevokeUserRefreshTokens(_ context.Context, userID string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for hash, t := range m.refresh {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &at
			m.refresh[hash] = t
		}
	}
	return nil
}

// --- Devices -----------------------------------------------------------------

func (m *Memory) CreateDevice(_ context.Context, d Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.devices[d.ID] = d
	return nil
}

func (m *Memory) GetDevice(_ context.Context, id string) (Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.devices[id]
	if !ok {
		return Device{}, ErrNotFound
	}
	return d, nil
}

func (m *Memory) GetDeviceBySerial(_ context.Context, serial string) (Device, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.devices {
		if d.Serial == serial {
			return d, nil
		}
	}
	return Device{}, ErrNotFound
}

func (m *Memory) UpdateDevice(_ context.Context, d Device) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.devices[d.ID]; !ok {
		return ErrNotFound
	}
	m.devices[d.ID] = d
	return nil
}

// --- Bindings ----------------------------------------------------------------

func (m *Memory) CreateBinding(_ context.Context, b Binding) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := bindKey(b.UserID, b.DeviceID)
	if _, ok := m.bindings[key]; ok {
		return ErrConflict
	}
	m.bindings[key] = b
	return nil
}

func (m *Memory) GetBinding(_ context.Context, userID, deviceID string) (Binding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.bindings[bindKey(userID, deviceID)]
	if !ok {
		return Binding{}, ErrNotFound
	}
	return b, nil
}

func (m *Memory) ListBindingsByUser(_ context.Context, userID string) ([]Binding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Binding
	for _, b := range m.bindings {
		if b.UserID == userID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(a, b2 int) bool { return out[a].CreatedAt.Before(out[b2].CreatedAt) })
	return out, nil
}

func (m *Memory) ListBindingsByDevice(_ context.Context, deviceID string) ([]Binding, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Binding
	for _, b := range m.bindings {
		if b.DeviceID == deviceID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (m *Memory) DeleteBinding(_ context.Context, userID, deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := bindKey(userID, deviceID)
	if _, ok := m.bindings[key]; !ok {
		return ErrNotFound
	}
	delete(m.bindings, key)
	return nil
}

// --- Pairing challenges ------------------------------------------------------

func (m *Memory) PutPairingChallenge(_ context.Context, c PairingChallenge) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pairing[pairKey(c.Kind, c.Secret)] = c
	return nil
}

func (m *Memory) GetPairingChallenge(_ context.Context, kind PairingKind, secret string) (PairingChallenge, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.pairing[pairKey(kind, secret)]
	if !ok {
		return PairingChallenge{}, ErrNotFound
	}
	return c, nil
}

func (m *Memory) MarkPairingClaimed(_ context.Context, kind PairingKind, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := pairKey(kind, secret)
	c, ok := m.pairing[key]
	if !ok {
		return ErrNotFound
	}
	c.Claimed = true
	m.pairing[key] = c
	return nil
}

// --- Push tokens -------------------------------------------------------------

func (m *Memory) UpsertPushToken(_ context.Context, t PushToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.push[t.UserID+"|"+t.Token] = t
	return nil
}

func (m *Memory) ListPushTokensByUser(_ context.Context, userID string) ([]PushToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []PushToken
	for _, t := range m.push {
		if t.UserID == userID {
			out = append(out, t)
		}
	}
	return out, nil
}

// --- Audit -------------------------------------------------------------------

func (m *Memory) AppendAudit(_ context.Context, e AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.audit = append(m.audit, e)
	return nil
}

func (m *Memory) ListAuditByUser(_ context.Context, userID string, limit int) ([]AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []AuditEntry
	for i := len(m.audit) - 1; i >= 0; i-- {
		if m.audit[i].UserID == userID {
			out = append(out, m.audit[i])
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

// Ensure Memory satisfies Store at compile time.
var _ Store = (*Memory)(nil)
