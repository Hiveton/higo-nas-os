package protocols

import "context"

// DevAdapter is the deterministic, optimistic devstub used off-Linux. All
// mutators succeed; Status echoes the desired state and reports every protocol
// as installed, with Running tracking the desired Enabled toggle.
type DevAdapter struct{}

func NewDevAdapter() *DevAdapter { return &DevAdapter{} }

func (a *DevAdapter) Status(ctx context.Context, desired []Protocol) ([]Protocol, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := cloneProtocols(desired)
	for i := range out {
		out[i].Installed = true
		out[i].Running = out[i].Enabled
	}
	return out, nil
}

func (a *DevAdapter) EnableProtocol(ctx context.Context, key ProtocolKey, cfg ProtocolBaseConfig) error {
	return ctx.Err()
}

func (a *DevAdapter) DisableProtocol(ctx context.Context, key ProtocolKey) error {
	return ctx.Err()
}

func (a *DevAdapter) Apply(ctx context.Context, key ProtocolKey, config ProtocolConfig, shares []Share) error {
	return ctx.Err()
}
