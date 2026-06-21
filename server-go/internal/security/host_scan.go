package security

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// HostScanAdapter runs real scans on a Linux host: `ss` for listening sockets
// and `nft`/`ufw`/`iptables` for the firewall. Every command goes through the
// injectable runner so the parsing is unit-testable off-Linux.
type HostScanAdapter struct {
	runner commandRunner
}

func NewHostScanAdapter() *HostScanAdapter { return NewHostScanAdapterWithRunner(runCommand) }

func NewHostScanAdapterWithRunner(runner commandRunner) *HostScanAdapter {
	if runner == nil {
		runner = runCommand
	}
	return &HostScanAdapter{runner: runner}
}

// sensitivePorts are services that are higher-risk when exposed on all interfaces.
var sensitivePorts = map[int]bool{
	22: true, 23: true, 445: true, 139: true, 3389: true, 2049: true,
	5432: true, 3306: true, 6379: true, 27017: true, 11211: true,
}

func (a *HostScanAdapter) ListeningPorts(ctx context.Context) ([]ListeningPort, error) {
	out, err := a.runner(ctx, "ss", "-Htulnp")
	if err != nil {
		// ss may be missing on minimal images; surface an empty list rather than
		// failing the whole scan.
		return nil, nil
	}
	var ports []ListeningPort
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		netid := fields[0]
		if netid != "tcp" && netid != "udp" {
			continue
		}
		addr, port, ok := splitHostPort(fields[4])
		if !ok {
			continue
		}
		p := ListeningPort{Protocol: netid, Address: addr, Port: port}
		if len(fields) >= 7 {
			p.Process, p.PID = parseSSProcess(fields[6])
		}
		p.Exposure = exposureFor(addr)
		p.Risk = riskFor(p.Exposure, port)
		ports = append(ports, p)
	}
	return ports, nil
}

func (a *HostScanAdapter) Firewall(ctx context.Context) (FirewallState, error) {
	// Prefer nftables (modern default), then ufw, then iptables.
	if out, err := a.runner(ctx, "nft", "list", "ruleset"); err == nil {
		text := strings.TrimSpace(string(out))
		if text != "" && strings.Contains(text, "table") {
			rules := countNftRules(text)
			return FirewallState{Backend: "nftables", Active: rules > 0, Rules: rules, Summary: fmt.Sprintf("nftables 已启用，%d 条规则", rules)}, nil
		}
		return FirewallState{Backend: "nftables", Active: false, Rules: 0, Summary: "nftables 可用但无规则"}, nil
	}
	if out, err := a.runner(ctx, "ufw", "status"); err == nil {
		text := string(out)
		active := strings.Contains(text, "Status: active")
		rules := 0
		var detail []string
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "ALLOW") || strings.Contains(line, "DENY") || strings.Contains(line, "REJECT") {
				rules++
				detail = append(detail, line)
			}
		}
		summary := "ufw 未启用"
		if active {
			summary = fmt.Sprintf("ufw 已启用，%d 条规则", rules)
		}
		return FirewallState{Backend: "ufw", Active: active, Rules: rules, Summary: summary, Detail: detail}, nil
	}
	if out, err := a.runner(ctx, "iptables", "-S"); err == nil {
		rules := 0
		for _, line := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "-A") {
				rules++
			}
		}
		return FirewallState{Backend: "iptables", Active: rules > 0, Rules: rules, Summary: fmt.Sprintf("iptables，%d 条规则", rules)}, nil
	}
	return FirewallState{Backend: "none", Active: false, Summary: "未检测到防火墙后端"}, nil
}

// splitHostPort parses ss's "addr:port" supporting IPv6 [::]:445, 0.0.0.0:445,
// *:111 and 127.0.0.1:5432.
func splitHostPort(s string) (string, int, bool) {
	i := strings.LastIndex(s, ":")
	if i < 0 {
		return "", 0, false
	}
	host := s[:i]
	host = strings.TrimPrefix(host, "[")
	host = strings.TrimSuffix(host, "]")
	if host == "*" {
		host = "0.0.0.0"
	}
	port, err := strconv.Atoi(s[i+1:])
	if err != nil {
		return "", 0, false
	}
	return host, port, true
}

// parseSSProcess extracts the process name + pid from ss's
// users:(("smbd",pid=1234,fd=35)) column.
func parseSSProcess(s string) (string, int) {
	name := ""
	if i := strings.Index(s, "((\""); i >= 0 {
		rest := s[i+3:]
		if j := strings.Index(rest, "\""); j >= 0 {
			name = rest[:j]
		}
	}
	pid := 0
	if i := strings.Index(s, "pid="); i >= 0 {
		rest := s[i+4:]
		end := strings.IndexFunc(rest, func(r rune) bool { return r < '0' || r > '9' })
		if end < 0 {
			end = len(rest)
		}
		pid, _ = strconv.Atoi(rest[:end])
	}
	return name, pid
}

func exposureFor(addr string) string {
	switch addr {
	case "0.0.0.0", "::", "":
		return "公开监听"
	case "127.0.0.1", "::1":
		return "仅本机"
	default:
		return "局域网"
	}
}

func riskFor(exposure string, port int) string {
	if exposure == "仅本机" {
		return "low"
	}
	if exposure == "公开监听" {
		if sensitivePorts[port] {
			return "high"
		}
		return "medium"
	}
	return "low"
}

// countNftRules approximates the active rule count by counting indented rule
// lines (skipping table/chain/brace boilerplate).
func countNftRules(ruleset string) int {
	count := 0
	for _, line := range strings.Split(ruleset, "\n") {
		t := strings.TrimSpace(line)
		if t == "" || t == "}" || strings.HasPrefix(t, "table ") || strings.HasPrefix(t, "chain ") || strings.HasPrefix(t, "type ") {
			continue
		}
		count++
	}
	return count
}
