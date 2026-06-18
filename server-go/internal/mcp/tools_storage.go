package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// StorageCreateSpaceInput is the input for higo.storage.spaces.create.
type StorageCreateSpaceInput struct {
	Name       string   `json:"name" jsonschema:"storage space name"`
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
}
