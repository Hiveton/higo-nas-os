// Package store is server-cloud's persistence boundary. Every cloud entity
// (users, login identities, refresh tokens, devices, account<->device bindings,
// pairing challenges, push tokens, audit) lives here behind the Store interface.
//
// Two implementations ship: Memory (the dev default — zero dependencies, boots
// instantly) and, behind the same interface, a Postgres backend for production.
// Keeping the entity structs in this package avoids an import cycle: domain
// services depend on store, never the reverse.
package store

import (
	"context"
	"errors"
	"time"
)

// ErrNotFound is returned by every lookup that misses.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a uniqueness invariant would be violated (e.g.
// registering a login identity that already belongs to another account).
var ErrConflict = errors.New("conflict")

// IdentityType enumerates the login methods a cloud account can carry.
type IdentityType string

const (
	IdentityPhone  IdentityType = "phone"
	IdentityEmail  IdentityType = "email"
	IdentityApple  IdentityType = "apple"
	IdentityWeChat IdentityType = "wechat"
)

// User is a HiGoOS cloud account. It carries no credentials directly — those
// live on its Identity rows, so one account can sign in by phone, email, Apple
// and WeChat interchangeably.
type User struct {
	ID          string    `json:"id"`
	DisplayName string    `json:"displayName"`
	Avatar      string    `json:"avatar,omitempty"`
	Status      string    `json:"status"` // active | disabled
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Identity is one login method bound to a User. Principal is the natural key for
// the method: phone number, email address, Apple `sub`, or WeChat `unionid`.
// SecretHash is only populated for the email+password method.
type Identity struct {
	ID         string       `json:"id"`
	UserID     string       `json:"userId"`
	Type       IdentityType `json:"type"`
	Principal  string       `json:"principal"`
	SecretHash string       `json:"-"`
	VerifiedAt *time.Time   `json:"verifiedAt,omitempty"`
	CreatedAt  time.Time    `json:"createdAt"`
}

// RefreshToken is a long-lived, revocable, rotating credential. Only the SHA-256
// of the opaque token is stored, never the token itself.
type RefreshToken struct {
	ID        string     `json:"id"`
	TokenHash string     `json:"-"`
	UserID    string     `json:"userId"`
	ExpiresAt time.Time  `json:"expiresAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

// Device is a registered NAS. SecretHash authenticates the agent's relay
// connection and control-plane calls; AgentPublicKey lets the cloud verify
// payloads the NAS signs. Serial is the human-readable code printed on the box.
type Device struct {
	ID             string    `json:"id"` // hg-xxxx, matches server-go identity.DeviceID
	Serial         string    `json:"serial"`
	SecretHash     string    `json:"-"`
	Model          string    `json:"model"`
	Version        string    `json:"version"`
	AgentPublicKey string    `json:"agentPublicKey,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	LastSeenAt     time.Time `json:"lastSeenAt"`
}

// Binding links a cloud account to a NAS. LocalUserID is the NAS-local user the
// device provisioned for this account (returned by server-go), Role is the
// cloud-side role granted at bind time.
type Binding struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId"`
	DeviceID    string    `json:"deviceId"`
	Role        string    `json:"role"` // admin | user
	LocalUserID string    `json:"localUserId,omitempty"`
	Label       string    `json:"label,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// PairingKind distinguishes the two NAS-issued challenge shapes.
type PairingKind string

const (
	PairingCode PairingKind = "code" // 6-digit / QR claim (flow A)
	PairingPIN  PairingKind = "pin"  // serial + on-screen PIN (flow C)
)

// PairingChallenge is a short-lived, single-use claim a NAS publishes so the App
// can prove it is operating that specific device.
type PairingChallenge struct {
	Secret    string      `json:"-"` // the code/pin the App must present
	DeviceID  string      `json:"deviceId"`
	Kind      PairingKind `json:"kind"`
	ExpiresAt time.Time   `json:"expiresAt"`
	Claimed   bool        `json:"claimed"`
}

// PushToken is an APNs device token registered by the App for a cloud account.
type PushToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Token     string    `json:"token"`
	Platform  string    `json:"platform"` // ios
	CreatedAt time.Time `json:"createdAt"`
}

// AuditEntry records a security-relevant cloud event (login, bind, unbind).
type AuditEntry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId,omitempty"`
	DeviceID  string    `json:"deviceId,omitempty"`
	Action    string    `json:"action"`
	Detail    string    `json:"detail,omitempty"`
	IP        string    `json:"ip,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// Store is the full persistence surface. All methods are context-aware so the
// Postgres backend can honour cancellation/timeouts; Memory ignores ctx.
type Store interface {
	// Users & identities -----------------------------------------------------
	CreateUser(ctx context.Context, u User) error
	GetUser(ctx context.Context, id string) (User, error)
	UpdateUser(ctx context.Context, u User) error

	CreateIdentity(ctx context.Context, i Identity) error
	GetIdentity(ctx context.Context, t IdentityType, principal string) (Identity, error)
	ListIdentitiesByUser(ctx context.Context, userID string) ([]Identity, error)
	UpdateIdentity(ctx context.Context, i Identity) error

	// Refresh tokens ---------------------------------------------------------
	CreateRefreshToken(ctx context.Context, t RefreshToken) error
	GetRefreshToken(ctx context.Context, tokenHash string) (RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string, at time.Time) error
	RevokeUserRefreshTokens(ctx context.Context, userID string, at time.Time) error

	// Devices ----------------------------------------------------------------
	CreateDevice(ctx context.Context, d Device) error
	GetDevice(ctx context.Context, id string) (Device, error)
	GetDeviceBySerial(ctx context.Context, serial string) (Device, error)
	UpdateDevice(ctx context.Context, d Device) error

	// Bindings ---------------------------------------------------------------
	CreateBinding(ctx context.Context, b Binding) error
	GetBinding(ctx context.Context, userID, deviceID string) (Binding, error)
	ListBindingsByUser(ctx context.Context, userID string) ([]Binding, error)
	ListBindingsByDevice(ctx context.Context, deviceID string) ([]Binding, error)
	DeleteBinding(ctx context.Context, userID, deviceID string) error

	// Pairing challenges -----------------------------------------------------
	PutPairingChallenge(ctx context.Context, c PairingChallenge) error
	GetPairingChallenge(ctx context.Context, kind PairingKind, secret string) (PairingChallenge, error)
	MarkPairingClaimed(ctx context.Context, kind PairingKind, secret string) error

	// Push tokens ------------------------------------------------------------
	UpsertPushToken(ctx context.Context, t PushToken) error
	ListPushTokensByUser(ctx context.Context, userID string) ([]PushToken, error)

	// Audit ------------------------------------------------------------------
	AppendAudit(ctx context.Context, e AuditEntry) error
	ListAuditByUser(ctx context.Context, userID string, limit int) ([]AuditEntry, error)
}
