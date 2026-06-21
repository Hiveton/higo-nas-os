package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// ISCSITargets lists iSCSI targets (GET /api/v1/iscsi/targets).
func (c *Client) ISCSITargets(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/iscsi/targets", nil, nil)
}

// ISCSICapabilities reports the host iSCSI stack (GET /api/v1/iscsi/capabilities).
func (c *Client) ISCSICapabilities(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/iscsi/capabilities", nil, nil)
}

// ISCSIAudit returns the iSCSI action log (GET /api/v1/iscsi/audit).
func (c *Client) ISCSIAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/iscsi/audit", nil, nil)
}

// ISCSICreateTarget creates a target (POST /api/v1/iscsi/targets).
func (c *Client) ISCSICreateTarget(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/iscsi/targets", nil, body)
}

// ISCSIDeleteTarget removes a target (DELETE /api/v1/iscsi/targets/{iqn}).
func (c *Client) ISCSIDeleteTarget(ctx context.Context, iqn string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/iscsi/targets/"+url.PathEscape(iqn), nil, nil)
}

// ISCSIAddLUN adds a LUN to a target (POST /api/v1/iscsi/targets/{iqn}/luns).
func (c *Client) ISCSIAddLUN(ctx context.Context, iqn string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/iscsi/targets/"+url.PathEscape(iqn)+"/luns", nil, body)
}

// ISCSIAddACL authorizes an initiator (POST /api/v1/iscsi/targets/{iqn}/acls).
func (c *Client) ISCSIAddACL(ctx context.Context, iqn string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/iscsi/targets/"+url.PathEscape(iqn)+"/acls", nil, body)
}
