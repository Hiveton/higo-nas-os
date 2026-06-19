package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// ProtocolKeyInput identifies one sharing protocol.
type ProtocolKeyInput struct {
	Key string `json:"key" jsonschema:"Protocol key: smb, nfs, webdav or dlna"`
}

// ProtocolActorInput is a protocol action carrying the audit actor.
type ProtocolActorInput struct {
	Key   string `json:"key" jsonschema:"Protocol key: smb, nfs, webdav or dlna"`
	Actor string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// ProtocolConfirmInput applies a previewed protocol change by confirmation id.
type ProtocolConfirmInput struct {
	Key            string `json:"key" jsonschema:"Protocol key the change belongs to"`
	ConfirmationID string `json:"confirmationId" jsonschema:"Confirmation id returned by the matching preview"`
	Actor          string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// ProtocolSharePreviewInput previews sharing a directory over a protocol.
type ProtocolSharePreviewInput struct {
	Key          string   `json:"key" jsonschema:"Protocol key to share over"`
	Name         string   `json:"name" jsonschema:"Display name of the share"`
	Path         string   `json:"path" jsonschema:"Directory path to share (must be under the NAS root)"`
	AccessLevel  string   `json:"accessLevel" jsonschema:"Access level: public, password, account or readonly"`
	AllowedUsers []string `json:"allowedUsers,omitempty" jsonschema:"NAS users granted access (for account/password)"`
	Guest        bool     `json:"guest,omitempty" jsonschema:"Whether guest access is allowed"`
	Actor        string   `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// ProtocolShareIDInput identifies a single share.
type ProtocolShareIDInput struct {
	ID    string `json:"id" jsonschema:"Share id"`
	Actor string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// ProtocolShareConfirmInput applies a previewed share change by confirmation id.
type ProtocolShareConfirmInput struct {
	ID             string `json:"id" jsonschema:"Share id the change belongs to"`
	ConfirmationID string `json:"confirmationId" jsonschema:"Confirmation id returned by the matching preview"`
	Actor          string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// ProtocolConfigUpdateInput updates one protocol's settings directly.
type ProtocolConfigUpdateInput struct {
	Key            string `json:"key" jsonschema:"Protocol key: smb, nfs, webdav or dlna"`
	ServerName     string `json:"serverName,omitempty" jsonschema:"SMB server display name"`
	Workgroup      string `json:"workgroup,omitempty" jsonschema:"SMB workgroup"`
	MinProtocol    string `json:"minProtocol,omitempty" jsonschema:"SMB minimum protocol: SMB2 or SMB3"`
	GuestAccess    bool   `json:"guestAccess,omitempty" jsonschema:"SMB allow guest access"`
	Squash         string `json:"squash,omitempty" jsonschema:"NFS squash: root_squash, all_squash or no_root_squash"`
	AllowedNetwork string `json:"allowedNetwork,omitempty" jsonschema:"NFS allowed network CIDR or *"`
	HTTPSEnabled   bool   `json:"httpsEnabled,omitempty" jsonschema:"WebDAV enable HTTPS"`
	FriendlyName   string `json:"friendlyName,omitempty" jsonschema:"DLNA device name shown to clients"`
	Actor          string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// ProtocolAuditRollbackInput reverses a confirmed change by its audit id.
type ProtocolAuditRollbackInput struct {
	ID     string `json:"id" jsonschema:"Audit entry id to roll back"`
	Actor  string `json:"actor,omitempty" jsonschema:"Actor performing the rollback for audit"`
	Reason string `json:"reason,omitempty" jsonschema:"Reason for the rollback"`
}

func registerProtocols(r *registry) {
	addTool(r, "protocols", "higo.protocols.list",
		"List network sharing protocols (SMB/NFS/WebDAV/DLNA) with live running state.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in noInput) (json.RawMessage, error) {
			return c.ProtocolsList(ctx)
		})

	addTool(r, "protocols", "higo.protocols.get",
		"Get a single sharing protocol by key.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolKeyInput) (json.RawMessage, error) {
			return c.ProtocolGet(ctx, in.Key)
		})

	addTool(r, "protocols", "higo.protocols.shares.list",
		"List every shared directory across all protocols.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in noInput) (json.RawMessage, error) {
			return c.ProtocolsShares(ctx)
		})

	addTool(r, "protocols", "higo.protocols.audit.list",
		"List the protocols governance audit log.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in noInput) (json.RawMessage, error) {
			return c.ProtocolsAudit(ctx)
		})

	addTool(r, "protocols", "higo.protocols.enable.preview",
		"Preview enabling a protocol (returns impact + confirmationId, no side effects).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolActorInput) (json.RawMessage, error) {
			return c.ProtocolEnablePreview(ctx, in.Key, in)
		})

	addTool(r, "protocols", "higo.protocols.enable.confirm",
		"Confirm enabling a protocol using a confirmationId from the preview.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolConfirmInput) (json.RawMessage, error) {
			return c.ProtocolEnableConfirm(ctx, in.Key, in)
		})

	addTool(r, "protocols", "higo.protocols.disable.preview",
		"Preview disabling a protocol (returns impact + confirmationId, no side effects).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolActorInput) (json.RawMessage, error) {
			return c.ProtocolDisablePreview(ctx, in.Key, in)
		})

	addTool(r, "protocols", "higo.protocols.disable.confirm",
		"Confirm disabling a protocol using a confirmationId from the preview.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolConfirmInput) (json.RawMessage, error) {
			return c.ProtocolDisableConfirm(ctx, in.Key, in)
		})

	addTool(r, "protocols", "higo.protocols.share.preview",
		"Preview sharing a directory over a protocol (returns impact + confirmationId).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolSharePreviewInput) (json.RawMessage, error) {
			return c.ProtocolSharePreview(ctx, in.Key, in)
		})

	addTool(r, "protocols", "higo.protocols.share.confirm",
		"Confirm creating a share using a confirmationId from the preview.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolConfirmInput) (json.RawMessage, error) {
			return c.ProtocolShareConfirm(ctx, in.Key, in)
		})

	addTool(r, "protocols", "higo.protocols.share.delete.preview",
		"Preview removing a shared directory by id (returns impact + confirmationId).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolShareIDInput) (json.RawMessage, error) {
			return c.ProtocolShareDeletePreview(ctx, in.ID, in)
		})

	addTool(r, "protocols", "higo.protocols.share.delete.confirm",
		"Confirm removing a shared directory using a confirmationId from the preview.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolShareConfirmInput) (json.RawMessage, error) {
			return c.ProtocolShareDeleteConfirm(ctx, in.ID, in)
		})

	addTool(r, "protocols", "higo.protocols.config.update",
		"Update one protocol's settings (workgroup, squash, device name, etc.). Direct, audited, rollbackable.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolConfigUpdateInput) (json.RawMessage, error) {
			body := map[string]any{
				"actor": in.Actor,
				"config": map[string]any{
					"serverName": in.ServerName, "workgroup": in.Workgroup, "minProtocol": in.MinProtocol,
					"guestAccess": in.GuestAccess, "squash": in.Squash, "allowedNetwork": in.AllowedNetwork,
					"httpsEnabled": in.HTTPSEnabled, "friendlyName": in.FriendlyName,
				},
			}
			return c.ProtocolUpdateConfig(ctx, in.Key, body)
		})

	addTool(r, "protocols", "higo.protocols.audit.rollback",
		"Roll back a confirmed protocol change by its audit id.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in ProtocolAuditRollbackInput) (json.RawMessage, error) {
			return c.ProtocolAuditRollback(ctx, in.ID, in)
		})
}
