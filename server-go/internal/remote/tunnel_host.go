package remote

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// WireGuardAdapter manages a real WireGuard interface on a Linux host. The NAS
// runs as a WireGuard server: enabling the tunnel brings up the `higoos`
// interface listening on a UDP port with a persistent server keypair; clients
// peer to it using the server public key + endpoint. Every command goes through
// the injectable runner so the parsing is unit-testable off-Linux.
type WireGuardAdapter struct {
	runner   commandRunner
	confPath string
}

func NewWireGuardAdapter() *WireGuardAdapter { return NewWireGuardAdapterWithRunner(runCommand) }

func NewWireGuardAdapterWithRunner(runner commandRunner) *WireGuardAdapter {
	if runner == nil {
		runner = runCommand
	}
	return &WireGuardAdapter{runner: runner, confPath: "/etc/wireguard/" + tunnelIface + ".conf"}
}

// Up ensures a persistent server config exists, then brings the interface up.
func (a *WireGuardAdapter) Up(ctx context.Context) (TunnelInfo, error) {
	// Already up? Return live state (idempotent).
	if info, err := a.Status(ctx); err == nil && info.Up {
		return info, nil
	}
	if err := a.ensureConfig(ctx); err != nil {
		return TunnelInfo{}, err
	}
	if _, err := a.runner(ctx, "wg-quick", "up", tunnelIface); err != nil {
		return TunnelInfo{}, fmt.Errorf("wg-quick up: %w", err)
	}
	return a.Status(ctx)
}

func (a *WireGuardAdapter) Down(ctx context.Context) error {
	if _, err := a.runner(ctx, "wg-quick", "down", tunnelIface); err != nil {
		// Already down is not an error worth surfacing.
		if strings.Contains(err.Error(), "is not a WireGuard") || strings.Contains(err.Error(), "does not exist") {
			return nil
		}
		return fmt.Errorf("wg-quick down: %w", err)
	}
	return nil
}

func (a *WireGuardAdapter) Status(ctx context.Context) (TunnelInfo, error) {
	info := TunnelInfo{Backend: "wireguard", Interface: tunnelIface, Address: tunnelAddress}
	pub, err := a.runner(ctx, "wg", "show", tunnelIface, "public-key")
	if err != nil {
		// Interface not up.
		info.Up = false
		return info, nil
	}
	info.Up = true
	info.PublicKey = strings.TrimSpace(string(pub))
	if port, err := a.runner(ctx, "wg", "show", tunnelIface, "listen-port"); err == nil {
		info.ListenPort, _ = strconv.Atoi(strings.TrimSpace(string(port)))
	}
	if peers, err := a.runner(ctx, "wg", "show", tunnelIface, "peers"); err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(peers)), "\n") {
			if strings.TrimSpace(line) != "" {
				info.Peers++
			}
		}
	}
	return info, nil
}

// ensureConfig writes /etc/wireguard/higoos.conf with a freshly-generated server
// key the first time; subsequent calls keep the existing key so client peerings
// stay valid.
func (a *WireGuardAdapter) ensureConfig(ctx context.Context) error {
	if _, err := os.Stat(a.confPath); err == nil {
		return nil // keep the existing key
	}
	if err := os.MkdirAll(filepath.Dir(a.confPath), 0o700); err != nil {
		return fmt.Errorf("create wireguard dir: %w", err)
	}
	priv, err := a.runner(ctx, "wg", "genkey")
	if err != nil {
		return fmt.Errorf("wg genkey: %w", err)
	}
	conf := fmt.Sprintf("[Interface]\nPrivateKey = %s\nAddress = %s\nListenPort = %d\n",
		strings.TrimSpace(string(priv)), tunnelAddress, tunnelPort)
	if err := os.WriteFile(a.confPath, []byte(conf), 0o600); err != nil {
		return fmt.Errorf("write wireguard config: %w", err)
	}
	return nil
}
