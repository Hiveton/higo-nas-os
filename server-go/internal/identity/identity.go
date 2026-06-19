// Package identity exposes a stable, unauthenticated device fingerprint used by
// the LAN discovery daemon (internal/discovery) and the public
// GET /api/v1/system/identity endpoint. The desktop assistant
// (tools/higo-assistant) reads it to confirm "this host is a HiGoOS device"
// before opening the web UI or configuring the network.
//
// The deviceId is generated once and persisted under HIGO_STATE_DIR; everything
// else (hostname, addresses, MAC, uptime) is sampled live on each snapshot.
package identity

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// Identity is the safe device fingerprint. It deliberately carries no secrets:
// it is served to unauthenticated callers on the LAN.
type Identity struct {
	DeviceID    string   `json:"deviceId"`
	Model       string   `json:"model"`
	Version     string   `json:"version"`
	Hostname    string   `json:"hostname"`
	Initialized bool     `json:"initialized"`
	HTTPPort    int      `json:"httpPort"`
	HTTPS       bool     `json:"https"`
	PrimaryMAC  string   `json:"primaryMac"`
	Addrs       []string `json:"addrs"`
	UptimeSec   int64    `json:"uptimeSec"`
}

type persisted struct {
	DeviceID    string `json:"deviceId"`
	Initialized bool   `json:"initialized"`
}

// Provider assembles Identity snapshots. Safe for concurrent use.
type Provider struct {
	mu          sync.RWMutex
	deviceID    string
	initialized bool
	statePath   string

	model    string
	version  string
	httpPort int
	https    bool
	bootedAt time.Time
}

// NewProvider builds a provider. The deviceId is loaded from
// stateDir/identity.json or generated and persisted on first run. httpAddr is
// the API listen address (e.g. ":8080") used to advertise the web port.
func NewProvider(stateDir, model, version, httpAddr string) (*Provider, error) {
	p := &Provider{
		model:    firstNonEmpty(model, "HiGoOS NAS"),
		version:  firstNonEmpty(version, "dev"),
		httpPort: portFromAddr(httpAddr, 8080),
		bootedAt: time.Now().UTC(),
	}
	if strings.TrimSpace(stateDir) != "" {
		p.statePath = filepath.Join(stateDir, "identity.json")
	}
	if err := p.load(); err != nil {
		return nil, err
	}
	if p.deviceID == "" {
		p.deviceID = generateDeviceID()
		if err := p.saveLocked(); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (p *Provider) load() error {
	if p.statePath == "" {
		return nil
	}
	var stored persisted
	if err := state.LoadJSON(p.statePath, &stored); err != nil {
		return err
	}
	p.deviceID = stored.DeviceID
	p.initialized = stored.Initialized
	return nil
}

func (p *Provider) saveLocked() error {
	if p.statePath == "" {
		return nil
	}
	return state.SaveJSON(p.statePath, persisted{DeviceID: p.deviceID, Initialized: p.initialized})
}

// Snapshot samples the live device fingerprint.
func (p *Provider) Snapshot() Identity {
	p.mu.RLock()
	id := Identity{
		DeviceID:    p.deviceID,
		Model:       p.model,
		Version:     p.version,
		Initialized: p.initialized,
		HTTPPort:    p.httpPort,
		HTTPS:       p.https,
		UptimeSec:   int64(time.Since(p.bootedAt).Seconds()),
	}
	p.mu.RUnlock()

	id.Hostname, _ = os.Hostname()
	id.PrimaryMAC, id.Addrs = primaryNetwork()
	if id.Addrs == nil {
		id.Addrs = []string{}
	}
	return id
}

// MarkInitialized records that first-boot onboarding has completed, flipping the
// fingerprint so the assistant offers "login" instead of "setup".
func (p *Provider) MarkInitialized() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.initialized {
		return nil
	}
	p.initialized = true
	return p.saveLocked()
}

// DeviceID returns the stable device identifier.
func (p *Provider) DeviceID() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.deviceID
}

func generateDeviceID() string {
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		// Fall back to a time-derived id; collisions on a single LAN are unlikely.
		return "hg-" + strconv.FormatInt(time.Now().UnixNano()&0xffffffff, 16)
	}
	return "hg-" + hex.EncodeToString(buf)
}

// primaryNetwork returns the MAC and IPv4 addresses of the first up, non-loopback
// interface carrying a global-unicast IPv4 address, plus every other global
// IPv4 address discovered.
func primaryNetwork() (mac string, addrs []string) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		entries, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, entry := range entries {
			ip := ipFromAddr(entry)
			if ip == nil || ip.To4() == nil || !ip.IsGlobalUnicast() {
				continue
			}
			if mac == "" {
				mac = iface.HardwareAddr.String()
			}
			addrs = append(addrs, ip.String())
		}
	}
	return mac, addrs
}

func ipFromAddr(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPNet:
		return v.IP
	case *net.IPAddr:
		return v.IP
	default:
		return nil
	}
}

func portFromAddr(addr string, fallback int) int {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fallback
	}
	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		// addr may be a bare port like "8080".
		if p, perr := strconv.Atoi(strings.TrimPrefix(addr, ":")); perr == nil {
			return p
		}
		return fallback
	}
	if p, err := strconv.Atoi(portStr); err == nil {
		return p
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
