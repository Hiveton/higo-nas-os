package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// AIProviders lists configured AI model providers (GET /api/v1/ai/providers).
func (c *Client) AIProviders(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/ai/providers", nil, nil)
}

// AIProviderCreate registers a new AI model provider (POST /api/v1/ai/providers).
func (c *Client) AIProviderCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/ai/providers", nil, body)
}

// AIProviderUpdate updates an AI model provider (PUT /api/v1/ai/providers/{id}).
func (c *Client) AIProviderUpdate(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/ai/providers/"+url.PathEscape(id), nil, body)
}

// AIProviderDelete removes an AI model provider (DELETE /api/v1/ai/providers/{id}).
func (c *Client) AIProviderDelete(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/ai/providers/"+url.PathEscape(id), nil, nil)
}

// AIProviderTest sends a tiny prompt to verify a provider's binding works
// (POST /api/v1/ai/providers/{id}/test).
func (c *Client) AIProviderTest(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/ai/providers/"+url.PathEscape(id)+"/test", nil, nil)
}
