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

// AppCenterCatalog lists every installable manifest across all sources.
func (c *Client) AppCenterCatalog(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/app-center/catalog", nil, nil)
}

// AppCenterCatalogItem fetches one catalog entry (including its manifest).
func (c *Client) AppCenterCatalogItem(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/app-center/catalog/"+url.PathEscape(id), nil, nil)
}

// AppCenterCatalogRefresh re-fetches the configured remote registries.
func (c *Client) AppCenterCatalogRefresh(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/app-center/catalog/refresh", nil, nil)
}

// AppCenterPreview previews a governed lifecycle action and returns its
// confirmationId + impact (no side effect). action is install|update|start|
// stop|uninstall.
func (c *Client) AppCenterPreview(ctx context.Context, id, action string, config map[string]string) (json.RawMessage, error) {
	path := "/api/v1/app-center/apps/" + url.PathEscape(id) + "/" + url.PathEscape(action)
	var body any
	if len(config) > 0 {
		body = map[string]any{"config": config}
	}
	return c.Do(ctx, http.MethodPost, path, nil, body)
}

// AppCenterConfirm executes a previously-previewed action.
func (c *Client) AppCenterConfirm(ctx context.Context, id, action, confirmationID, actor string, config map[string]string) (json.RawMessage, error) {
	path := "/api/v1/app-center/apps/" + url.PathEscape(id) + "/" + url.PathEscape(action) + "/confirm"
	body := map[string]any{"confirmationId": confirmationID, "actor": actor}
	if len(config) > 0 {
		body["config"] = config
	}
	return c.Do(ctx, http.MethodPost, path, nil, body)
}

// AppCenterAudit lists the app-center audit log.
func (c *Client) AppCenterAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/app-center/audit", nil, nil)
}

// AppCenterRollback reverses a previously-confirmed action.
func (c *Client) AppCenterRollback(ctx context.Context, auditID, actor string) (json.RawMessage, error) {
	path := "/api/v1/app-center/audit/" + url.PathEscape(auditID) + "/rollback"
	return c.Do(ctx, http.MethodPost, path, nil, map[string]any{"actor": actor})
}

// AppCenterRegistries lists the configured remote registry sources.
func (c *Client) AppCenterRegistries(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/app-center/registries", nil, nil)
}

// AppCenterAddRegistry registers (or re-enables) a remote registry source.
func (c *Client) AppCenterAddRegistry(ctx context.Context, name, registryURL string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/app-center/registries", nil, map[string]any{"name": name, "url": registryURL})
}

// AppCenterRemoveRegistry drops a remote registry source.
func (c *Client) AppCenterRemoveRegistry(ctx context.Context, name string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/app-center/registries/"+url.PathEscape(name), nil, nil)
}
