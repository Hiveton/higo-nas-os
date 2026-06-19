// Package bindings owns the account<->device relationship and the three ways an
// App establishes it: claim a NAS-published pairing code/QR (flow A), confirm a
// LAN-discovered device (flow B), or enter a serial + on-screen PIN (flow C).
//
// All three converge on bind(): create the binding, ask the NAS to provision a
// local user for the cloud account (via the Provisioner seam — relay-backed in
// production, a deterministic stub in dev), and mint a device access token the App
// uses to reach that NAS.
package bindings

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"time"

	"higoos/server-cloud/internal/auth"
	"higoos/server-cloud/internal/devices"
	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/store"
)

var (
	ErrChallengeInvalid = errors.New("pairing challenge invalid or expired")
	ErrDeviceUnknown    = errors.New("device not found")
	ErrNotBound         = errors.New("device is not bound to this account")
	ErrAlreadyBound     = errors.New("device is already bound to this account")
)

// Provisioner asks a NAS to create/link a local user for a cloud account and to
// tear it down on unbind. The relay-backed implementation forwards these calls to
// server-go over the agent tunnel; the dev stub returns a synthetic id.
type Provisioner interface {
	LinkCloudAccount(ctx context.Context, deviceID, cloudUserID, role string) (localUserID string, err error)
	Unlink(ctx context.Context, deviceID, cloudUserID string) error
}

// StubProvisioner is the dev default: it fabricates a deterministic local user id
// without contacting a NAS, so the binding flow is exercisable offline.
type StubProvisioner struct{}

func (StubProvisioner) LinkCloudAccount(_ context.Context, _, cloudUserID, _ string) (string, error) {
	return "luid-" + cloudUserID, nil
}
func (StubProvisioner) Unlink(_ context.Context, _, _ string) error { return nil }

// Service implements the binding flows.
type Service struct {
	store       store.Store
	devices     *devices.Service
	tokens      *auth.Manager
	provisioner Provisioner
	pairingTTL  time.Duration
}

func NewService(s store.Store, d *devices.Service, tokens *auth.Manager, p Provisioner, pairingTTL time.Duration) *Service {
	if p == nil {
		p = StubProvisioner{}
	}
	return &Service{store: s, devices: d, tokens: tokens, provisioner: p, pairingTTL: pairingTTL}
}

// Result is returned by every successful bind: the binding plus a ready-to-use
// device access token.
type Result struct {
	Binding      store.Binding `json:"binding"`
	Device       DeviceInfo    `json:"device"`
	DeviceToken  string        `json:"deviceToken"`
	TokenExpires time.Time     `json:"tokenExpires"`
}

// DeviceInfo is the safe device projection returned to the App.
type DeviceInfo struct {
	ID         string    `json:"id"`
	Serial     string    `json:"serial"`
	Model      string    `json:"model"`
	Version    string    `json:"version"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	Online     bool      `json:"online"`
}

// --- Pairing publish (called by the NAS, device-authenticated) ---------------

// IssuePairing registers a short-lived challenge a NAS publishes on its screen or
// web desktop. kind is "code" (flow A) or "pin" (flow C). Returns the secret the
// NAS displays.
func (s *Service) IssuePairing(ctx context.Context, deviceID string, kind store.PairingKind) (secret string, expiresAt time.Time, err error) {
	if _, err := s.devices.Get(ctx, deviceID); err != nil {
		return "", time.Time{}, ErrDeviceUnknown
	}
	secret = sixDigits()
	expiresAt = time.Now().Add(s.pairingTTL)
	if err := s.store.PutPairingChallenge(ctx, store.PairingChallenge{
		Secret:    secret,
		DeviceID:  deviceID,
		Kind:      kind,
		ExpiresAt: expiresAt,
	}); err != nil {
		return "", time.Time{}, err
	}
	return secret, expiresAt, nil
}

// --- Flow A: claim a pairing code -------------------------------------------

// ClaimPairing binds the device behind a pairing code to the cloud account.
func (s *Service) ClaimPairing(ctx context.Context, userID, code string) (Result, error) {
	ch, err := s.consumeChallenge(ctx, store.PairingCode, code)
	if err != nil {
		return Result{}, err
	}
	return s.bind(ctx, userID, ch.DeviceID)
}

// --- Flow B: confirm a LAN-discovered device --------------------------------

// LANConfirm binds a device the App discovered on the local network. The device
// must already be registered with the cloud.
func (s *Service) LANConfirm(ctx context.Context, userID, deviceID string) (Result, error) {
	if _, err := s.devices.Get(ctx, deviceID); err != nil {
		return Result{}, ErrDeviceUnknown
	}
	return s.bind(ctx, userID, deviceID)
}

// --- Flow C: serial + PIN ----------------------------------------------------

// BindSerial binds a device identified by its printed serial, gated by the
// on-screen PIN the NAS published.
func (s *Service) BindSerial(ctx context.Context, userID, serial, pin string) (Result, error) {
	device, err := s.store.GetDeviceBySerial(ctx, serial)
	if err != nil {
		return Result{}, ErrDeviceUnknown
	}
	ch, err := s.consumeChallenge(ctx, store.PairingPIN, pin)
	if err != nil {
		return Result{}, err
	}
	if ch.DeviceID != device.ID {
		return Result{}, ErrChallengeInvalid
	}
	return s.bind(ctx, userID, device.ID)
}

// --- Listing / unbind / access ticket ---------------------------------------

// List returns the devices bound to an account.
func (s *Service) List(ctx context.Context, userID string) ([]Result, error) {
	bs, err := s.store.ListBindingsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Result, 0, len(bs))
	for _, b := range bs {
		d, err := s.devices.Get(ctx, b.DeviceID)
		if err != nil {
			continue
		}
		out = append(out, Result{Binding: b, Device: s.deviceInfo(d)})
	}
	return out, nil
}

// Unbind removes a binding and asks the NAS to drop the local mapping.
func (s *Service) Unbind(ctx context.Context, userID, deviceID string) error {
	if _, err := s.store.GetBinding(ctx, userID, deviceID); err != nil {
		return ErrNotBound
	}
	_ = s.provisioner.Unlink(ctx, deviceID, userID)
	return s.store.DeleteBinding(ctx, userID, deviceID)
}

// IssueAccessTicket mints a fresh device access token for a bound device. The App
// calls this whenever its short-lived token is near expiry.
func (s *Service) IssueAccessTicket(ctx context.Context, userID, deviceID string) (Result, error) {
	b, err := s.store.GetBinding(ctx, userID, deviceID)
	if err != nil {
		return Result{}, ErrNotBound
	}
	d, err := s.devices.Get(ctx, deviceID)
	if err != nil {
		return Result{}, ErrDeviceUnknown
	}
	tok, exp, err := s.tokens.SignDeviceToken(userID, deviceID, b.LocalUserID, b.Role)
	if err != nil {
		return Result{}, err
	}
	return Result{Binding: b, Device: s.deviceInfo(d), DeviceToken: tok, TokenExpires: exp}, nil
}

// --- internals ---------------------------------------------------------------

// bind is the shared tail of all three flows.
func (s *Service) bind(ctx context.Context, userID, deviceID string) (Result, error) {
	if _, err := s.store.GetBinding(ctx, userID, deviceID); err == nil {
		return Result{}, ErrAlreadyBound
	}
	device, err := s.devices.Get(ctx, deviceID)
	if err != nil {
		return Result{}, ErrDeviceUnknown
	}

	// First account to bind a device becomes its admin; later accounts are users.
	role := "user"
	if existing, _ := s.store.ListBindingsByDevice(ctx, deviceID); len(existing) == 0 {
		role = "admin"
	}

	localUserID, err := s.provisioner.LinkCloudAccount(ctx, deviceID, userID, role)
	if err != nil {
		return Result{}, err
	}

	binding := store.Binding{
		ID:          platform.NewID("bind"),
		UserID:      userID,
		DeviceID:    deviceID,
		Role:        role,
		LocalUserID: localUserID,
		Label:       device.Model,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.store.CreateBinding(ctx, binding); err != nil {
		return Result{}, err
	}
	_ = s.store.AppendAudit(ctx, store.AuditEntry{
		ID:        platform.NewID("aud"),
		UserID:    userID,
		DeviceID:  deviceID,
		Action:    "device.bind",
		Detail:    string(role),
		CreatedAt: time.Now().UTC(),
	})

	tok, exp, err := s.tokens.SignDeviceToken(userID, deviceID, localUserID, role)
	if err != nil {
		return Result{}, err
	}
	return Result{Binding: binding, Device: s.deviceInfo(device), DeviceToken: tok, TokenExpires: exp}, nil
}

// consumeChallenge validates a pairing challenge and marks it claimed (single use).
func (s *Service) consumeChallenge(ctx context.Context, kind store.PairingKind, secret string) (store.PairingChallenge, error) {
	ch, err := s.store.GetPairingChallenge(ctx, kind, secret)
	if err != nil {
		return store.PairingChallenge{}, ErrChallengeInvalid
	}
	if ch.Claimed || time.Now().After(ch.ExpiresAt) {
		return store.PairingChallenge{}, ErrChallengeInvalid
	}
	if err := s.store.MarkPairingClaimed(ctx, kind, secret); err != nil {
		return store.PairingChallenge{}, err
	}
	return ch, nil
}

func (s *Service) deviceInfo(d store.Device) DeviceInfo {
	return DeviceInfo{
		ID:         d.ID,
		Serial:     d.Serial,
		Model:      d.Model,
		Version:    d.Version,
		LastSeenAt: d.LastSeenAt,
		Online:     time.Since(d.LastSeenAt) < 90*time.Second,
	}
}

func sixDigits() string {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "000000"
	}
	s := n.String()
	for len(s) < 6 {
		s = "0" + s
	}
	return s
}
