package remote

import (
	"context"
	"os/exec"
)

// TunnelInfo is the live state of the remote-access tunnel. PublicKey + Address
// + ListenPort are what a client needs to build its peer config.
type TunnelInfo struct {
	Backend    string `json:"backend"` // wireguard / devstub / none
	Up         bool   `json:"up"`
	Interface  string `json:"interface"`
	PublicKey  string `json:"publicKey"`
	ListenPort int    `json:"listenPort"`
	Address    string `json:"address"`
	Peers      int    `json:"peers"`
	Note       string `json:"note,omitempty"`
}

// tunnelAdapter brings a real remote-access tunnel up/down. The WireGuard host
// adapter manages a wg interface on Linux; the devstub fakes it elsewhere.
type tunnelAdapter interface {
	Up(ctx context.Context) (TunnelInfo, error)
	Down(ctx context.Context) error
	Status(ctx context.Context) (TunnelInfo, error)
}

type commandRunner func(context.Context, string, ...string) ([]byte, error)

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

const (
	tunnelIface   = "higoos"
	tunnelPort    = 51820
	tunnelAddress = "10.66.66.1/24"
)

// DevTunnelAdapter is the Mac/dev devstub: an in-memory tunnel that flips up/down
// optimistically with a deterministic fake public key.
type DevTunnelAdapter struct{ up bool }

func NewDevTunnelAdapter() *DevTunnelAdapter { return &DevTunnelAdapter{} }

func (a *DevTunnelAdapter) Up(ctx context.Context) (TunnelInfo, error) {
	if err := ctx.Err(); err != nil {
		return TunnelInfo{}, err
	}
	a.up = true
	return a.info(), nil
}

func (a *DevTunnelAdapter) Down(ctx context.Context) error {
	a.up = false
	return ctx.Err()
}

func (a *DevTunnelAdapter) Status(ctx context.Context) (TunnelInfo, error) {
	if err := ctx.Err(); err != nil {
		return TunnelInfo{}, err
	}
	return a.info(), nil
}

func (a *DevTunnelAdapter) info() TunnelInfo {
	return TunnelInfo{
		Backend:    "devstub",
		Up:         a.up,
		Interface:  tunnelIface,
		PublicKey:  "DEVSTUBpublicKEY0000000000000000000000000000=",
		ListenPort: tunnelPort,
		Address:    tunnelAddress,
		Note:       "开发主机模拟隧道（devstub）",
	}
}
