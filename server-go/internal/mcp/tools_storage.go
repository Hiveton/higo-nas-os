package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// StorageCreateSpaceInput is the input for higo.storage.spaces.create.
// StorageDefaultSpaceInput sets the system-wide default storage space.
type StorageDefaultSpaceInput struct {
	DefaultSpaceID string `json:"defaultSpaceId" jsonschema:"id of the storage space to make default"`
}

type StorageCreateSpaceInput struct {
	Name       string   `json:"name" jsonschema:"storage space name"`
	PoolID     string   `json:"poolId" jsonschema:"storage pool to create the space on (first-level choice; disks derived from it)"`
	Mode       string   `json:"mode" jsonschema:"redundancy mode for the space"`
	FileSystem string   `json:"fileSystem" jsonschema:"file system for the space"`
	DiskSlots  []string `json:"diskSlots" jsonschema:"disk slots backing the space"`
	MountPath  string   `json:"mountPath" jsonschema:"mount path for the space"`
	Actor      string   `json:"actor" jsonschema:"actor performing the action"`
	Confirm    bool     `json:"confirm" jsonschema:"confirm the create operation"`
	FormatDisk *bool    `json:"formatDisk,omitempty" jsonschema:"whether to format backing disks"`
}

// StorageDeleteSpaceInput is the input for higo.storage.spaces.delete.
type StorageDeleteSpaceInput struct {
	ID      string `json:"id" jsonschema:"storage space id"`
	Actor   string `json:"actor" jsonschema:"actor performing the action"`
	Confirm bool   `json:"confirm" jsonschema:"confirm the delete operation"`
}

// StorageDeleteSpacePreviewInput is the input for higo.storage.spaces.delete.preview.
type StorageDeleteSpacePreviewInput struct {
	ID    string `json:"id" jsonschema:"storage space id"`
	Actor string `json:"actor" jsonschema:"actor performing the action"`
}

// StorageDeleteSpaceConfirmInput is the input for higo.storage.spaces.delete.confirm.
type StorageDeleteSpaceConfirmInput struct {
	ID             string `json:"id" jsonschema:"storage space id"`
	ConfirmationID string `json:"confirmationId" jsonschema:"single-use confirmation id from the delete preview"`
	Actor          string `json:"actor" jsonschema:"actor performing the action"`
}

// StorageAddDiskInput is the input for higo.storage.disks.add.
type StorageAddDiskInput struct {
	Name      string `json:"name" jsonschema:"disk name"`
	MountPath string `json:"mountPath" jsonschema:"disk mount path"`
	Role      string `json:"role" jsonschema:"disk role"`
	Actor     string `json:"actor" jsonschema:"actor performing the action"`
}

// StorageRemoveDiskInput is the input for higo.storage.disks.remove.
type StorageRemoveDiskInput struct {
	ID      string `json:"id" jsonschema:"disk id"`
	Actor   string `json:"actor" jsonschema:"actor performing the action"`
	Confirm bool   `json:"confirm" jsonschema:"confirm the remove operation"`
}

// StorageDiskSettingsInput is the input for higo.storage.disks.settings.
type StorageDiskSettingsInput struct {
	ID             string `json:"id" jsonschema:"disk id"`
	StandbyMinutes int    `json:"standbyMinutes" jsonschema:"idle minutes before spindown"`
	SSDCache       bool   `json:"ssdCache" jsonschema:"enable SSD cache"`
	CacheMode      string `json:"cacheMode" jsonschema:"cache mode"`
	Actor          string `json:"actor" jsonschema:"actor performing the action"`
}

// StorageTaskTargetInput is the input for storage maintenance tasks.
type StorageTaskTargetInput struct {
	TargetSlot string `json:"targetSlot,omitempty" jsonschema:"target disk slot"`
	TargetPool string `json:"targetPool,omitempty" jsonschema:"target pool id"`
}

// StorageTaskInput is the input for higo.storage.tasks.get.
type StorageTaskInput struct {
	ID string `json:"id" jsonschema:"storage task id"`
}

// StorageSnapshotsInput is the input for higo.storage.snapshots.list.
type StorageSnapshotsInput struct {
	SpaceID string `json:"spaceId" jsonschema:"storage space (pool) id"`
}

// StorageSnapshotRollbackInput is the input for higo.storage.snapshots.rollback.
type StorageSnapshotRollbackInput struct {
	Snapshot string `json:"snapshot" jsonschema:"snapshot name to roll back to (pool@snapshot)"`
}

// StorageSnapshotScheduleInput is the input for higo.storage.snapshots.schedule.set.
type StorageSnapshotScheduleInput struct {
	SpaceID       string `json:"spaceId" jsonschema:"storage space (pool) id"`
	Enabled       bool   `json:"enabled" jsonschema:"whether automatic snapshots are enabled"`
	IntervalHours int    `json:"intervalHours" jsonschema:"hours between automatic snapshots"`
	Keep          int    `json:"keep" jsonschema:"number of snapshots to retain"`
}

func registerStorage(r *registry) {
	addTool(r, "storage", "higo.storage.pools.list",
		"List storage pools with capacity and health.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.StoragePools(ctx)
		})

	addTool(r, "storage", "higo.storage.spaces.list",
		"List storage spaces.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.StorageSpaces(ctx)
		})

	addTool(r, "storage", "higo.storage.default-space.get",
		"Get the system-wide default storage space.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.StorageDefaultSpace(ctx)
		})

	addTool(r, "storage", "higo.storage.default-space.set",
		"Set the system-wide default storage space.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StorageDefaultSpaceInput) (json.RawMessage, error) {
			return c.StorageSetDefaultSpace(ctx, map[string]any{"defaultSpaceId": in.DefaultSpaceID})
		})

	addTool(r, "storage", "higo.storage.spaces.create",
		"Create a storage space across one or more disks.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StorageCreateSpaceInput) (json.RawMessage, error) {
			return c.StorageCreateSpace(ctx, in)
		})

	addTool(r, "storage", "higo.storage.spaces.delete",
		"Delete a storage space (HIGH RISK; destroys data on the space).",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in StorageDeleteSpaceInput) (json.RawMessage, error) {
			return c.StorageDeleteSpace(ctx, in.ID, map[string]any{"actor": in.Actor, "confirm": in.Confirm})
		})

	addTool(r, "storage", "higo.storage.spaces.delete.preview",
		"Preview deleting a storage space: returns a single-use confirmationId and impact summary, with no side effect.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StorageDeleteSpacePreviewInput) (json.RawMessage, error) {
			return c.StoragePreviewDeleteSpace(ctx, in.ID, map[string]any{"actor": in.Actor})
		})

	addTool(r, "storage", "higo.storage.spaces.delete.confirm",
		"Confirm a previewed storage space deletion (HIGH RISK; destroys data). Requires the confirmationId from the preview.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in StorageDeleteSpaceConfirmInput) (json.RawMessage, error) {
			return c.StorageConfirmDeleteSpace(ctx, in.ID, map[string]any{"confirmationId": in.ConfirmationID, "actor": in.Actor})
		})

	addTool(r, "storage", "higo.storage.disks.list",
		"List physical storage disks.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.StorageDisks(ctx)
		})

	addTool(r, "storage", "higo.storage.disks.add",
		"Add a storage disk.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StorageAddDiskInput) (json.RawMessage, error) {
			return c.StorageAddDisk(ctx, in)
		})

	addTool(r, "storage", "higo.storage.disks.remove",
		"Remove a storage disk (HIGH RISK; may destroy data).",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in StorageRemoveDiskInput) (json.RawMessage, error) {
			return c.StorageRemoveDisk(ctx, in.ID, map[string]any{"actor": in.Actor, "confirm": in.Confirm})
		})

	addTool(r, "storage", "higo.storage.disks.settings",
		"Update disk settings such as spindown and SSD cache.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StorageDiskSettingsInput) (json.RawMessage, error) {
			return c.StorageUpdateDiskSettings(ctx, in.ID, map[string]any{
				"standbyMinutes": in.StandbyMinutes,
				"ssdCache":       in.SSDCache,
				"cacheMode":      in.CacheMode,
				"actor":          in.Actor,
			})
		})

	addTool(r, "storage", "higo.storage.smart.list",
		"List SMART health reports for disks.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.StorageSmartReports(ctx)
		})

	addTool(r, "storage", "higo.storage.tasks.smart-scan",
		"Start a SMART scan task.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StorageTaskTargetInput) (json.RawMessage, error) {
			return c.StorageSmartScan(ctx, in)
		})

	addTool(r, "storage", "higo.storage.tasks.repair",
		"Start a storage repair task (HIGH RISK; may rebuild or destroy data).",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in StorageTaskTargetInput) (json.RawMessage, error) {
			return c.StorageRepair(ctx, in)
		})

	addTool(r, "storage", "higo.storage.tasks.snapshot",
		"Create a storage snapshot task.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StorageTaskTargetInput) (json.RawMessage, error) {
			return c.StorageSnapshot(ctx, in)
		})

	addTool(r, "storage", "higo.storage.tasks.get",
		"Get the status of a storage task by id.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in StorageTaskInput) (json.RawMessage, error) {
			return c.StorageTask(ctx, in.ID)
		})

	addTool(r, "storage", "higo.storage.snapshots.list",
		"List the ZFS snapshots of a storage space (pool).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in StorageSnapshotsInput) (json.RawMessage, error) {
			return c.StorageSnapshots(ctx, in.SpaceID)
		})

	addTool(r, "storage", "higo.storage.zfs.detail",
		"Get a ZFS storage space's efficiency/health figures (compression, fragmentation, dedup, capacity).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in StorageSnapshotsInput) (json.RawMessage, error) {
			return c.StorageZFSDetail(ctx, in.SpaceID)
		})

	addTool(r, "storage", "higo.storage.snapshots.schedule.get",
		"Get a ZFS space's automatic-snapshot policy.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in StorageSnapshotsInput) (json.RawMessage, error) {
			return c.StorageSnapshotSchedule(ctx, in.SpaceID)
		})

	addTool(r, "storage", "higo.storage.snapshots.schedule.set",
		"Configure a ZFS space's automatic-snapshot policy (interval + retention).",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StorageSnapshotScheduleInput) (json.RawMessage, error) {
			return c.StorageSetSnapshotSchedule(ctx, in.SpaceID, map[string]any{
				"enabled":       in.Enabled,
				"intervalHours": in.IntervalHours,
				"keep":          in.Keep,
			})
		})

	addTool(r, "storage", "higo.storage.snapshots.rollback",
		"Roll a managed ZFS pool back to a snapshot (destructive: discards changes since the snapshot).",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in StorageSnapshotRollbackInput) (json.RawMessage, error) {
			return c.StorageSnapshotRollback(ctx, in)
		})
}
