package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// NetworkConfigInput previews a host network configuration change. Changing the
// address is high risk, so this only previews — apply with the confirm tool.
type NetworkConfigInput struct {
	Interface string   `json:"interface,omitempty" jsonschema:"Interface name, e.g. eth0 (defaults to the primary interface)"`
	Mode      string   `json:"mode" jsonschema:"Address mode: dhcp or static"`
	Address   string   `json:"address,omitempty" jsonschema:"Static IPv4 address (required when mode=static)"`
	Prefix    int      `json:"prefix,omitempty" jsonschema:"Static network prefix length 1-32 (required when mode=static)"`
	Gateway   string   `json:"gateway,omitempty" jsonschema:"Default gateway IPv4 address"`
	DNS       []string `json:"dns,omitempty" jsonschema:"DNS server IP addresses"`
	Hostname  string   `json:"hostname,omitempty" jsonschema:"Host name to set"`
	Actor     string   `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// NetworkConfirmInput applies a previewed network change by confirmation id.
type NetworkConfirmInput struct {
	ConfirmationID string `json:"confirmationId" jsonschema:"Confirmation id returned by the config preview"`
	Actor          string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// NetworkRollbackInput reverses a confirmed network change by its audit id.
type NetworkRollbackInput struct {
	ID     string `json:"id" jsonschema:"Audit entry id to roll back"`
	Actor  string `json:"actor,omitempty" jsonschema:"Actor performing the rollback for audit"`
	Reason string `json:"reason,omitempty" jsonschema:"Reason for the rollback"`
}

func registerNetwork(r *registry) {
	addTool(r, "network", "higo.network.interfaces.list",
		"List host network interfaces with their addresses and link state.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.NetworkInterfaces(ctx)
		})

	addTool(r, "network", "higo.network.config.get",
		"Get the current effective host network configuration.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.NetworkConfig(ctx)
		})

	addTool(r, "network", "higo.network.config.preview",
		"Preview a host network configuration change (returns impact, risk and a confirmation id). Does not apply.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in NetworkConfigInput) (json.RawMessage, error) {
			return c.NetworkConfigPreview(ctx, in)
		})

	addTool(r, "network", "higo.network.config.confirm",
		"Apply a previewed host network change by confirmation id. High risk: may change the device IP and drop connections.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in NetworkConfirmInput) (json.RawMessage, error) {
			return c.NetworkConfigConfirm(ctx, in)
		})

	addTool(r, "network", "higo.network.audit.list",
		"List the network governance audit log (newest first).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.NetworkAudit(ctx)
		})

	addTool(r, "network", "higo.network.audit.rollback",
		"Roll back a confirmed network change by its audit id, re-applying the previous configuration.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in NetworkRollbackInput) (json.RawMessage, error) {
			return c.NetworkAuditRollback(ctx, in.ID, in)
		})
}
