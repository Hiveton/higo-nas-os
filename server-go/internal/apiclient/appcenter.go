package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// AppCenterApps lists available and installed apps (GET /api/v1/app-center/apps).
func (c *Client) AppCenterApps(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/app-center/apps", nil, nil)
}

// AppCenterAppInstall installs an app (POST /api/v1/app-center/apps/{id}/install).
func (c *Client) AppCenterAppInstall(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/app-center/apps/" + url.PathEscape(id) + "/install"
	return c.Do(ctx, http.MethodPost, path, nil, nil)
}

// AppCenterAppUpdate updates an installed app (POST /api/v1/app-center/apps/{id}/update).
func (c *Client) AppCenterAppUpdate(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/app-center/apps/" + url.PathEscape(id) + "/update"
	return c.Do(ctx, http.MethodPost, path, nil, nil)
}

// AppCenterAppStart starts an installed app (POST /api/v1/app-center/apps/{id}/start).
func (c *Client) AppCenterAppStart(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/app-center/apps/" + url.PathEscape(id) + "/start"
	return c.Do(ctx, http.MethodPost, path, nil, nil)
}

// AppCenterAppStop stops an installed app (POST /api/v1/app-center/apps/{id}/stop).
func (c *Client) AppCenterAppStop(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/app-center/apps/" + url.PathEscape(id) + "/stop"
	return c.Do(ctx, http.MethodPost, path, nil, nil)
}
