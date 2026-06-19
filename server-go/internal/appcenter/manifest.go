package appcenter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"higoos/server-go/internal/audit"
)

// SchemaVersion is the manifest schema this build understands. Third-party
// packages must declare it so the catalog can reject incompatible manifests.
const SchemaVersion = "1"

// AppManifest is the declarative package format a third-party developer authors
// to publish an app to the HiGoOS app center. It is the single source of truth
// for the catalog, the install pipeline (container specs) and the desktop UI
// integration (web entry). It deliberately carries no install/runtime state —
// that lives in the per-app install state managed by the service.
type AppManifest struct {
	SchemaVersion string          `json:"schemaVersion"`
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	Category      string          `json:"category"`
	Description   string          `json:"description"`
	Author        Author          `json:"author,omitempty"`
	IconURL       string          `json:"iconUrl,omitempty"`
	Risk          string          `json:"risk"`   // low|medium|high
	Source        string          `json:"source"` // official|community|third-party
	Containers    []ContainerSpec `json:"containers"`
	WebEntry      *WebEntry       `json:"webEntry,omitempty"`
	Permissions   []string        `json:"permissions,omitempty"`
	Config        []ConfigField   `json:"config,omitempty"`
}

// Author identifies the package publisher.
type Author struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

// ContainerSpec describes one container the app runs. Apps may declare several.
type ContainerSpec struct {
	Name          string        `json:"name"`
	Image         string        `json:"image"`
	Ports         []PortMapping `json:"ports,omitempty"`
	Volumes       []VolumeMount `json:"volumes,omitempty"`
	Env           []EnvVar      `json:"env,omitempty"`
	Command       string        `json:"command,omitempty"`
	Resources     Resources     `json:"resources"`
	RestartPolicy string        `json:"restartPolicy,omitempty"`
}

// PortMapping is a container port optionally published on a host port.
type PortMapping struct {
	Container int    `json:"container"`
	Host      int    `json:"host,omitempty"`
	Protocol  string `json:"protocol,omitempty"` // tcp|udp (default tcp)
}

// VolumeMount maps a named volume to a path inside the container.
type VolumeMount struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// EnvVar is an environment variable. Value supports {{config.KEY}} interpolation
// against the user-supplied config collected at install time.
type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Resources is the container resource budget, translated to docker limits.
type Resources struct {
	CPU      int `json:"cpu"`
	MemoryMB int `json:"memoryMb"`
}

// WebEntry tells the desktop shell how to surface the app's own web UI. When
// Display is "embed" the shell renders it inside an iframe window; "external"
// opens it in a new browser tab.
type WebEntry struct {
	Container string `json:"container,omitempty"`
	Port      int    `json:"port"`
	Path      string `json:"path,omitempty"`
	Display   string `json:"display,omitempty"` // embed|external (default embed)
}

// ConfigField is one user-supplied parameter requested at install time and made
// available to env interpolation via {{config.KEY}}.
type ConfigField struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Type     string `json:"type"` // text|number|password|boolean
	Default  string `json:"default,omitempty"`
	Required bool   `json:"required,omitempty"`
	Secret   bool   `json:"secret,omitempty"`
}

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ParseManifest decodes and validates a manifest document.
func ParseManifest(data []byte) (AppManifest, error) {
	var m AppManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return AppManifest{}, fmt.Errorf("decode manifest: %w", err)
	}
	if err := m.Validate(); err != nil {
		return AppManifest{}, err
	}
	return m.normalized(), nil
}

// Validate enforces the manifest contract. It is the single gate every catalog
// source funnels through, so a malformed third-party package can never reach
// the install pipeline.
func (m AppManifest) Validate() error {
	if m.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schemaVersion %q (want %q)", m.SchemaVersion, SchemaVersion)
	}
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("invalid id %q (want kebab-case [a-z0-9-])", m.ID)
	}
	if strings.TrimSpace(m.Name) == "" {
		return fmt.Errorf("app %s: name is required", m.ID)
	}
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("app %s: version is required", m.ID)
	}
	switch m.Risk {
	case "low", "medium", "high":
	default:
		return fmt.Errorf("app %s: risk must be low|medium|high, got %q", m.ID, m.Risk)
	}
	if m.Source != "" {
		switch m.Source {
		case "official", "community", "third-party":
		default:
			return fmt.Errorf("app %s: source must be official|community|third-party, got %q", m.ID, m.Source)
		}
	}
	if len(m.Containers) == 0 {
		return fmt.Errorf("app %s: at least one container is required", m.ID)
	}
	names := map[string]bool{}
	for i, c := range m.Containers {
		if strings.TrimSpace(c.Name) == "" {
			return fmt.Errorf("app %s: container[%d] name is required", m.ID, i)
		}
		if strings.TrimSpace(c.Image) == "" {
			return fmt.Errorf("app %s: container %q image is required", m.ID, c.Name)
		}
		if names[c.Name] {
			return fmt.Errorf("app %s: duplicate container name %q", m.ID, c.Name)
		}
		names[c.Name] = true
	}
	if m.WebEntry != nil {
		if m.WebEntry.Port <= 0 {
			return fmt.Errorf("app %s: webEntry.port must be > 0", m.ID)
		}
		if m.WebEntry.Container != "" && !names[m.WebEntry.Container] {
			return fmt.Errorf("app %s: webEntry.container %q is not a declared container", m.ID, m.WebEntry.Container)
		}
	}
	keys := map[string]bool{}
	for i, f := range m.Config {
		if strings.TrimSpace(f.Key) == "" {
			return fmt.Errorf("app %s: config[%d] key is required", m.ID, i)
		}
		if keys[f.Key] {
			return fmt.Errorf("app %s: duplicate config key %q", m.ID, f.Key)
		}
		keys[f.Key] = true
	}
	return nil
}

// normalized fills in conventional defaults so downstream code can assume them.
func (m AppManifest) normalized() AppManifest {
	if m.Source == "" {
		m.Source = "third-party"
	}
	if m.WebEntry != nil {
		if m.WebEntry.Display == "" {
			m.WebEntry.Display = "embed"
		}
		if m.WebEntry.Container == "" && len(m.Containers) > 0 {
			m.WebEntry.Container = m.Containers[0].Name
		}
	}
	for i := range m.Containers {
		for j := range m.Containers[i].Ports {
			if m.Containers[i].Ports[j].Protocol == "" {
				m.Containers[i].Ports[j].Protocol = "tcp"
			}
		}
	}
	return m
}

// RiskLevel maps the manifest risk to the shared audit risk vocabulary.
func (m AppManifest) RiskLevel() audit.RiskLevel {
	switch m.Risk {
	case "high":
		return audit.RiskHigh
	case "medium":
		return audit.RiskMedium
	default:
		return audit.RiskLow
	}
}

// primaryImage returns the first container image (used for the App summary).
func (m AppManifest) primaryImage() string {
	if len(m.Containers) == 0 {
		return ""
	}
	return m.Containers[0].Image
}

// portStrings flattens every container port into the "host:container/proto"
// display/wire form the rest of the platform already uses.
func (m AppManifest) portStrings() []string {
	var out []string
	for _, c := range m.Containers {
		for _, p := range c.Ports {
			out = append(out, formatPort(p))
		}
	}
	return out
}

func formatPort(p PortMapping) string {
	proto := p.Protocol
	if proto == "" {
		proto = "tcp"
	}
	if p.Host != 0 && p.Host != p.Container {
		return fmt.Sprintf("%d:%d/%s", p.Host, p.Container, proto)
	}
	return fmt.Sprintf("%d/%s", p.Container, proto)
}

func riskDisplay(risk string) string {
	switch risk {
	case "high":
		return "高风险"
	case "medium":
		return "中风险"
	default:
		return "低风险"
	}
}

func sourceDisplay(source string) string {
	switch source {
	case "official":
		return "官方镜像"
	case "community":
		return "社区精选"
	default:
		return "三方应用"
	}
}

func resourceText(r Resources) string {
	if r.CPU == 0 && r.MemoryMB == 0 {
		return "按需分配"
	}
	return fmt.Sprintf("%d CPU / %d MB", r.CPU, r.MemoryMB)
}

// totalResources sums the resource budget across every container, for the App
// summary line and the install impact summary.
func (m AppManifest) totalResources() Resources {
	var out Resources
	for _, c := range m.Containers {
		out.CPU += c.Resources.CPU
		out.MemoryMB += c.Resources.MemoryMB
	}
	return out
}

// configDefaults returns the config map seeded from the field defaults, used as
// the starting point before user-supplied values are merged.
func (m AppManifest) configDefaults() map[string]string {
	out := map[string]string{}
	for _, f := range m.Config {
		if f.Default != "" {
			out[f.Key] = f.Default
		}
	}
	return out
}

// configKeysSorted returns config keys in a stable order for deterministic
// interpolation and impact summaries.
func (m AppManifest) configKeysSorted() []string {
	keys := make([]string, 0, len(m.Config))
	for _, f := range m.Config {
		keys = append(keys, f.Key)
	}
	sort.Strings(keys)
	return keys
}
