package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// SemanticSearchInput is the body for a cross-content semantic search query.
type SemanticSearchInput struct {
	Query  string   `json:"query" jsonschema:"natural language search query"`
	Scopes []string `json:"scopes,omitempty" jsonschema:"spaces/scopes to restrict the search to"`
	Limit  int      `json:"limit,omitempty" jsonschema:"maximum number of results to return"`
}

// registerSearch exposes the AI semantic search under a top-level "search"
// domain so the in-app assistant (which excludes the assistant domain to avoid
// recursion) can still query the real pgvector index.
func registerSearch(r *registry) {
	addTool(r, "search", "higo.search.semantic",
		"Semantic search across indexed files, photos, audio and video. Use this to find documents or media by meaning, not just keywords.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in SemanticSearchInput) (json.RawMessage, error) {
			return c.AssistantSemanticSearch(ctx, in)
		})
}
