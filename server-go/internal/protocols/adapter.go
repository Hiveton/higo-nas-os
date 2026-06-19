package protocols

import (
	"context"
	"os/exec"
)

// commandRunner abstracts shelling out so tests can inject a fake. Mirrors the
// storage host adapter's runner.
type commandRunner func(context.Context, string, ...string) ([]byte, error)

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// Adapter abstracts the host protocol stack. HostAdapter drives real systemd
// services and HiGoOS-owned config files on Linux; DevAdapter is an in-memory
// devstub used on dev hosts.
//
// Apply regenerates the entire HiGoOS-owned config for one protocol from its
// full desired state (settings + share list) — an atomic whole-file rewrite, so
// changing a setting, or adding/removing a share, is the same call with
// different inputs. This avoids config drift and partial-edit corruption.
type Adapter interface {
	// Status reconciles the desired protocols against the host, filling in the
	// live Running/Installed flags.
	Status(ctx context.Context, desired []Protocol) ([]Protocol, error)
	// EnableProtocol writes base config + enables/starts the systemd unit(s).
	EnableProtocol(ctx context.Context, key ProtocolKey, cfg ProtocolBaseConfig) error
	// DisableProtocol stops/disables the unit(s).
	DisableProtocol(ctx context.Context, key ProtocolKey) error
	// Apply regenerates the managed config for one protocol from its settings +
	// full desired share list and reloads the service.
	Apply(ctx context.Context, key ProtocolKey, config ProtocolConfig, shares []Share) error
}
