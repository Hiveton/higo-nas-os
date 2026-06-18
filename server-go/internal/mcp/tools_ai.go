package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// AIProviderCreateInput is the input for higo.ai.providers.create.
type AIProviderCreateInput struct {
	Name      string `json:"name" jsonschema:"display name for the provider"`
	Kind      string `json:"kind" jsonschema:"wire protocol: openai, anthropic, or gemini"`
	Model     string `json:"model" jsonschema:"model id, e.g. gpt-4o or claude-sonnet-4-6"`
	BaseURL   string `json:"baseUrl,omitempty" jsonschema:"override the API base URL (optional)"`
	APIKey    string `json:"apiKey,omitempty" jsonschema:"API key or token (optional)"`
	Enabled   *bool  `json:"enabled,omitempty" jsonschema:"whether the provider is enabled"`
	IsDefault *bool  `json:"isDefault,omitempty" jsonschema:"set this provider as the default"`
}

// AIProviderUpdateInput is the input for higo.ai.providers.update. Only the
// fields you set are changed (partial update); id selects the target.
type AIProviderUpdateInput struct {
	ID        string `json:"id" jsonschema:"id of the provider to update"`
	Name      string `json:"name,omitempty" jsonschema:"new display name"`
	Kind      string `json:"kind,omitempty" jsonschema:"wire protocol: openai, anthropic, or gemini"`
	Model     string `json:"model,omitempty" jsonschema:"new model id"`
	BaseURL   string `json:"baseUrl,omitempty" jsonschema:"new API base URL"`
	APIKey    string `json:"apiKey,omitempty" jsonschema:"new API key or token"`
	Enabled   *bool  `json:"enabled,omitempty" jsonschema:"enable or disable the provider"`
	IsDefault *bool  `json:"isDefault,omitempty" jsonschema:"set or unset as the default provider"`
}

// AIProviderIDInput is the input for provider actions keyed only by id.
type AIProviderIDInput struct {
	ID string `json:"id" jsonschema:"provider id"`
}

// aiProviderBody collects only the supplied fields into a partial-update body,
// matching llm.ProviderInput. The backend rejects unknown fields, so id (a path
// parameter) is never included here.
func aiProviderBody(name, kind, model, baseURL, apiKey string, enabled, isDefault *bool) map[string]any {
	body := map[string]any{}
	if name != "" {
		body["name"] = name
	}
	if kind != "" {
		body["kind"] = kind
	}
	if model != "" {
		body["model"] = model
	}
	if baseURL != "" {
		body["baseUrl"] = baseURL
	}
	if apiKey != "" {
		body["apiKey"] = apiKey
	}
	if enabled != nil {
		body["enabled"] = *enabled
	}
	if isDefault != nil {
		body["isDefault"] = *isDefault
	}
	return body
}

func registerAI(r *registry) {
	addTool(r, "ai", "higo.ai.providers.list",
		"List configured AI model providers (OpenAI/Anthropic/Gemini-compatible), including which is default and whether each has a key.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AIProviders(ctx)
		})

	addTool(r, "ai", "higo.ai.providers.create",
		"Register a new AI model provider. Requires name, kind (openai/anthropic/gemini) and model.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AIProviderCreateInput) (json.RawMessage, error) {
			return c.AIProviderCreate(ctx, aiProviderBody(in.Name, in.Kind, in.Model, in.BaseURL, in.APIKey, in.Enabled, in.IsDefault))
		})

	addTool(r, "ai", "higo.ai.providers.update",
		"Update an AI model provider. Only the fields you set are changed.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AIProviderUpdateInput) (json.RawMessage, error) {
			return c.AIProviderUpdate(ctx, in.ID, aiProviderBody(in.Name, in.Kind, in.Model, in.BaseURL, in.APIKey, in.Enabled, in.IsDefault))
		})

	addTool(r, "ai", "higo.ai.providers.delete",
		"Delete an AI model provider by id.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in AIProviderIDInput) (json.RawMessage, error) {
			return c.AIProviderDelete(ctx, in.ID)
		})

	addTool(r, "ai", "higo.ai.providers.test",
		"Send a tiny prompt to verify a provider's binding works; returns model, reply and latency. Makes a real external model call.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AIProviderIDInput) (json.RawMessage, error) {
			return c.AIProviderTest(ctx, in.ID)
		})
}
