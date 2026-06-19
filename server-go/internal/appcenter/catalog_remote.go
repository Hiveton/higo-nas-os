package appcenter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"higoos/server-go/internal/state"
)

// remoteIndex is the document a registry serves at its index URL. Either form is
// accepted: a bare JSON array of manifests, or an object with an "apps" array.
type remoteIndex struct {
	SchemaVersion string        `json:"schemaVersion"`
	Apps          []AppManifest `json:"apps"`
}

func (c *Catalog) loadRegistries() {
	if c.statePath == "" {
		return
	}
	var regs []Registry
	if err := state.LoadJSON(c.statePath, &regs); err != nil {
		c.logger.Error("appcenter: load registries failed", "error", err)
		return
	}
	c.mu.Lock()
	c.registries = regs
	c.mu.Unlock()
}

func (c *Catalog) saveRegistriesLocked() error {
	if c.statePath == "" {
		return nil
	}
	return state.SaveJSON(c.statePath, c.registries)
}

// Registries returns the configured remote sources.
func (c *Catalog) Registries() []Registry {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := append([]Registry(nil), c.registries...)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// AddRegistry registers (or re-enables) a remote source and persists it. The
// caller typically follows with RefreshRemote to pull its index immediately.
func (c *Catalog) AddRegistry(name, url string) (Registry, error) {
	name = strings.TrimSpace(name)
	url = strings.TrimSpace(url)
	if name == "" || url == "" {
		return Registry{}, fmt.Errorf("registry name and url are required")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.registries {
		if c.registries[i].Name == name {
			c.registries[i].URL = url
			c.registries[i].Enabled = true
			reg := c.registries[i]
			if err := c.saveRegistriesLocked(); err != nil {
				return Registry{}, err
			}
			return reg, nil
		}
	}
	reg := Registry{Name: name, URL: url, Enabled: true, AddedAt: time.Now()}
	c.registries = append(c.registries, reg)
	if err := c.saveRegistriesLocked(); err != nil {
		return Registry{}, err
	}
	return reg, nil
}

// RemoveRegistry drops a remote source and forgets the apps it contributed.
func (c *Catalog) RemoveRegistry(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := c.registries[:0]
	for _, r := range c.registries {
		if r.Name != name {
			next = append(next, r)
		}
	}
	c.registries = next
	for id, e := range c.remote {
		if e.Registry == name {
			delete(c.remote, id)
		}
	}
	return c.saveRegistriesLocked()
}

// RefreshRemote re-fetches every enabled registry index and rebuilds the remote
// catalog slice. Network/parse failures for one registry are logged and skipped
// so the rest of the catalog stays usable (offline-tolerant by design).
func (c *Catalog) RefreshRemote(ctx context.Context) error {
	c.mu.RLock()
	regs := append([]Registry(nil), c.registries...)
	c.mu.RUnlock()

	next := map[string]CatalogEntry{}
	var firstErr error
	for _, reg := range regs {
		if !reg.Enabled {
			continue
		}
		manifests, err := c.fetchIndex(ctx, reg.URL)
		if err != nil {
			c.logger.Warn("appcenter: registry refresh failed", "registry", reg.Name, "error", err)
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		for _, m := range manifests {
			next[m.ID] = CatalogEntry{Manifest: m, Origin: OriginRemote, Registry: reg.Name}
		}
	}
	c.mu.Lock()
	c.remote = next
	c.mu.Unlock()
	return firstErr
}

func (c *Catalog) fetchIndex(ctx context.Context, url string) ([]AppManifest, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("registry %s returned %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}

	var raw []AppManifest
	if err := json.Unmarshal(body, &raw); err != nil {
		var idx remoteIndex
		if err2 := json.Unmarshal(body, &idx); err2 != nil {
			return nil, fmt.Errorf("parse registry index: %w", err)
		}
		raw = idx.Apps
	}

	var valid []AppManifest
	for _, m := range raw {
		if err := m.Validate(); err != nil {
			c.logger.Warn("appcenter: registry manifest rejected", "id", m.ID, "error", err)
			continue
		}
		valid = append(valid, m.normalized())
	}
	return valid, nil
}
