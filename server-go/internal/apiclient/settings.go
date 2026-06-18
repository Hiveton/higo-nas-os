package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
)

// SettingsGet returns the current settings (GET /api/v1/settings).
func (c *Client) SettingsGet(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/settings", nil, nil)
}

// SettingsUpdate replaces the settings (PUT /api/v1/settings).
func (c *Client) SettingsUpdate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/settings", nil, body)
}

// SettingsRestoreDefaults restores default settings (POST /api/v1/settings/defaults).
func (c *Client) SettingsRestoreDefaults(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/settings/defaults", nil, nil)
}
