package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// AccountsSummary returns users, groups and grants (GET /api/v1/accounts/summary).
func (c *Client) AccountsSummary(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/accounts/summary", nil, nil)
}

// AccountUsers lists user accounts (GET /api/v1/accounts/users).
func (c *Client) AccountUsers(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/accounts/users", nil, nil)
}

// AccountUserCreate creates a user account (POST /api/v1/accounts/users).
func (c *Client) AccountUserCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/accounts/users", nil, body)
}

// AccountUserUpdate updates a user account (PUT /api/v1/accounts/users/{id}).
func (c *Client) AccountUserUpdate(ctx context.Context, id string, body any) (json.RawMessage, error) {
	path := "/api/v1/accounts/users/" + url.PathEscape(id)
	return c.Do(ctx, http.MethodPut, path, nil, body)
}

// AccountUserDelete deletes a user account (DELETE /api/v1/accounts/users/{id}).
func (c *Client) AccountUserDelete(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/accounts/users/" + url.PathEscape(id)
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

// AccountGroups lists user groups (GET /api/v1/accounts/groups).
func (c *Client) AccountGroups(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/accounts/groups", nil, nil)
}

// AccountGroupCreate creates a user group (POST /api/v1/accounts/groups).
func (c *Client) AccountGroupCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/accounts/groups", nil, body)
}

// AccountGroupMembersUpdate replaces a group's members
// (PUT /api/v1/accounts/groups/{id}/members).
func (c *Client) AccountGroupMembersUpdate(ctx context.Context, id string, body any) (json.RawMessage, error) {
	path := "/api/v1/accounts/groups/" + url.PathEscape(id) + "/members"
	return c.Do(ctx, http.MethodPut, path, nil, body)
}

// AccountGrantCreate grants space access to a subject (POST /api/v1/accounts/grants).
func (c *Client) AccountGrantCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/accounts/grants", nil, body)
}

// AccountGrantDelete revokes a space access grant
// (DELETE /api/v1/accounts/grants/{id}).
func (c *Client) AccountGrantDelete(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/accounts/grants/" + url.PathEscape(id)
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}
