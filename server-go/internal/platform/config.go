package platform

import (
	"os"
	"path/filepath"
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
}

func LoadConfig() Config {
	return Config{
		AppName:      getenv("HIGO_APP_NAME", "HiGoOS"),
		Environment:  getenv("HIGO_ENV", "dev"),
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

func defaultStateDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "higoos", "state")
}
