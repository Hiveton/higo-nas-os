// Package auth owns server-cloud's cryptographic identity: the HS256 secret that
// signs cloud access tokens, the Ed25519 key pair that signs device access tokens
// (and whose public half every NAS trusts), and the helpers that mint and hash
// the opaque refresh tokens. It is pure crypto + time; persistence of refresh
// tokens lives in the account service on top of the store.
package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"higoos/server-cloud/internal/token"
)

const (
	// AudienceCloud scopes a cloud access token to the cloud control plane.
	AudienceCloud = "cloud"
	// DeviceAudiencePrefix + deviceID scopes a device access token to one NAS.
	DeviceAudiencePrefix = "device:"
)

// Manager holds the signing material and TTL policy.
type Manager struct {
	secret      []byte
	edPriv      ed25519.PrivateKey
	edPub       ed25519.PublicKey
	accessTTL   time.Duration
	refreshTTL  time.Duration
	deviceTTL   time.Duration
}

// NewManager builds a Manager. secret signs cloud access tokens (a random one is
// generated when empty — fine for dev, where tokens need not survive a restart).
// seedHex, when a valid 64-hex Ed25519 seed, yields a stable device-token key so
// the cloud public key distributed to NAS devices is durable; otherwise an
// ephemeral key pair is generated.
func NewManager(secret, seedHex string, accessTTL, refreshTTL, deviceTTL time.Duration) (*Manager, error) {
	sec := []byte(secret)
	if len(sec) == 0 {
		sec = make([]byte, 32)
		if _, err := rand.Read(sec); err != nil {
			return nil, err
		}
	}
	var priv ed25519.PrivateKey
	if seedHex != "" {
		seed, err := hex.DecodeString(seedHex)
		if err != nil || len(seed) != ed25519.SeedSize {
			return nil, errors.New("HIGO_CLOUD_DEVICE_TOKEN_SEED must be 64 hex chars")
		}
		priv = ed25519.NewKeyFromSeed(seed)
	} else {
		var err error
		_, priv, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
	}
	return &Manager{
		secret:     sec,
		edPriv:     priv,
		edPub:      priv.Public().(ed25519.PublicKey),
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
		deviceTTL:  deviceTTL,
	}, nil
}

// PublicKeyHex returns the Ed25519 public key the NAS uses to verify device
// access tokens. It is handed to a device at registration time.
func (m *Manager) PublicKeyHex() string {
	return hex.EncodeToString(m.edPub)
}

// SignAccess issues a cloud access token for userID. Returns the compact token
// and its expiry.
func (m *Manager) SignAccess(userID, display string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.accessTTL)
	tok, err := token.SignHS256(m.secret, token.Claims{
		Subject:   userID,
		Audience:  AudienceCloud,
		Display:   display,
		IssuedAt:  now.Unix(),
		ExpiresAt: exp.Unix(),
	})
	return tok, exp, err
}

// VerifyAccess validates a cloud access token and returns its claims.
func (m *Manager) VerifyAccess(tok string) (token.Claims, error) {
	return token.VerifyHS256(m.secret, tok, AudienceCloud)
}

// SignDeviceToken issues an Ed25519 device access token: the App presents it to a
// bound NAS, which verifies it with the cloud public key. The audience pins the
// token to one device so it cannot be replayed against another.
func (m *Manager) SignDeviceToken(cloudUserID, deviceID, localUserID, role string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.deviceTTL)
	tok, err := token.SignEd25519(m.edPriv, token.Claims{
		Subject:     cloudUserID,
		Audience:    DeviceAudiencePrefix + deviceID,
		DeviceID:    deviceID,
		LocalUserID: localUserID,
		Role:        role,
		IssuedAt:    now.Unix(),
		ExpiresAt:   exp.Unix(),
	})
	return tok, exp, err
}

// RefreshTTL exposes the configured refresh-token lifetime.
func (m *Manager) RefreshTTL() time.Duration { return m.refreshTTL }

// NewOpaqueToken returns a random opaque token (given to the client) and its
// SHA-256 hex hash (stored). Only the hash is ever persisted.
func NewOpaqueToken() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, HashToken(raw), nil
}

// HashToken returns the SHA-256 hex of a token for constant-shape storage/lookup.
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
