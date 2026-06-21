package security

import (
	"context"
	"os/exec"
)

// scanAdapter abstracts host security scanning. HostScanAdapter runs real
// commands on Linux; DevScanAdapter returns deterministic seed data elsewhere.
type scanAdapter interface {
	ListeningPorts(ctx context.Context) ([]ListeningPort, error)
	Firewall(ctx context.Context) (FirewallState, error)
}

type commandRunner func(context.Context, string, ...string) ([]byte, error)

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// DevScanAdapter is the Mac/dev devstub: representative ports + an inactive
// firewall, so the security UI works without a Linux host.
type DevScanAdapter struct{}

func NewDevScanAdapter() *DevScanAdapter { return &DevScanAdapter{} }

func (a *DevScanAdapter) ListeningPorts(ctx context.Context) ([]ListeningPort, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []ListeningPort{
		{Protocol: "tcp", Address: "0.0.0.0", Port: 8080, Process: "higo-api", Exposure: "公开监听", Risk: "medium"},
		{Protocol: "tcp", Address: "0.0.0.0", Port: 445, Process: "smbd", Exposure: "公开监听", Risk: "medium"},
		{Protocol: "tcp", Address: "127.0.0.1", Port: 5432, Process: "postgres", Exposure: "仅本机", Risk: "low"},
		{Protocol: "udp", Address: "0.0.0.0", Port: 19999, Process: "higo-api", Exposure: "公开监听", Risk: "low"},
	}, nil
}

func (a *DevScanAdapter) Firewall(ctx context.Context) (FirewallState, error) {
	if err := ctx.Err(); err != nil {
		return FirewallState{}, err
	}
	return FirewallState{Backend: "none", Active: false, Rules: 0, Summary: "开发主机未启用防火墙（devstub）"}, nil
}
