package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
)

// SystemInfo returns system identity/version info (GET /api/v1/system/info).
func (c *Client) SystemInfo(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/system/info", nil, nil)
}

// SystemUpdates returns the current update status (GET /api/v1/system/updates).
func (c *Client) SystemUpdates(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/system/updates", nil, nil)
}

// SystemUpdateCheck triggers an update check (POST /api/v1/system/updates/check).
func (c *Client) SystemUpdateCheck(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/system/updates/check", nil, nil)
}

// SystemBackup creates a system backup (POST /api/v1/system/backups).
func (c *Client) SystemBackup(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/system/backups", nil, body)
}
