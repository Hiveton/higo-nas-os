package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// Tasks lists background tasks from the task runtime (GET /api/v1/tasks).
// When kind is non-empty it filters by task kind.
func (c *Client) Tasks(ctx context.Context, kind string) (json.RawMessage, error) {
	var query url.Values
	if kind != "" {
		query = url.Values{"kind": {kind}}
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/tasks", query, nil)
}

// Task gets a single background task by id (GET /api/v1/tasks/{id}).
func (c *Client) Task(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/tasks/"+url.PathEscape(id), nil, nil)
}

// TaskCancel cancels a queued background task (POST /api/v1/tasks/{id}/cancel).
func (c *Client) TaskCancel(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/tasks/"+url.PathEscape(id)+"/cancel", nil, nil)
}
