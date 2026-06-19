package appcenter

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"higoos/server-go/internal/audit"
	hdocker "higoos/server-go/internal/docker"
	"higoos/server-go/internal/state"
)

// App is the wire projection of an installable app: the manifest's catalog
// fields overlaid with its install/runtime state. The shape is backward
// compatible with the previous hardcoded model (frontend + existing tests rely
// on it) and is additively extended with manifest-driven fields.
type App struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	Version         string    `json:"version"`
	LatestVersion   string    `json:"latestVersion"`
	Status          string    `json:"status"`
	Description     string    `json:"description"`
	Source          string    `json:"source"`
	Risk            string    `json:"risk"`
	Resource        string    `json:"resource"`
	Image           string    `json:"image,omitempty"`
	Ports           []string  `json:"ports"`
	ContainerID     string    `json:"containerId,omitempty"`
	Installed       bool      `json:"installed"`
	Running         bool      `json:"running"`
	UpdateAvailable bool      `json:"updateAvailable"`
	ManifestID      string    `json:"manifestId,omitempty"`
	WebEntry        *WebEntry `json:"webEntry,omitempty"`
	Permissions     []string  `json:"permissions,omitempty"`
}

// persistedApp is the internal record: the wire App plus install detail that is
// never surfaced to clients (config values may carry secrets, container ids per
// spec). It serializes the App fields flat, so an old app-center.json ([]App)
// still loads cleanly into []persistedApp during migration.
type persistedApp struct {
	App
	ConfigValues map[string]string `json:"configValues,omitempty"`
	Containers   map[string]string `json:"containers,omitempty"`
}

// ContainerRunner is the slice of the docker domain the app center needs to
// orchestrate real containers. *docker.DevService satisfies it (and routes to
// the live docker CLI on a Linux host).
type ContainerRunner interface {
	CreateContainer(ctx context.Context, request hdocker.CreateContainerRequest) (hdocker.Container, error)
	Start(ctx context.Context, containerID string) (hdocker.Container, error)
	Stop(ctx context.Context, containerID string) (hdocker.Container, error)
	RemoveContainer(ctx context.Context, containerID string, request hdocker.RemoveContainerRequest) error
}

type Service struct {
	mu        sync.RWMutex
	apps      []persistedApp
	statePath string
	docker    ContainerRunner
	catalog   *Catalog
	audit     *audit.Store
	logger    *slog.Logger

	// governance (see governance.go)
	confirms    map[string]pendingConfirm
	nextConfirm int
	auditLog    []AuditRecord
	auditPath   string
}

// demoState reproduces the previous hardcoded install state for the seed apps so
// a fresh control plane (and the existing test suite) sees the same demo NAS.
// Third-party packages never carry install state — it is pure runtime.
var demoState = map[string]struct {
	Version string
	Running bool
	Status  string
}{
	"home-assistant": {Version: "2026.4.1", Running: true, Status: "需更新"},
	"qdrant":         {Version: "1.14.0", Running: true, Status: "运行中"},
	"tailscale":      {Version: "1.82.5", Running: false, Status: "已停止"},
}

// NewService builds an in-memory service from the embedded seed catalog (no
// state dir). Used by unit tests and as the stateless fallback.
func NewService() *Service {
	return newService(NewCatalog("", slog.Default()))
}

func newService(cat *Catalog) *Service {
	s := &Service{
		catalog:  cat,
		logger:   slog.Default(),
		confirms: map[string]pendingConfirm{},
	}
	s.apps = seedApps(cat)
	return s
}

// NewServiceWithStateDir builds a service whose catalog also scans local
// packages and registries under stateDir, then restores persisted install state.
func NewServiceWithStateDir(stateDir string) (*Service, error) {
	s := newService(NewCatalog(stateDir, slog.Default()))
	if stateDir == "" {
		return s, nil
	}
	s.statePath = filepath.Join(stateDir, "app-center.json")
	s.auditPath = filepath.Join(stateDir, "appcenter-audit.json")
	var persisted []persistedApp
	if err := state.LoadJSON(s.statePath, &persisted); err != nil {
		return nil, err
	}
	s.mergePersisted(persisted)
	if err := state.LoadJSON(s.auditPath, &s.auditLog); err != nil {
		return nil, err
	}
	return s, nil
}

// seedApps materializes one persistedApp per catalog entry, then overlays the
// demo install state for the known seed apps.
func seedApps(cat *Catalog) []persistedApp {
	var out []persistedApp
	for _, e := range cat.Entries() {
		out = append(out, newPersistedApp(e.Manifest))
	}
	for i := range out {
		d, ok := demoState[out[i].ID]
		if !ok {
			continue
		}
		out[i].Installed = true
		out[i].Running = d.Running
		out[i].Version = d.Version
		out[i].Status = d.Status
		out[i].UpdateAvailable = out[i].Version != out[i].LatestVersion
	}
	return out
}

// newPersistedApp projects a manifest into a not-yet-installed app record.
func newPersistedApp(m AppManifest) persistedApp {
	app := App{
		ID:            m.ID,
		Name:          m.Name,
		Category:      m.Category,
		Version:       "",
		LatestVersion: m.Version,
		Status:        "可安装",
		Description:   m.Description,
		Source:        sourceDisplay(m.Source),
		Risk:          riskDisplay(m.Risk),
		Resource:      resourceText(m.totalResources()),
		Image:         m.primaryImage(),
		Ports:         m.portStrings(),
		ManifestID:    m.ID,
		WebEntry:      m.WebEntry,
		Permissions:   m.Permissions,
	}
	return persistedApp{App: app, ConfigValues: m.configDefaults(), Containers: map[string]string{}}
}

// mergePersisted overlays restored install state onto the freshly-seeded catalog
// apps. Catalog metadata (latest version, image, ports) stays authoritative;
// only runtime state is restored. Installed apps missing from the catalog are
// kept so an app is never silently lost when its manifest disappears.
func (s *Service) mergePersisted(persisted []persistedApp) {
	byID := map[string]int{}
	for i := range s.apps {
		byID[s.apps[i].ID] = i
	}
	for _, p := range persisted {
		if idx, ok := byID[p.ID]; ok {
			a := &s.apps[idx]
			a.Installed = p.Installed
			a.Running = p.Running
			a.Version = p.Version
			a.Status = p.Status
			a.ContainerID = p.ContainerID
			a.ConfigValues = p.ConfigValues
			a.Containers = p.Containers
			a.UpdateAvailable = a.Installed && a.Version != a.LatestVersion
			continue
		}
		s.apps = append(s.apps, p)
		byID[p.ID] = len(s.apps) - 1
	}
}

// AttachDocker wires the docker domain so install/start/stop orchestrate real
// containers. Without it the app center keeps its in-memory state behaviour.
func (s *Service) AttachDocker(d ContainerRunner) { s.docker = d }

// AttachAudit wires the shared audit store so governed actions are recorded and
// reversible. Without it governance still gates side effects but skips audit.
func (s *Service) AttachAudit(a *audit.Store) { s.audit = a }

// Catalog exposes the catalog for the HTTP/MCP layers (list, detail, refresh).
func (s *Service) CatalogSource() *Catalog { return s.catalog }

// Apps returns every known app (installed and installable) as wire projections.
func (s *Service) Apps(ctx context.Context) ([]App, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]App, 0, len(s.apps))
	for _, p := range s.apps {
		out = append(out, cloneApp(p.App))
	}
	return out, nil
}

// Get returns a single app projection.
func (s *Service) Get(id string) (App, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.apps {
		if p.ID == id {
			return cloneApp(p.App), true
		}
	}
	return App{}, false
}

// ----- lifecycle (pure orchestration; governance wraps these in governance.go) -----

// Install marks an app installed/running and orchestrates its containers. If the
// app id is only present in the catalog (e.g. a freshly-scanned local package or
// a registry app) it is materialized first.
func (s *Service) Install(ctx context.Context, id string) (App, error) {
	return s.InstallWithConfig(ctx, id, nil)
}

// InstallWithConfig is Install with user-supplied config values merged over the
// manifest defaults (used for env interpolation).
func (s *Service) InstallWithConfig(ctx context.Context, id string, config map[string]string) (App, error) {
	if err := s.ensureMaterialized(id); err != nil {
		return App{}, err
	}
	m, ok := s.manifestFor(id)
	merged := s.mergedConfig(id, m, config)
	if _, err := s.update(ctx, id, func(a *persistedApp) {
		a.Installed = true
		a.Running = true
		a.Version = a.LatestVersion
		a.Status = "运行中"
		a.UpdateAvailable = false
		if config != nil {
			a.ConfigValues = merged
		}
	}); err != nil {
		return App{}, err
	}
	if !ok {
		return s.snapshot(id)
	}
	return s.orchestrateRunning(ctx, id, m, merged)
}

// Update bumps the app to its latest version and recreates containers so the new
// image takes effect.
func (s *Service) Update(ctx context.Context, id string) (App, error) {
	m, ok := s.manifestFor(id)
	app, err := s.update(ctx, id, func(a *persistedApp) {
		a.Installed = true
		a.Running = true
		a.Version = a.LatestVersion
		a.Status = "已更新"
		a.UpdateAvailable = false
	})
	if err != nil {
		return App{}, err
	}
	if !ok || s.docker == nil || app.Image == "" {
		return app, nil
	}
	// Only recreate when containers were actually orchestrated before; an app in
	// the optimistic dev-fallback state (no bound container) just bumps version.
	if len(s.containersFor(id)) == 0 {
		return app, nil
	}
	// Recreate so the new image actually takes effect: remove old containers,
	// clear the binding, then orchestrate fresh ones.
	s.removeContainers(ctx, id, false)
	if _, err := s.update(ctx, id, func(a *persistedApp) {
		a.Containers = map[string]string{}
		a.ContainerID = ""
	}); err != nil {
		return App{}, err
	}
	return s.orchestrateRunning(ctx, id, m, s.configFor(id))
}

// Start brings a stopped app back up.
func (s *Service) Start(ctx context.Context, id string) (App, error) {
	m, ok := s.manifestFor(id)
	app, err := s.update(ctx, id, func(a *persistedApp) {
		a.Installed = true
		a.Running = true
		if a.Version == "" {
			a.Version = a.LatestVersion
		}
		a.Status = "运行中"
	})
	if err != nil {
		return App{}, err
	}
	if !ok {
		return app, nil
	}
	return s.orchestrateRunning(ctx, id, m, s.configFor(id))
}

// Stop stops an app's containers.
func (s *Service) Stop(ctx context.Context, id string) (App, error) {
	app, err := s.update(ctx, id, func(a *persistedApp) {
		a.Running = false
		a.Status = "已停止"
	})
	if err != nil {
		return App{}, err
	}
	if err := s.stopContainers(ctx, id); err != nil {
		return s.update(ctx, id, func(a *persistedApp) {
			a.Running = false
			a.Status = "已停止（停止容器失败：" + err.Error() + "）"
		})
	}
	return app, nil
}

// Uninstall stops and force-removes the app's containers (optionally its
// volumes) and resets it to the installable state.
func (s *Service) Uninstall(ctx context.Context, id string, removeVolumes bool) (App, error) {
	if _, ok := s.find(id); !ok {
		return App{}, fmt.Errorf("app center item not found: %s", id)
	}
	s.removeContainers(ctx, id, removeVolumes)
	return s.update(ctx, id, func(a *persistedApp) {
		a.Installed = false
		a.Running = false
		a.Version = ""
		a.Status = "可安装"
		a.UpdateAvailable = false
		a.ContainerID = ""
		a.Containers = map[string]string{}
	})
}

// ----- internal helpers -----

func (s *Service) manifestFor(id string) (AppManifest, bool) {
	if e, ok := s.catalog.Lookup(id); ok {
		return e.Manifest, true
	}
	return AppManifest{}, false
}

// ensureMaterialized adds a catalog-only app (e.g. from a registry) to the
// working set so it can carry install state.
func (s *Service) ensureMaterialized(id string) error {
	if _, ok := s.find(id); ok {
		return nil
	}
	e, ok := s.catalog.Lookup(id)
	if !ok {
		return fmt.Errorf("app center item not found: %s", id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.apps {
		if s.apps[i].ID == id {
			return nil
		}
	}
	s.apps = append(s.apps, newPersistedApp(e.Manifest))
	return s.saveLocked()
}

func (s *Service) find(id string) (persistedApp, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.apps {
		if p.ID == id {
			return p, true
		}
	}
	return persistedApp{}, false
}

func (s *Service) snapshot(id string) (App, error) {
	if p, ok := s.find(id); ok {
		return cloneApp(p.App), nil
	}
	return App{}, fmt.Errorf("app center item not found: %s", id)
}

func (s *Service) containersFor(id string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.apps {
		if p.ID == id {
			out := map[string]string{}
			for k, v := range p.Containers {
				out[k] = v
			}
			return out
		}
	}
	return map[string]string{}
}

func (s *Service) configFor(id string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.apps {
		if p.ID == id {
			return cloneStrMap(p.ConfigValues)
		}
	}
	return map[string]string{}
}

// mergedConfig layers user-supplied values over the app's stored config and the
// manifest defaults.
func (s *Service) mergedConfig(id string, m AppManifest, override map[string]string) map[string]string {
	out := m.configDefaults()
	for k, v := range s.configFor(id) {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func (s *Service) update(ctx context.Context, id string, mutate func(*persistedApp)) (App, error) {
	if err := ctx.Err(); err != nil {
		return App{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.apps {
		if s.apps[i].ID == id {
			if s.apps[i].Containers == nil {
				s.apps[i].Containers = map[string]string{}
			}
			mutate(&s.apps[i])
			if err := s.saveLocked(); err != nil {
				return App{}, err
			}
			return cloneApp(s.apps[i].App), nil
		}
	}
	return App{}, fmt.Errorf("app center item not found: %s", id)
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, s.apps)
}

func cloneApp(app App) App {
	app.Ports = append([]string(nil), app.Ports...)
	app.Permissions = append([]string(nil), app.Permissions...)
	if app.WebEntry != nil {
		w := *app.WebEntry
		app.WebEntry = &w
	}
	return app
}

func cloneStrMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
