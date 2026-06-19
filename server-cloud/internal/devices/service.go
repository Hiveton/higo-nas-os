// Package devices is the NAS registry. A NAS registers once (getting a durable
// secret + human-readable serial), then authenticates with that secret for its
// relay connection, heartbeats and pairing publishes.
package devices

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"higoos/server-cloud/internal/store"
)

// ErrUnauthorized is returned when a device secret does not match.
var ErrUnauthorized = errors.New("device authentication failed")

// Service manages device records.
type Service struct {
	store store.Store
}

func NewService(s store.Store) *Service { return &Service{store: s} }

// Registration is the one-time response a NAS persists after registering.
type Registration struct {
	Device       store.Device `json:"device"`
	DeviceSecret string       `json:"deviceSecret"` // returned once, only the hash is stored
	Serial       string       `json:"serial"`
}

// Register creates (or re-creates the secret for) a device. deviceID is the NAS's
// stable identity.DeviceID ("hg-xxxx"). Re-registration with a known id rotates
// the secret and refreshes metadata — the common case when a NAS reinstalls.
func (s *Service) Register(ctx context.Context, deviceID, model, version, agentPubKey string) (Registration, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return Registration{}, errors.New("deviceId is required")
	}
	secret, hash, err := newSecret()
	if err != nil {
		return Registration{}, err
	}
	now := time.Now().UTC()

	existing, err := s.store.GetDevice(ctx, deviceID)
	if err == nil {
		existing.SecretHash = hash
		existing.Model = model
		existing.Version = version
		existing.AgentPublicKey = agentPubKey
		existing.LastSeenAt = now
		if err := s.store.UpdateDevice(ctx, existing); err != nil {
			return Registration{}, err
		}
		return Registration{Device: existing, DeviceSecret: secret, Serial: existing.Serial}, nil
	}

	device := store.Device{
		ID:             deviceID,
		Serial:         newSerial(),
		SecretHash:     hash,
		Model:          firstNonEmpty(model, "HiGoOS NAS"),
		Version:        firstNonEmpty(version, "dev"),
		AgentPublicKey: agentPubKey,
		CreatedAt:      now,
		LastSeenAt:     now,
	}
	if err := s.store.CreateDevice(ctx, device); err != nil {
		return Registration{}, err
	}
	return Registration{Device: device, DeviceSecret: secret, Serial: device.Serial}, nil
}

// Authenticate verifies a device secret and returns the device. Used by the relay
// agent endpoint and device-authenticated control calls (pairing publish).
func (s *Service) Authenticate(ctx context.Context, deviceID, secret string) (store.Device, error) {
	d, err := s.store.GetDevice(ctx, deviceID)
	if err != nil {
		return store.Device{}, ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(d.SecretHash), []byte(secret)) != nil {
		return store.Device{}, ErrUnauthorized
	}
	return d, nil
}

// Heartbeat refreshes LastSeenAt for an authenticated device.
func (s *Service) Heartbeat(ctx context.Context, deviceID string) error {
	d, err := s.store.GetDevice(ctx, deviceID)
	if err != nil {
		return err
	}
	d.LastSeenAt = time.Now().UTC()
	return s.store.UpdateDevice(ctx, d)
}

// Get returns a device by id.
func (s *Service) Get(ctx context.Context, deviceID string) (store.Device, error) {
	return s.store.GetDevice(ctx, deviceID)
}

func newSecret() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	h, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", "", err
	}
	return raw, string(h), nil
}

// newSerial returns a typeable code like "HG-7F3A-2B9C".
func newSerial() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "HG-0000-0000"
	}
	s := strings.ToUpper(hex.EncodeToString(b))
	return "HG-" + s[:4] + "-" + s[4:]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
