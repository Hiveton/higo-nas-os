package platform

import (
	"os"
	"strconv"
	"time"
)

// Config is the resolved server-cloud configuration. Every field has a
// HIGO_CLOUD_* environment variable so the binary is twelve-factor and the dev
// defaults let it boot with zero setup (memory store, dev SMS/email/oauth stubs).
type Config struct {
	AppName      string
	Environment  string // dev | test | production
	Version      string
	HTTPAddr     string
	PublicOrigin string // CORS allow-origin; "*" allows any

	// Store selects the persistence backend: "memory" (default in dev) or
	// "postgres" (requires DatabaseURL).
	Store       string
	DatabaseURL string

	// AuthRequired enforces a valid cloud access token on guarded routes. False
	// (dev default) injects an implicit test principal so the API is pokeable
	// with curl. Production must set HIGO_CLOUD_AUTH_REQUIRED=true.
	AuthRequired bool

	// JWTSecret signs cloud access tokens (HS256). Generated per-boot in dev if
	// empty; production must set a stable secret so tokens survive restarts.
	JWTSecret      string
	AccessTokenTTL time.Duration
	RefreshTokenTTL time.Duration

	// DeviceTokenTTL bounds the short-lived device-access tokens the cloud signs
	// (Ed25519) for the App to call a bound NAS.
	DeviceTokenTTL time.Duration
	// DeviceTokenSeedHex optionally fixes the Ed25519 signing seed (64 hex
	// chars) so the cloud public key — distributed to every NAS at registration
	// — is stable across restarts. Empty generates an ephemeral key in dev.
	DeviceTokenSeedHex string

	// PairingTTL bounds how long a NAS pairing code / PIN stays claimable.
	PairingTTL time.Duration
	// VerificationTTL bounds how long an SMS/email verification code is valid.
	VerificationTTL time.Duration
	// VerificationDebug echoes issued codes in the API response (dev only).
	VerificationDebug bool

	// External providers. Empty values keep the dev stubs active.
	AppleClientID  string // Sign in with Apple aud / bundle id
	WeChatAppID    string
	WeChatSecret   string
	APNsBundleID   string
	APNsKeyID      string
	APNsTeamID     string
	APNsKeyPath    string // path to the .p8 token-auth key
}

func LoadConfig() Config {
	env := getenv("HIGO_CLOUD_ENV", "dev")
	dev := env == "dev" || env == "test"
	return Config{
		AppName:      getenv("HIGO_CLOUD_APP_NAME", "HiGoOS Cloud"),
		Environment:  env,
		Version:      getenv("HIGO_CLOUD_VERSION", "dev"),
		HTTPAddr:     getenv("HIGO_CLOUD_HTTP_ADDR", ":8090"),
		PublicOrigin: getenv("HIGO_CLOUD_PUBLIC_ORIGIN", "*"),

		Store:       getenv("HIGO_CLOUD_STORE", "memory"),
		DatabaseURL: getenv("HIGO_CLOUD_DB_DSN", ""),

		AuthRequired: getenvBool("HIGO_CLOUD_AUTH_REQUIRED", !dev),

		JWTSecret:       getenv("HIGO_CLOUD_JWT_SECRET", ""),
		AccessTokenTTL:  getenvDuration("HIGO_CLOUD_ACCESS_TTL", 15*time.Minute),
		RefreshTokenTTL: getenvDuration("HIGO_CLOUD_REFRESH_TTL", 60*24*time.Hour),

		DeviceTokenTTL:     getenvDuration("HIGO_CLOUD_DEVICE_TOKEN_TTL", 10*time.Minute),
		DeviceTokenSeedHex: getenv("HIGO_CLOUD_DEVICE_TOKEN_SEED", ""),

		PairingTTL:        getenvDuration("HIGO_CLOUD_PAIRING_TTL", 5*time.Minute),
		VerificationTTL:   getenvDuration("HIGO_CLOUD_VERIFICATION_TTL", 5*time.Minute),
		VerificationDebug: getenvBool("HIGO_CLOUD_VERIFICATION_DEBUG", dev),

		AppleClientID: getenv("HIGO_CLOUD_APPLE_CLIENT_ID", "com.hiveton.higoos"),
		WeChatAppID:   getenv("HIGO_CLOUD_WECHAT_APPID", ""),
		WeChatSecret:  getenv("HIGO_CLOUD_WECHAT_SECRET", ""),
		APNsBundleID:  getenv("HIGO_CLOUD_APNS_BUNDLE_ID", "com.hiveton.higoos"),
		APNsKeyID:     getenv("HIGO_CLOUD_APNS_KEY_ID", ""),
		APNsTeamID:    getenv("HIGO_CLOUD_APNS_TEAM_ID", ""),
		APNsKeyPath:   getenv("HIGO_CLOUD_APNS_KEY_PATH", ""),
	}
}

// WithDefaults backfills empty fields so a zero-ish Config (e.g. from a test)
// still boots sensibly.
func (c Config) WithDefaults() Config {
	if c.AppName == "" {
		c.AppName = "HiGoOS Cloud"
	}
	if c.Environment == "" {
		c.Environment = "dev"
	}
	if c.Version == "" {
		c.Version = "dev"
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = ":8090"
	}
	if c.PublicOrigin == "" {
		c.PublicOrigin = "*"
	}
	if c.Store == "" {
		c.Store = "memory"
	}
	if c.AccessTokenTTL <= 0 {
		c.AccessTokenTTL = 15 * time.Minute
	}
	if c.RefreshTokenTTL <= 0 {
		c.RefreshTokenTTL = 60 * 24 * time.Hour
	}
	if c.DeviceTokenTTL <= 0 {
		c.DeviceTokenTTL = 10 * time.Minute
	}
	if c.PairingTTL <= 0 {
		c.PairingTTL = 5 * time.Minute
	}
	if c.VerificationTTL <= 0 {
		c.VerificationTTL = 5 * time.Minute
	}
	if c.AppleClientID == "" {
		c.AppleClientID = "com.hiveton.higoos"
	}
	return c
}

// IsDev reports whether convenience fallbacks (implicit principal, debug codes)
// are permitted.
func (c Config) IsDev() bool {
	return c.Environment == "dev" || c.Environment == "test"
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	switch os.Getenv(key) {
	case "":
		return fallback
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	case "0", "false", "FALSE", "False", "no", "off":
		return false
	default:
		return fallback
	}
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return fallback
}

var _ = getenvInt // reserved for future numeric knobs
