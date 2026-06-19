package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// SystemIdentity returns the unauthenticated device fingerprint
// (GET /api/v1/system/identity).
func (c *Client) SystemIdentity(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/system/identity", nil, nil)
}

// NetworkInterfaces lists network interfaces (GET /api/v1/network/interfaces).
func (c *Client) NetworkInterfaces(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/network/interfaces", nil, nil)
}

// NetworkConfig returns the current effective config (GET /api/v1/network/config).
func (c *Client) NetworkConfig(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/network/config", nil, nil)
}

// NetworkConfigPreview previews a configuration change (PUT /api/v1/network/config).
func (c *Client) NetworkConfigPreview(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/network/config", nil, body)
}

// NetworkConfigConfirm applies a previewed change (POST /api/v1/network/config/confirm).
func (c *Client) NetworkConfigConfirm(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/network/config/confirm", nil, body)
}

// NetworkAudit returns the governance audit log (GET /api/v1/network/audit).
func (c *Client) NetworkAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/network/audit", nil, nil)
}

// NetworkAuditRollback reverses a confirmed change (POST /api/v1/network/audit/{id}/rollback).
func (c *Client) NetworkAuditRollback(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/network/audit/"+url.PathEscape(id)+"/rollback", nil, body)
}
