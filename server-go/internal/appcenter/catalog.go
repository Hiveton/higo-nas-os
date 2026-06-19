package appcenter

import (
	"embed"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

//go:embed seed/*.json
var seedFS embed.FS

// Origin labels where a catalog entry came from. Same-id precedence is
// local > remote > builtin so an operator or registry can override a seed app.
const (
	OriginBuiltin = "builtin"
	OriginLocal   = "local"
	OriginRemote  = "remote"
)

// CatalogEntry is one installable app as published by a catalog source. It is
// decoupled from install/runtime state — that lives in the service's App list.
type CatalogEntry struct {
	Manifest AppManifest `json:"manifest"`
	Origin   string      `json:"origin"`
	Registry string      `json:"registry,omitempty"`
}

// Registry is a remote source of manifests, addressed by an index URL.
type Registry struct {
	Name    string    `json:"name"`
	URL     string    `json:"url"`
	Enabled bool      `json:"enabled"`
	AddedAt time.Time `json:"addedAt"`
}

// Catalog aggregates the installable app manifests from three sources:
//   - builtin: embedded seed manifests (offline, always available)
//   - local:   $STATE/appcenter/apps/<id>/manifest.json (drop a package in)
//   - remote:  registries fetched over HTTP (see catalog_remote.go)
//
// It is the modular replacement for the previous hardcoded Go seed data: adding
// an app no longer requires editing or recompiling the control plane.
type Catalog struct {
	mu         sync.RWMutex
	builtin    map[string]CatalogEntry
	local      map[string]CatalogEntry
	remote     map[string]CatalogEntry
	localDir   string
	statePath  string
	registries []Registry
	httpClient *http.Client
	logger     *slog.Logger
}

// NewCatalog builds a catalog rooted at stateDir. The builtin seed always
// loads; local packages and registry config load when a state dir is set.
func NewCatalog(stateDir string, logger *slog.Logger) *Catalog {
	if logger == nil {
		logger = slog.Default()
	}
	c := &Catalog{
		builtin:    map[string]CatalogEntry{},
		local:      map[string]CatalogEntry{},
		remote:     map[string]CatalogEntry{},
		httpClient: &http.Client{Timeout: 15 * time.Second},
		logger:     logger,
	}
	c.loadBuiltin()
	if stateDir != "" {
		c.localDir = filepath.Join(stateDir, "appcenter", "apps")
		c.statePath = filepath.Join(stateDir, "appcenter", "registries.json")
		c.ScanLocal()
		c.loadRegistries()
	}
	return c
}

// loadBuiltin parses every embedded seed manifest.
func (c *Catalog) loadBuiltin() {
	entries, err := seedFS.ReadDir("seed")
	if err != nil {
		c.logger.Error("appcenter: read embedded seed failed", "error", err)
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := seedFS.ReadFile(filepath.Join("seed", e.Name()))
		if err != nil {
			c.logger.Error("appcenter: read seed file failed", "file", e.Name(), "error", err)
			continue
		}
		m, err := ParseManifest(data)
		if err != nil {
			c.logger.Error("appcenter: invalid seed manifest", "file", e.Name(), "error", err)
			continue
		}
		c.builtin[m.ID] = CatalogEntry{Manifest: m, Origin: OriginBuiltin}
	}
}

// ScanLocal re-reads the local package directory. Each app lives in its own
// folder: $STATE/appcenter/apps/<id>/manifest.json. Invalid manifests are
// skipped with a log so one bad package can't break the catalog.
func (c *Catalog) ScanLocal() {
	if c.localDir == "" {
		return
	}
	dirs, err := os.ReadDir(c.localDir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		c.logger.Error("appcenter: scan local dir failed", "dir", c.localDir, "error", err)
		return
	}
	next := map[string]CatalogEntry{}
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		path := filepath.Join(c.localDir, d.Name(), "manifest.json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		m, err := ParseManifest(data)
		if err != nil {
			c.logger.Warn("appcenter: invalid local manifest", "path", path, "error", err)
			continue
		}
		next[m.ID] = CatalogEntry{Manifest: m, Origin: OriginLocal}
	}
	c.mu.Lock()
	c.local = next
	c.mu.Unlock()
}

// Entries returns every installable app, merged across sources with
// local > remote > builtin precedence, sorted by id for stable output.
func (c *Catalog) Entries() []CatalogEntry {
	c.mu.RLock()
	merged := map[string]CatalogEntry{}
	for id, e := range c.builtin {
		merged[id] = e
	}
	for id, e := range c.remote {
		merged[id] = e
	}
	for id, e := range c.local {
		merged[id] = e
	}
	c.mu.RUnlock()

	out := make([]CatalogEntry, 0, len(merged))
	for _, e := range merged {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Manifest.ID < out[j].Manifest.ID })
	return out
}

// Lookup resolves a single manifest by id, honoring source precedence.
func (c *Catalog) Lookup(id string) (CatalogEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if e, ok := c.local[id]; ok {
		return e, true
	}
	if e, ok := c.remote[id]; ok {
		return e, true
	}
	if e, ok := c.builtin[id]; ok {
		return e, true
	}
	return CatalogEntry{}, false
}
