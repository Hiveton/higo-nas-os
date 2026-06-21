package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// SyncIDInput identifies a sync pair.
type SyncIDInput struct {
	ID    string `json:"id" jsonschema:"Sync pair id"`
	Actor string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// SyncCreateInput creates a sync pair.
type SyncCreateInput struct {
	Name           string   `json:"name" jsonschema:"Display name of the sync task"`
	Source         string   `json:"source" jsonschema:"Source directory (under the NAS root)"`
	Target         string   `json:"target" jsonschema:"Target directory (under the NAS root)"`
	Direction      string   `json:"direction,omitempty" jsonschema:"mirror (one-way) or two-way"`
	ConflictPolicy string   `json:"conflictPolicy,omitempty" jsonschema:"newer, source, target or manual"`
	Includes       []string `json:"includes,omitempty" jsonschema:"Optional glob patterns for selective sync"`
	BandwidthLimit string   `json:"bandwidthLimit,omitempty" jsonschema:"Advisory bandwidth cap, e.g. 10 MB/s"`
	IntervalHours  int      `json:"intervalHours,omitempty" jsonschema:"Auto-run interval in hours, 0 = manual"`
	Actor          string   `json:"actor,omitempty"`
}

// SyncUpdateInput patches a sync pair (omitted fields unchanged).
type SyncUpdateInput struct {
	ID             string `json:"id" jsonschema:"Sync pair id"`
	Name           string `json:"name,omitempty"`
	Direction      string `json:"direction,omitempty"`
	ConflictPolicy string `json:"conflictPolicy,omitempty"`
	BandwidthLimit string `json:"bandwidthLimit,omitempty"`
	Enabled        *bool  `json:"enabled,omitempty"`
	IntervalHours  *int   `json:"intervalHours,omitempty"`
	Actor          string `json:"actor,omitempty"`
}

// SyncResolveInput resolves a conflict by choosing a side.
type SyncResolveInput struct {
	ID    string `json:"id" jsonschema:"Conflict id"`
	Side  string `json:"side" jsonschema:"Which side wins: source or target"`
	Actor string `json:"actor,omitempty"`
}

func registerSync(r *registry) {
	addTool(r, "sync", "higo.sync.list",
		"List folder synchronization pairs with their last-run state.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in noInput) (json.RawMessage, error) {
			return c.SyncList(ctx)
		})

	addTool(r, "sync", "higo.sync.get",
		"Get a single sync pair by id.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in SyncIDInput) (json.RawMessage, error) {
			return c.SyncGet(ctx, in.ID)
		})

	addTool(r, "sync", "higo.sync.conflicts.list",
		"List pending two-way sync conflicts.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in noInput) (json.RawMessage, error) {
			return c.SyncConflicts(ctx)
		})

	addTool(r, "sync", "higo.sync.audit.list",
		"List the sync audit log.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in noInput) (json.RawMessage, error) {
			return c.SyncAudit(ctx)
		})

	addTool(r, "sync", "higo.sync.create",
		"Create a folder sync pair (mirror or two-way).",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SyncCreateInput) (json.RawMessage, error) {
			return c.SyncCreate(ctx, in)
		})

	addTool(r, "sync", "higo.sync.update",
		"Update a sync pair's settings.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SyncUpdateInput) (json.RawMessage, error) {
			return c.SyncUpdate(ctx, in.ID, in)
		})

	addTool(r, "sync", "higo.sync.run",
		"Start a sync run for a pair (real file synchronization).",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SyncIDInput) (json.RawMessage, error) {
			return c.SyncRun(ctx, in.ID, map[string]any{"actor": in.Actor})
		})

	addTool(r, "sync", "higo.sync.verify",
		"Verify a pair's target matches its source by content hash.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in SyncIDInput) (json.RawMessage, error) {
			return c.SyncVerify(ctx, in.ID, map[string]any{"actor": in.Actor})
		})

	addTool(r, "sync", "higo.sync.conflict.resolve",
		"Resolve a sync conflict by choosing the source or target side.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SyncResolveInput) (json.RawMessage, error) {
			return c.SyncResolveConflict(ctx, in.ID, map[string]any{"side": in.Side, "actor": in.Actor})
		})

	addTool(r, "sync", "higo.sync.delete",
		"Delete a sync pair (config only; synced files are left in place).",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in SyncIDInput) (json.RawMessage, error) {
			return c.SyncDelete(ctx, in.ID)
		})
}
