package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// VMList lists virtual machines (GET /api/v1/vm/machines).
func (c *Client) VMList(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/vm/machines", nil, nil)
}

// VMGet returns one VM (GET /api/v1/vm/machines/{name}).
func (c *Client) VMGet(ctx context.Context, name string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/vm/machines/"+url.PathEscape(name), nil, nil)
}

// VMCapabilities reports host hypervisor capabilities (GET /api/v1/vm/capabilities).
func (c *Client) VMCapabilities(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/vm/capabilities", nil, nil)
}

// VMAudit returns the VM action log (GET /api/v1/vm/audit).
func (c *Client) VMAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/vm/audit", nil, nil)
}

// VMAction applies a lifecycle action (POST /api/v1/vm/machines/{name}/{action}).
func (c *Client) VMAction(ctx context.Context, name, action string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/vm/machines/"+url.PathEscape(name)+"/"+url.PathEscape(action), nil, body)
}
