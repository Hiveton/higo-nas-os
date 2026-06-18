package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// AssistantSemanticSearch runs a semantic search query (POST /api/v1/search/semantic).
func (c *Client) AssistantSemanticSearch(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/search/semantic", nil, body)
}

// AssistantCreateThread opens or returns the current assistant thread
// (POST /api/v1/assistant/threads).
func (c *Client) AssistantCreateThread(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/assistant/threads", nil, nil)
}

// AssistantThread fetches an assistant thread by id
// (GET /api/v1/assistant/threads/{id}).
func (c *Client) AssistantThread(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/assistant/threads/"+url.PathEscape(id), nil, nil)
}

// AssistantAddMessage posts a message to an assistant thread
// (POST /api/v1/assistant/threads/{id}/messages).
func (c *Client) AssistantAddMessage(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/assistant/threads/"+url.PathEscape(id)+"/messages", nil, body)
}

// AssistantConfirmAction confirms a pending assistant action
// (POST /api/v1/assistant/actions/{id}/confirm).
func (c *Client) AssistantConfirmAction(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/assistant/actions/"+url.PathEscape(id)+"/confirm", nil, body)
}
