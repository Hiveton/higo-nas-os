package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// AssistantSearchInput is the body for a semantic search query.
type AssistantSearchInput struct {
	ActorID string   `json:"actorId,omitempty" jsonschema:"id of the actor running the search"`
	Query   string   `json:"query" jsonschema:"natural language search query"`
	Scopes  []string `json:"scopes,omitempty" jsonschema:"scopes to restrict the search to"`
	Limit   int      `json:"limit,omitempty" jsonschema:"maximum number of results to return"`
}

// AssistantThreadInput identifies an assistant thread.
type AssistantThreadInput struct {
	ID string `json:"id" jsonschema:"assistant thread id"`
}

// AssistantAddMessageInput is the body for posting a message to a thread.
type AssistantAddMessageInput struct {
	ID          string   `json:"id" jsonschema:"assistant thread id to post into"`
	ActorID     string   `json:"actorId,omitempty" jsonschema:"id of the actor sending the message"`
	Role        string   `json:"role,omitempty" jsonschema:"message role (user or assistant)"`
	Text        string   `json:"text" jsonschema:"message text content"`
	Scopes      []string `json:"scopes,omitempty" jsonschema:"scopes granted for this message"`
	ModelPolicy string   `json:"modelPolicy,omitempty" jsonschema:"model policy to apply to the response"`
}

// AssistantConfirmActionInput is the body for confirming a pending action.
type AssistantConfirmActionInput struct {
	ID      string `json:"id" jsonschema:"pending assistant action id to confirm"`
	ActorID string `json:"actorId,omitempty" jsonschema:"id of the actor confirming the action"`
	Intent  string `json:"intent,omitempty" jsonschema:"intent describing the confirmed action"`
}

func registerAssistant(r *registry) {
	addTool(r, "assistant", "higo.assistant.search.semantic",
		"Run a semantic search across the assistant-indexed content.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in AssistantSearchInput) (json.RawMessage, error) {
			return c.AssistantSemanticSearch(ctx, in)
		})

	addTool(r, "assistant", "higo.assistant.threads.create",
		"Open or return the current assistant thread.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AssistantCreateThread(ctx)
		})

	addTool(r, "assistant", "higo.assistant.threads.get",
		"Get an assistant thread and its messages by id.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in AssistantThreadInput) (json.RawMessage, error) {
			return c.AssistantThread(ctx, in.ID)
		})

	addTool(r, "assistant", "higo.assistant.threads.messages.add",
		"Post a message to an assistant thread and get the assistant reply.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AssistantAddMessageInput) (json.RawMessage, error) {
			return c.AssistantAddMessage(ctx, in.ID, map[string]any{
				"actorId":     in.ActorID,
				"role":        in.Role,
				"text":        in.Text,
				"scopes":      in.Scopes,
				"modelPolicy": in.ModelPolicy,
			})
		})

	addTool(r, "assistant", "higo.assistant.actions.confirm",
		"Confirm a previously-returned pending assistant action by its confirmationId.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AssistantConfirmActionInput) (json.RawMessage, error) {
			return c.AssistantConfirmAction(ctx, in.ID, map[string]any{
				"actorId": in.ActorID,
				"intent":  in.Intent,
			})
		})
}
