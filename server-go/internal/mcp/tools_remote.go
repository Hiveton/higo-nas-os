package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// RemoteTunnelModeInput is the input for higo.remote.tunnel-mode.set.
type RemoteTunnelModeInput struct {
	Mode string `json:"mode" jsonschema:"tunnel mode to apply"`
}

// RemoteMFAInput is the input for higo.remote.mfa.set.
type RemoteMFAInput struct {
	Enabled bool `json:"enabled" jsonschema:"whether MFA is enabled"`
}

// RemotePolicyInput is the input for higo.remote.policy.set.
type RemotePolicyInput struct {
	Key string `json:"key" jsonschema:"policy key to select"`
}

// RemoteDeviceInput is the input for remote device bind/unbind actions.
type RemoteDeviceInput struct {
	ID string `json:"id" jsonschema:"remote device id"`
}

func registerRemote(r *registry) {
	addTool(r, "remote", "higo.remote.status",
		"Get remote access status including channel, tunnel and policy state.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.RemoteStatus(ctx)
		})

	addTool(r, "remote", "higo.remote.channel.start",
		"Start the remote access channel.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.RemoteStartChannel(ctx)
		})

	addTool(r, "remote", "higo.remote.channel.stop",
		"Stop the remote access channel.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.RemoteStopChannel(ctx)
		})

	addTool(r, "remote", "higo.remote.tunnel-mode.set",
		"Set the remote tunnel mode.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in RemoteTunnelModeInput) (json.RawMessage, error) {
			return c.RemoteSetTunnelMode(ctx, map[string]any{"mode": in.Mode})
		})

	addTool(r, "remote", "higo.remote.mfa.set",
		"Enable or disable remote access MFA.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in RemoteMFAInput) (json.RawMessage, error) {
			return c.RemoteSetMFA(ctx, map[string]any{"enabled": in.Enabled})
		})

	addTool(r, "remote", "higo.remote.policy.set",
		"Select the active remote access policy.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in RemotePolicyInput) (json.RawMessage, error) {
			return c.RemoteSetPolicy(ctx, map[string]any{"key": in.Key})
		})

	addTool(r, "remote", "higo.remote.domain-token.create",
		"Create a remote domain token.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.RemoteCreateDomainToken(ctx)
		})

	addTool(r, "remote", "higo.remote.domain-token.rotate",
		"Rotate the remote domain token.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.RemoteRotateDomainToken(ctx)
		})

	addTool(r, "remote", "higo.remote.devices.list",
		"List bound remote devices.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.RemoteDevices(ctx)
		})

	addTool(r, "remote", "higo.remote.devices.bind",
		"Bind a remote device.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in RemoteDeviceInput) (json.RawMessage, error) {
			return c.RemoteBindDevice(ctx, in.ID)
		})

	addTool(r, "remote", "higo.remote.devices.unbind",
		"Unbind a remote device.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in RemoteDeviceInput) (json.RawMessage, error) {
			return c.RemoteUnbindDevice(ctx, in.ID)
		})

	addTool(r, "remote", "higo.remote.login-alerts.list",
		"List remote login alerts.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.RemoteLoginAlerts(ctx)
		})

	addTool(r, "remote", "higo.remote.share-scan",
		"Scan remote share links for risks.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.RemoteShareScan(ctx)
		})
}
