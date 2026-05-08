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
	Ready        bool
}

func LoadConfig() Config {
	return Config{
		AppName:      getenv("HIGO_APP_NAME", "HiGoOS"),
		Environment:  getenv("HIGO_ENV", "dev"),
		Version:      getenv("HIGO_VERSION", "dev"),
		HTTPAddr:     getenv("HIGO_HTTP_ADDR", ":8080"),
		PublicOrigin: getenv("HIGO_PUBLIC_ORIGIN", "http://localhost:5173"),
		StateDir:     getenv("HIGO_STATE_DIR", defaultStateDir()),
		Ready:        true,
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

func defaultStateDir() string {
	base, err := os.UserCacheDir()
	if err != nil || base == "" {
		base = os.TempDir()
	}
	return filepath.Join(base, "higoos", "state")
}
