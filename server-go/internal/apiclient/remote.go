package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// RemoteStatus returns remote access status (GET /api/v1/remote/status).
func (c *Client) RemoteStatus(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/remote/status", nil, nil)
}

// RemoteStartChannel starts the remote access channel (POST /api/v1/remote/channel/start).
func (c *Client) RemoteStartChannel(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/remote/channel/start", nil, nil)
}

// RemoteStopChannel stops the remote access channel (POST /api/v1/remote/channel/stop).
func (c *Client) RemoteStopChannel(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/remote/channel/stop", nil, nil)
}

// RemoteSetTunnelMode updates the remote tunnel mode (PUT /api/v1/remote/tunnel-mode).
func (c *Client) RemoteSetTunnelMode(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/remote/tunnel-mode", nil, body)
}

// RemoteSetMFA toggles remote MFA (PUT /api/v1/remote/mfa).
func (c *Client) RemoteSetMFA(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/remote/mfa", nil, body)
}

// RemoteSetPolicy selects the remote access policy (PUT /api/v1/remote/policy).
func (c *Client) RemoteSetPolicy(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/remote/policy", nil, body)
}

// RemoteCreateDomainToken creates a remote domain token (POST /api/v1/remote/domain-token).
func (c *Client) RemoteCreateDomainToken(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/remote/domain-token", nil, nil)
}

// RemoteRotateDomainToken rotates the remote domain token (POST /api/v1/remote/domain-token/rotate).
func (c *Client) RemoteRotateDomainToken(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/remote/domain-token/rotate", nil, nil)
}

// RemoteDevices lists bound remote devices (GET /api/v1/remote/devices).
func (c *Client) RemoteDevices(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/remote/devices", nil, nil)
}

// RemoteBindDevice binds a remote device (POST /api/v1/remote/devices/{id}/bind).
func (c *Client) RemoteBindDevice(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/remote/devices/"+url.PathEscape(id)+"/bind", nil, nil)
}

// RemoteUnbindDevice unbinds a remote device (POST /api/v1/remote/devices/{id}/unbind).
func (c *Client) RemoteUnbindDevice(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/remote/devices/"+url.PathEscape(id)+"/unbind", nil, nil)
}

// RemoteLoginAlerts lists remote login alerts (GET /api/v1/remote/login-alerts).
func (c *Client) RemoteLoginAlerts(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/remote/login-alerts", nil, nil)
}

// RemoteShareScan scans remote share links (POST /api/v1/remote/share-scan).
func (c *Client) RemoteShareScan(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/remote/share-scan", nil, nil)
}
