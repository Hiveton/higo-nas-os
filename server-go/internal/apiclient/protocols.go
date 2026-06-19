package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// ProtocolsList lists every sharing protocol with live state (GET /api/v1/protocols).
func (c *Client) ProtocolsList(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/protocols", nil, nil)
}

// ProtocolGet returns a single protocol (GET /api/v1/protocols/{key}).
func (c *Client) ProtocolGet(ctx context.Context, key string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/protocols/"+url.PathEscape(key), nil, nil)
}

// ProtocolUpdateConfig updates one protocol's settings (PUT /api/v1/protocols/{key}/config).
func (c *Client) ProtocolUpdateConfig(ctx context.Context, key string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/protocols/"+url.PathEscape(key)+"/config", nil, body)
}

// ProtocolShares lists shares for one protocol (GET /api/v1/protocols/{key}/shares).
func (c *Client) ProtocolShares(ctx context.Context, key string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/protocols/"+url.PathEscape(key)+"/shares", nil, nil)
}

// ProtocolsShares lists every share across protocols (GET /api/v1/protocols/shares).
func (c *Client) ProtocolsShares(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/protocols/shares", nil, nil)
}

// ProtocolsAudit returns the governance audit log (GET /api/v1/protocols/audit).
func (c *Client) ProtocolsAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/protocols/audit", nil, nil)
}

// ProtocolEnablePreview previews enabling a protocol (POST .../{key}/enable/preview).
func (c *Client) ProtocolEnablePreview(ctx context.Context, key string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/"+url.PathEscape(key)+"/enable/preview", nil, body)
}

// ProtocolDisablePreview previews disabling a protocol (POST .../{key}/disable/preview).
func (c *Client) ProtocolDisablePreview(ctx context.Context, key string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/"+url.PathEscape(key)+"/disable/preview", nil, body)
}

// ProtocolEnableConfirm applies a previewed enable (POST .../{key}/enable/confirm).
func (c *Client) ProtocolEnableConfirm(ctx context.Context, key string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/"+url.PathEscape(key)+"/enable/confirm", nil, body)
}

// ProtocolDisableConfirm applies a previewed disable (POST .../{key}/disable/confirm).
func (c *Client) ProtocolDisableConfirm(ctx context.Context, key string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/"+url.PathEscape(key)+"/disable/confirm", nil, body)
}

// ProtocolSharePreview previews creating a share (POST .../{key}/shares/preview).
func (c *Client) ProtocolSharePreview(ctx context.Context, key string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/"+url.PathEscape(key)+"/shares/preview", nil, body)
}

// ProtocolShareConfirm applies a previewed share create (POST .../{key}/shares/confirm).
func (c *Client) ProtocolShareConfirm(ctx context.Context, key string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/"+url.PathEscape(key)+"/shares/confirm", nil, body)
}

// ProtocolShareDeletePreview previews deleting a share (POST /protocols/shares/{id}/delete/preview).
func (c *Client) ProtocolShareDeletePreview(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/shares/"+url.PathEscape(id)+"/delete/preview", nil, body)
}

// ProtocolShareDeleteConfirm applies a previewed share delete (POST /protocols/shares/{id}/delete/confirm).
func (c *Client) ProtocolShareDeleteConfirm(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/shares/"+url.PathEscape(id)+"/delete/confirm", nil, body)
}

// ProtocolAuditRollback reverses a confirmed change (POST /protocols/audit/{id}/rollback).
func (c *Client) ProtocolAuditRollback(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/protocols/audit/"+url.PathEscape(id)+"/rollback", nil, body)
}
