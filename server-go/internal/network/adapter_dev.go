package network

import (
	"context"
	"sync"
)

// DevAdapter is the deterministic, Mac-safe network adapter used in development
// and tests. It keeps the configuration in memory; Apply round-trips so the
// preview/confirm flow can be exercised without touching the host.
type DevAdapter struct {
	mu      sync.Mutex
	ifaces  []Interface
	current NetworkConfig
}

// NewDevAdapter seeds a representative interface and a DHCP default.
func NewDevAdapter() *DevAdapter {
	return &DevAdapter{
		ifaces: []Interface{
			{Name: "eth0", MAC: "02:42:ac:11:00:02", Up: true, Addrs: []string{"10.211.55.3/24"}, Primary: true, Mode: ModeDHCP},
			{Name: "lo", MAC: "", Up: true, Addrs: []string{"127.0.0.1/8"}},
		},
		current: NetworkConfig{
			Interface: "eth0",
			Mode:      ModeDHCP,
			Address:   "10.211.55.3",
			Prefix:    24,
			Gateway:   "10.211.55.1",
			DNS:       []string{"223.5.5.5", "8.8.8.8"},
			Hostname:  "higoos",
		},
	}
}

func (d *DevAdapter) Interfaces(ctx context.Context) ([]Interface, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Interface(nil), d.ifaces...), nil
}

func (d *DevAdapter) CurrentConfig(ctx context.Context) (NetworkConfig, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.current, nil
}

func (d *DevAdapter) Apply(ctx context.Context, cfg NetworkConfig) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if cfg.Interface == "" {
		cfg.Interface = d.current.Interface
	}
	d.current = cfg
	for i := range d.ifaces {
		if d.ifaces[i].Name != cfg.Interface {
			continue
		}
		d.ifaces[i].Mode = cfg.Mode
		if cfg.Mode == ModeStatic && cfg.Address != "" {
			d.ifaces[i].Addrs = []string{cfg.cidr()}
		}
	}
	return nil
}
