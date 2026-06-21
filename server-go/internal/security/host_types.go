package security

import "time"

// ListeningPort is one socket the host is listening on, with a derived exposure
// + risk assessment (real data from `ss`/`/proc/net` on Linux).
type ListeningPort struct {
	Protocol string `json:"protocol"`          // tcp / udp
	Address  string `json:"address"`           // bind address, e.g. 0.0.0.0 / 127.0.0.1 / ::
	Port     int    `json:"port"`              //
	Process  string `json:"process,omitempty"` // owning process name
	PID      int    `json:"pid,omitempty"`
	Exposure string `json:"exposure"` // 公开监听 / 局域网 / 仅本机
	Risk     string `json:"risk"`     // low / medium / high
}

// FirewallState describes the host firewall (nftables / ufw / iptables).
type FirewallState struct {
	Backend string   `json:"backend"` // nftables / ufw / iptables / none
	Active  bool     `json:"active"`
	Rules   int      `json:"rules"`
	Summary string   `json:"summary"`
	Detail  []string `json:"detail,omitempty"`
}

// HostScanResult bundles a single security scan: open sockets + firewall + a
// count of ports exposed on all interfaces.
type HostScanResult struct {
	Ports     []ListeningPort `json:"ports"`
	Firewall  FirewallState   `json:"firewall"`
	OpenToAll int             `json:"openToAll"`
	ScannedAt time.Time       `json:"scannedAt"`
}
