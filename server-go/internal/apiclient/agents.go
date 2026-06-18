package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// AgentTemplates lists agent templates (GET /api/v1/agents/templates).
func (c *Client) AgentTemplates(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/agents/templates", nil, nil)
}

// AgentCreate instantiates an agent from a template (POST /api/v1/agents).
func (c *Client) AgentCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/agents", nil, body)
}

// AgentTools lists the tools available to an agent (GET /api/v1/agents/{id}/tools).
func (c *Client) AgentTools(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/agents/"+url.PathEscape(id)+"/tools", nil, nil)
}

// WorkflowPreview previews a workflow plan (POST /api/v1/workflows/preview).
func (c *Client) WorkflowPreview(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/workflows/preview", nil, body)
}

// WorkflowStartRun starts a workflow run (POST /api/v1/workflows/runs).
func (c *Client) WorkflowStartRun(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/workflows/runs", nil, body)
}

// WorkflowConfirmRun confirms a pending workflow run
// (POST /api/v1/workflows/runs/{id}/confirm).
func (c *Client) WorkflowConfirmRun(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/workflows/runs/"+url.PathEscape(id)+"/confirm", nil, body)
}

// WorkflowCancelRun cancels a workflow run (POST /api/v1/workflows/runs/{id}/cancel).
func (c *Client) WorkflowCancelRun(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/workflows/runs/"+url.PathEscape(id)+"/cancel", nil, body)
}

// WorkflowRunEvents returns the current events for a workflow run
// (GET /api/v1/workflows/runs/{id}/events).
func (c *Client) WorkflowRunEvents(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/workflows/runs/"+url.PathEscape(id)+"/events", nil, nil)
}
