package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
)

// DesktopApps returns the registered desktop apps (GET /api/v1/desktop/apps).
func (c *Client) DesktopApps(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/desktop/apps", nil, nil)
}

// DesktopWindows returns the open desktop windows (GET /api/v1/desktop/windows).
func (c *Client) DesktopWindows(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/desktop/windows", nil, nil)
}

// DesktopSession returns the current desktop session (GET /api/v1/desktop/session).
func (c *Client) DesktopSession(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/desktop/session", nil, nil)
}

// DesktopUpdateSession patches the desktop session (PUT /api/v1/desktop/session).
func (c *Client) DesktopUpdateSession(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/desktop/session", nil, body)
}
