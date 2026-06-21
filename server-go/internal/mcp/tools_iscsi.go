package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// ISCSICreateInput creates an iSCSI target (IQN auto-generated when empty).
type ISCSICreateInput struct {
	IQN   string `json:"iqn,omitempty" jsonschema:"Target IQN; leave empty to auto-generate"`
	Actor string `json:"actor,omitempty"`
}

// ISCSITargetInput identifies a target.
type ISCSITargetInput struct {
	IQN   string `json:"iqn" jsonschema:"Target IQN"`
	Actor string `json:"actor,omitempty"`
}

// ISCSILUNInput adds a fileio LUN to a target.
type ISCSILUNInput struct {
	IQN    string `json:"iqn" jsonschema:"Target IQN"`
	Name   string `json:"name" jsonschema:"Backstore/LUN name"`
	SizeMB int    `json:"sizeMB" jsonschema:"LUN size in MB"`
	Actor  string `json:"actor,omitempty"`
}

// ISCSIACLInput authorizes an initiator on a target.
type ISCSIACLInput struct {
	IQN       string `json:"iqn" jsonschema:"Target IQN"`
	Initiator string `json:"initiator" jsonschema:"Initiator IQN to authorize"`
	Actor     string `json:"actor,omitempty"`
}

func registerISCSI(r *registry) {
	addTool(r, "iscsi", "higo.iscsi.targets.list",
		"List iSCSI targets with their LUN count, ACLs and portals.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.ISCSITargets(ctx)
		})

	addTool(r, "iscsi", "higo.iscsi.capabilities",
		"Report whether the host LIO/targetcli iSCSI stack is available.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.ISCSICapabilities(ctx)
		})

	addTool(r, "iscsi", "higo.iscsi.audit.list",
		"List the iSCSI action audit log.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.ISCSIAudit(ctx)
		})

	addTool(r, "iscsi", "higo.iscsi.target.create",
		"Create an iSCSI target (exports block storage — high risk).",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in ISCSICreateInput) (json.RawMessage, error) {
			return c.ISCSICreateTarget(ctx, in)
		})

	addTool(r, "iscsi", "higo.iscsi.target.delete",
		"Delete an iSCSI target and its mappings.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in ISCSITargetInput) (json.RawMessage, error) {
			return c.ISCSIDeleteTarget(ctx, in.IQN)
		})

	addTool(r, "iscsi", "higo.iscsi.lun.add",
		"Add a fileio-backed LUN to a target.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in ISCSILUNInput) (json.RawMessage, error) {
			return c.ISCSIAddLUN(ctx, in.IQN, in)
		})

	addTool(r, "iscsi", "higo.iscsi.acl.add",
		"Authorize an initiator IQN on a target.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in ISCSIACLInput) (json.RawMessage, error) {
			return c.ISCSIAddACL(ctx, in.IQN, in)
		})
}
