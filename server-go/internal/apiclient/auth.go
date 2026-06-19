package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// AuthMe returns the currently authenticated user (GET /api/v1/auth/me).
func (c *Client) AuthMe(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/auth/me", nil, nil)
}

// AuthSessions lists active sessions (GET /api/v1/auth/sessions). When userId is
// non-empty it is passed as the userId query parameter (admin viewing another
// user's sessions); empty means the caller's own sessions.
func (c *Client) AuthSessions(ctx context.Context, userId string) (json.RawMessage, error) {
	var query url.Values
	if userId != "" {
		query = url.Values{"userId": {userId}}
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/auth/sessions", query, nil)
}

// AuthSessionRevoke revokes a session (DELETE /api/v1/auth/sessions/{id}).
func (c *Client) AuthSessionRevoke(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/auth/sessions/" + url.PathEscape(id)
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

// AuthAudit returns the account/auth audit trail (GET /api/v1/auth/audit, admin).
func (c *Client) AuthAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/auth/audit", nil, nil)
}
