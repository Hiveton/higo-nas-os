package network

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// HostAdapter drives real Linux networking: it reads live state with `ip` and
// writes a HiGoOS-owned netplan file applied via `netplan apply`. It only runs
// on the NAS host (selected by runtime.GOOS in NewService) but compiles
// everywhere — the commands simply aren't invoked on dev machines.
type HostAdapter struct {
	netplanPath string
}

// NewHostAdapter builds the Linux adapter. The netplan file path can be
// overridden with HIGO_NETPLAN_FILE (handy for staging).
func NewHostAdapter() *HostAdapter {
	path := strings.TrimSpace(os.Getenv("HIGO_NETPLAN_FILE"))
	if path == "" {
		path = "/etc/netplan/99-higoos.yaml"
	}
	return &HostAdapter{netplanPath: path}
}

// ipAddrEntry mirrors the relevant fields of `ip -j addr show`.
type ipAddrEntry struct {
	IfName   string `json:"ifname"`
	Address  string `json:"address"` // MAC
	Operstate string `json:"operstate"`
	Flags    []string `json:"flags"`
	AddrInfo []struct {
		Family    string `json:"family"`
		Local     string `json:"local"`
		PrefixLen int    `json:"prefixlen"`
	} `json:"addr_info"`
}

type ipRouteEntry struct {
	Dst     string `json:"dst"`
	Gateway string `json:"gateway"`
	Dev     string `json:"dev"`
}

func (h *HostAdapter) Interfaces(ctx context.Context) ([]Interface, error) {
	entries, err := h.readAddrs(ctx)
	if err != nil {
		return nil, err
	}
	primaryDev, _ := h.defaultRoute(ctx)
	out := make([]Interface, 0, len(entries))
	for _, e := range entries {
		if e.IfName == "lo" {
			continue
		}
		iface := Interface{
			Name:    e.IfName,
			MAC:     e.Address,
			Up:      e.Operstate == "UP" || containsFlag(e.Flags, "UP"),
			Primary: e.IfName == primaryDev,
		}
		for _, a := range e.AddrInfo {
			if a.Family == "inet" {
				iface.Addrs = append(iface.Addrs, fmt.Sprintf("%s/%d", a.Local, a.PrefixLen))
			}
		}
		out = append(out, iface)
	}
	return out, nil
}

func (h *HostAdapter) CurrentConfig(ctx context.Context) (NetworkConfig, error) {
	ifaces, err := h.Interfaces(ctx)
	if err != nil {
		return NetworkConfig{}, err
	}
	cfg := NetworkConfig{Mode: ModeDHCP}
	cfg.Hostname, _ = os.Hostname()
	dev, gw := h.defaultRoute(ctx)
	cfg.Gateway = gw
	// Prefer the primary (default-route) interface, else the first with an addr.
	var chosen *Interface
	for i := range ifaces {
		if ifaces[i].Name == dev {
			chosen = &ifaces[i]
			break
		}
	}
	if chosen == nil {
		for i := range ifaces {
			if len(ifaces[i].Addrs) > 0 {
				chosen = &ifaces[i]
				break
			}
		}
	}
	if chosen != nil {
		cfg.Interface = chosen.Name
		if len(chosen.Addrs) > 0 {
			if addr, prefix, ok := splitCIDR(chosen.Addrs[0]); ok {
				cfg.Address = addr
				cfg.Prefix = prefix
			}
		}
	}
	cfg.DNS = readResolvConf()
	return cfg, nil
}

func (h *HostAdapter) Apply(ctx context.Context, cfg NetworkConfig) error {
	if cfg.Interface == "" {
		return fmt.Errorf("interface is required to apply network config")
	}
	yaml := renderNetplan(cfg)
	if err := os.WriteFile(h.netplanPath, []byte(yaml), 0o600); err != nil {
		return fmt.Errorf("write netplan: %w", err)
	}
	if out, err := exec.CommandContext(ctx, "netplan", "apply").CombinedOutput(); err != nil {
		return fmt.Errorf("netplan apply: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if cfg.Hostname != "" {
		if out, err := exec.CommandContext(ctx, "hostnamectl", "set-hostname", cfg.Hostname).CombinedOutput(); err != nil {
			return fmt.Errorf("set hostname: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func (h *HostAdapter) readAddrs(ctx context.Context) ([]ipAddrEntry, error) {
	out, err := exec.CommandContext(ctx, "ip", "-j", "addr", "show").Output()
	if err != nil {
		return nil, fmt.Errorf("ip addr: %w", err)
	}
	var entries []ipAddrEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("parse ip addr: %w", err)
	}
	return entries, nil
}

func (h *HostAdapter) defaultRoute(ctx context.Context) (dev, gateway string) {
	out, err := exec.CommandContext(ctx, "ip", "-j", "route", "show", "default").Output()
	if err != nil {
		return "", ""
	}
	var routes []ipRouteEntry
	if json.Unmarshal(out, &routes) != nil {
		return "", ""
	}
	for _, r := range routes {
		if r.Dst == "default" {
			return r.Dev, r.Gateway
		}
	}
	return "", ""
}

// renderNetplan produces a minimal networkd-style netplan for one interface.
func renderNetplan(cfg NetworkConfig) string {
	var b strings.Builder
	b.WriteString("# Managed by HiGoOS — edits may be overwritten.\n")
	b.WriteString("network:\n  version: 2\n  renderer: networkd\n  ethernets:\n")
	fmt.Fprintf(&b, "    %s:\n", cfg.Interface)
	if cfg.Mode == ModeStatic {
		b.WriteString("      dhcp4: false\n")
		fmt.Fprintf(&b, "      addresses: [%s]\n", cfg.cidr())
		if cfg.Gateway != "" {
			fmt.Fprintf(&b, "      routes:\n        - to: default\n          via: %s\n", cfg.Gateway)
		}
		if len(cfg.DNS) > 0 {
			fmt.Fprintf(&b, "      nameservers:\n        addresses: [%s]\n", strings.Join(cfg.DNS, ", "))
		}
	} else {
		b.WriteString("      dhcp4: true\n")
	}
	return b.String()
}

func readResolvConf() []string {
	data, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return nil
	}
	var dns []string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "nameserver" {
			dns = append(dns, fields[1])
		}
	}
	return dns
}

func splitCIDR(cidr string) (addr string, prefix int, ok bool) {
	parts := strings.SplitN(cidr, "/", 2)
	if len(parts) != 2 {
		return "", 0, false
	}
	var p int
	if _, err := fmt.Sscanf(parts[1], "%d", &p); err != nil {
		return "", 0, false
	}
	return parts[0], p, true
}

func containsFlag(flags []string, want string) bool {
	for _, f := range flags {
		if f == want {
			return true
		}
	}
	return false
}
