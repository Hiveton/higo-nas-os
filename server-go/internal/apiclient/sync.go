package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// SyncList lists sync pairs (GET /api/v1/sync/pairs).
func (c *Client) SyncList(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/sync/pairs", nil, nil)
}

// SyncGet returns one sync pair (GET /api/v1/sync/pairs/{id}).
func (c *Client) SyncGet(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/sync/pairs/"+url.PathEscape(id), nil, nil)
}

// SyncCreate creates a sync pair (POST /api/v1/sync/pairs).
func (c *Client) SyncCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/sync/pairs", nil, body)
}

// SyncUpdate patches a sync pair (PUT /api/v1/sync/pairs/{id}).
func (c *Client) SyncUpdate(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/sync/pairs/"+url.PathEscape(id), nil, body)
}

// SyncDelete removes a sync pair (DELETE /api/v1/sync/pairs/{id}).
func (c *Client) SyncDelete(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/sync/pairs/"+url.PathEscape(id), nil, nil)
}

// SyncRun starts a sync run (POST /api/v1/sync/pairs/{id}/run).
func (c *Client) SyncRun(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/sync/pairs/"+url.PathEscape(id)+"/run", nil, body)
}

// SyncVerify starts a verify run (POST /api/v1/sync/pairs/{id}/verify).
func (c *Client) SyncVerify(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/sync/pairs/"+url.PathEscape(id)+"/verify", nil, body)
}

// SyncConflicts lists pending conflicts (GET /api/v1/sync/conflicts).
func (c *Client) SyncConflicts(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/sync/conflicts", nil, nil)
}

// SyncResolveConflict resolves a conflict (POST /api/v1/sync/conflicts/{id}/resolve).
func (c *Client) SyncResolveConflict(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/sync/conflicts/"+url.PathEscape(id)+"/resolve", nil, body)
}

// SyncAudit returns the sync audit log (GET /api/v1/sync/audit).
func (c *Client) SyncAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/sync/audit", nil, nil)
}
