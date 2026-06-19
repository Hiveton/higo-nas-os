package platform

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	AppName      string
	Environment  string
	Version      string
	HTTPAddr     string
	PublicOrigin string
	StateDir     string
	NASRoot      string
	StaticDir    string
	Ready        bool

	// DatabaseURL is the Postgres (pgvector) DSN powering the AI index/search
	// backbone. Empty disables DB-backed AI features (the app degrades to the
	// JSON-state stubs), so Mac dev keeps working without a database.
	DatabaseURL string

	// MCPEnabled controls whether the embedded Model Context Protocol endpoint
	// (/mcp) is mounted on the API server. Defaults to true in dev/test.
	MCPEnabled bool
	// MCPDomains is an optional comma-separated allowlist of tool domains
	// (e.g. "files,storage,docker"). Empty means expose every domain.
	MCPDomains string

	// RemoteProbeAddr is an optional host:port the remote-access channel dials
	// on start to measure real reachability and latency. Empty (the default)
	// keeps the dev demo's representative estimate; real deployments set it to
	// their relay/public endpoint.
	RemoteProbeAddr string

	// --- User center / authentication --------------------------------------

	// AuthRequired enforces a valid HiGoOS session on guarded routes. When
	// false (the dev default) the session guard injects an implicit admin
	// principal so the desktop keeps working without logging in. Production
	// must set this true (HIGO_AUTH_REQUIRED=true).
	AuthRequired bool
	// SessionTTL is how long an issued session cookie stays valid.
	SessionTTL time.Duration
	// CookieSameSite controls the SameSite attribute of the session cookie:
	// "lax" (default), "strict", or "none" (forces Secure, for cross-site
	// remote access behind HTTPS).
	CookieSameSite string
	// CSRFDisabled turns off double-submit CSRF validation. Defaults true in
	// dev (convenience), false otherwise.
	CSRFDisabled bool
	// AccountsBackend selects the accounts directory: "system" (real Linux
	// users via useradd/chpasswd), "devstub" (JSON store), or "" (auto: system
	// on Linux, devstub elsewhere).
	AccountsBackend string
	// AccountsUIDBase is the lowest UID the system backend will create/manage,
	// keeping HiGoOS-managed users out of the system account range.
	AccountsUIDBase int
	// AccountsGroup is the primary group HiGoOS-managed users belong to.
	AccountsGroup string
	// AccountsAdminGroup is the group whose membership maps to the admin role.
	AccountsAdminGroup string
	// LoginMaxFailures locks an account after this many consecutive failed
	// password attempts. Zero disables lockout.
	LoginMaxFailures int
	// AdminBootstrapPassword optionally fixes the initial admin password
	// (automation/imaging). Empty generates a random one printed once at boot.
	AdminBootstrapPassword string
}

func LoadConfig() Config {
	env := getenv("HIGO_ENV", "dev")
	return Config{
		AppName:      getenv("HIGO_APP_NAME", "HiGoOS"),
		Environment:  env,
		Version:      getenv("HIGO_VERSION", "dev"),
		HTTPAddr:     getenv("HIGO_HTTP_ADDR", ":8080"),
		PublicOrigin: getenv("HIGO_PUBLIC_ORIGIN", "http://localhost:5173"),
		StateDir:     getenv("HIGO_STATE_DIR", defaultStateDir()),
		NASRoot:      getenv("HIGO_NAS_ROOT", ""),
		StaticDir:    getenv("HIGO_STATIC_DIR", ""),
		DatabaseURL:  getenv("HIGO_DATABASE_URL", ""),
		Ready:        true,
		MCPEnabled:      getenvBool("HIGO_MCP_ENABLED", true),
		MCPDomains:      getenv("HIGO_MCP_DOMAINS", ""),
		RemoteProbeAddr: getenv("HIGO_REMOTE_PROBE_ADDR", ""),

		AuthRequired:           getenvBool("HIGO_AUTH_REQUIRED", env != "dev" && env != "test"),
		SessionTTL:             getenvDuration("HIGO_SESSION_TTL", 720*time.Hour),
		CookieSameSite:         getenv("HIGO_COOKIE_SAMESITE", "lax"),
		CSRFDisabled:           getenvBool("HIGO_CSRF_DISABLED", env == "dev" || env == "test"),
		AccountsBackend:        getenv("HIGO_ACCOUNTS_BACKEND", ""),
		AccountsUIDBase:        getenvInt("HIGO_ACCOUNTS_UID_BASE", 3000),
		AccountsGroup:          getenv("HIGO_ACCOUNTS_GROUP", "higoos"),
		AccountsAdminGroup:     getenv("HIGO_ACCOUNTS_ADMIN_GROUP", "higoos-admins"),
		LoginMaxFailures:       getenvInt("HIGO_LOGIN_MAX_FAILURES", 5),
		AdminBootstrapPassword: getenv("HIGO_ADMIN_BOOTSTRAP_PASSWORD", ""),
	}
}

func (c Config) WithDefaults() Config {
	if c.AppName == "" {
		c.AppName = "HiGoOS"
	}
	if c.Environment == "" {
		c.Environment = "dev"
	}
	if c.Version == "" {
		c.Version = "dev"
	}
	if c.HTTPAddr == "" {
		c.HTTPAddr = ":8080"
	}
	if c.PublicOrigin == "" {
		c.PublicOrigin = "http://localhost:5173"
	}
	if c.SessionTTL <= 0 {
		c.SessionTTL = 720 * time.Hour
	}
	if c.CookieSameSite == "" {
		c.CookieSameSite = "lax"
	}
	if c.AccountsGroup == "" {
		c.AccountsGroup = "higoos"
	}
	if c.AccountsAdminGroup == "" {
		c.AccountsAdminGroup = "higoos-admins"
	}
	if c.AccountsUIDBase == 0 {
		c.AccountsUIDBase = 3000
	}
	return c
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	switch value {
	case "1", "true", "TRUE", "True", "yes", "on":
		return true
	case "0", "false", "FALSE", "False", "no", "off":
		return false
	default:
		return fallback
	}
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func defaultStateDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "higoos", "state")
}
