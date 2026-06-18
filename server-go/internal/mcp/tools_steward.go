package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// StewardPreviewSuggestionInput is the body for previewing a suggestion.
type StewardPreviewSuggestionInput struct {
	ID      string `json:"id" jsonschema:"steward suggestion id to preview"`
	ActorID string `json:"actorId,omitempty" jsonschema:"id of the actor requesting the preview"`
}

// StewardConfirmSuggestionInput is the body for confirming a suggestion.
type StewardConfirmSuggestionInput struct {
	ID             string `json:"id" jsonschema:"steward suggestion id to confirm"`
	ActorID        string `json:"actorId,omitempty" jsonschema:"id of the actor confirming the suggestion"`
	ConfirmationID string `json:"confirmationId,omitempty" jsonschema:"confirmationId returned by the preview"`
}

// StewardDismissSuggestionInput is the body for dismissing a suggestion.
type StewardDismissSuggestionInput struct {
	ID      string `json:"id" jsonschema:"steward suggestion id to dismiss"`
	ActorID string `json:"actorId,omitempty" jsonschema:"id of the actor dismissing the suggestion"`
	Reason  string `json:"reason,omitempty" jsonschema:"reason for dismissing the suggestion"`
}

// StewardRollbackAuditInput is the body for rolling back an audit entry.
type StewardRollbackAuditInput struct {
	ID      string `json:"id" jsonschema:"steward audit entry id to roll back"`
	ActorID string `json:"actorId,omitempty" jsonschema:"id of the actor performing the rollback"`
	Reason  string `json:"reason,omitempty" jsonschema:"reason for the rollback"`
}

func registerSteward(r *registry) {
	addTool(r, "steward", "higo.steward.suggestions.list",
		"List steward maintenance suggestions.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.StewardSuggestions(ctx)
		})

	addTool(r, "steward", "higo.steward.suggestions.refresh",
		"Re-analyze the file tree (duplicates, large/stale files) and regenerate maintenance suggestions.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.StewardRefresh(ctx)
		})

	addTool(r, "steward", "higo.steward.suggestions.preview",
		"Preview the impact of a steward suggestion before confirming it.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StewardPreviewSuggestionInput) (json.RawMessage, error) {
			return c.StewardPreviewSuggestion(ctx, in.ID, map[string]any{
				"actorId": in.ActorID,
			})
		})

	addTool(r, "steward", "higo.steward.suggestions.confirm",
		"Confirm a previously-previewed steward suggestion by its confirmationId.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StewardConfirmSuggestionInput) (json.RawMessage, error) {
			return c.StewardConfirmSuggestion(ctx, in.ID, map[string]any{
				"actorId":        in.ActorID,
				"confirmationId": in.ConfirmationID,
			})
		})

	addTool(r, "steward", "higo.steward.suggestions.dismiss",
		"Dismiss a steward suggestion.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StewardDismissSuggestionInput) (json.RawMessage, error) {
			return c.StewardDismissSuggestion(ctx, in.ID, map[string]any{
				"actorId": in.ActorID,
				"reason":  in.Reason,
			})
		})

	addTool(r, "steward", "higo.steward.audit.list",
		"List steward audit entries.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.StewardAudit(ctx)
		})

	addTool(r, "steward", "higo.steward.audit.rollback",
		"Roll back a previously-executed steward audit entry.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in StewardRollbackAuditInput) (json.RawMessage, error) {
			return c.StewardRollbackAudit(ctx, in.ID, map[string]any{
				"actorId": in.ActorID,
				"reason":  in.Reason,
			})
		})
}
