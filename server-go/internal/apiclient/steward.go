package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// StewardSuggestions lists steward suggestions (GET /api/v1/steward/suggestions).
func (c *Client) StewardSuggestions(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/steward/suggestions", nil, nil)
}

// StewardPreviewSuggestion previews a steward suggestion
// (POST /api/v1/steward/suggestions/{id}/preview).
func (c *Client) StewardPreviewSuggestion(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/steward/suggestions/"+url.PathEscape(id)+"/preview", nil, body)
}

// StewardConfirmSuggestion confirms a steward suggestion
// (POST /api/v1/steward/suggestions/{id}/confirm).
func (c *Client) StewardConfirmSuggestion(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/steward/suggestions/"+url.PathEscape(id)+"/confirm", nil, body)
}

// StewardDismissSuggestion dismisses a steward suggestion
// (POST /api/v1/steward/suggestions/{id}/dismiss).
func (c *Client) StewardDismissSuggestion(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/steward/suggestions/"+url.PathEscape(id)+"/dismiss", nil, body)
}

// StewardAudit lists steward audit entries (GET /api/v1/steward/audit).
func (c *Client) StewardAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/steward/audit", nil, nil)
}

// StewardRollbackAudit rolls back a steward audit entry
// (POST /api/v1/steward/audit/{id}/rollback).
func (c *Client) StewardRollbackAudit(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/steward/audit/"+url.PathEscape(id)+"/rollback", nil, body)
}
