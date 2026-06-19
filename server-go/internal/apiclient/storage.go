package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// StoragePools lists storage pools (GET /api/v1/storage/pools).
func (c *Client) StoragePools(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/storage/pools", nil, nil)
}

// StorageSpaces lists storage spaces (GET /api/v1/storage/spaces).
func (c *Client) StorageSpaces(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/storage/spaces", nil, nil)
}

// StorageCreateSpace creates a storage space (POST /api/v1/storage/spaces).
func (c *Client) StorageCreateSpace(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/storage/spaces", nil, body)
}

// StorageDeleteSpace deletes a storage space (DELETE /api/v1/storage/spaces/{id}).
func (c *Client) StorageDeleteSpace(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/storage/spaces/"+url.PathEscape(id), nil, body)
}

// StoragePreviewDeleteSpace previews a governed space deletion, returning a
// single-use confirmationId and impact summary
// (POST /api/v1/storage/spaces/{id}/delete/preview).
func (c *Client) StoragePreviewDeleteSpace(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/storage/spaces/"+url.PathEscape(id)+"/delete/preview", nil, body)
}

// StorageConfirmDeleteSpace executes a previously previewed space deletion
// (POST /api/v1/storage/spaces/{id}/delete/confirm).
func (c *Client) StorageConfirmDeleteSpace(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/storage/spaces/"+url.PathEscape(id)+"/delete/confirm", nil, body)
}

// StorageDisks lists storage disks (GET /api/v1/storage/disks).
func (c *Client) StorageDisks(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/storage/disks", nil, nil)
}

// StorageAddDisk adds a storage disk (POST /api/v1/storage/disks).
func (c *Client) StorageAddDisk(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/storage/disks", nil, body)
}

// StorageRemoveDisk removes a storage disk (DELETE /api/v1/storage/disks/{id}).
func (c *Client) StorageRemoveDisk(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/storage/disks/"+url.PathEscape(id), nil, body)
}

// StorageUpdateDiskSettings updates disk settings (PUT /api/v1/storage/disks/{id}/settings).
func (c *Client) StorageUpdateDiskSettings(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/storage/disks/"+url.PathEscape(id)+"/settings", nil, body)
}

// StorageSmartReports lists SMART reports (GET /api/v1/storage/smart).
func (c *Client) StorageSmartReports(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/storage/smart", nil, nil)
}

// StorageSmartScan starts a SMART scan task (POST /api/v1/storage/tasks/smart-scan).
func (c *Client) StorageSmartScan(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/storage/tasks/smart-scan", nil, body)
}

// StorageRepair starts a repair task (POST /api/v1/storage/tasks/repair).
func (c *Client) StorageRepair(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/storage/tasks/repair", nil, body)
}

// StorageSnapshot creates a snapshot task (POST /api/v1/storage/tasks/snapshot).
func (c *Client) StorageSnapshot(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/storage/tasks/snapshot", nil, body)
}

// StorageTask returns a storage task by id (GET /api/v1/storage/tasks/{id}).
func (c *Client) StorageTask(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/storage/tasks/"+url.PathEscape(id), nil, nil)
}

// StorageSnapshots lists a ZFS space's snapshots
// (GET /api/v1/storage/spaces/{id}/snapshots).
func (c *Client) StorageSnapshots(ctx context.Context, spaceID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/storage/spaces/"+url.PathEscape(spaceID)+"/snapshots", nil, nil)
}

// StorageZFSDetail returns a ZFS space's efficiency/health figures
// (GET /api/v1/storage/spaces/{id}/zfs).
func (c *Client) StorageZFSDetail(ctx context.Context, spaceID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/storage/spaces/"+url.PathEscape(spaceID)+"/zfs", nil, nil)
}

// StorageSnapshotSchedule reads a space's auto-snapshot policy
// (GET /api/v1/storage/spaces/{id}/snapshot-schedule).
func (c *Client) StorageSnapshotSchedule(ctx context.Context, spaceID string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/storage/spaces/"+url.PathEscape(spaceID)+"/snapshot-schedule", nil, nil)
}

// StorageSetSnapshotSchedule sets a space's auto-snapshot policy
// (PUT /api/v1/storage/spaces/{id}/snapshot-schedule).
func (c *Client) StorageSetSnapshotSchedule(ctx context.Context, spaceID string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/storage/spaces/"+url.PathEscape(spaceID)+"/snapshot-schedule", nil, body)
}

// StorageSnapshotRollback rolls a ZFS pool back to a snapshot
// (POST /api/v1/storage/snapshots/rollback).
func (c *Client) StorageSnapshotRollback(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/storage/snapshots/rollback", nil, body)
}
