// Package llm binds HiGoOS to standard LLM providers. It owns the provider
// configuration store (persisted as JSON state, API keys masked when read back)
// and a small set of streaming chat clients that speak the native wire format of
// each supported provider kind.
//
// Scope is chat completion only: callers send a list of ChatMessages and receive
// streamed StreamChunks. Tool/function calling is intentionally not modeled here
// yet — the Client interface leaves room to add it later without changing callers.
package llm

import "time"

// ProviderKind selects which wire protocol a provider speaks.
type ProviderKind string

const (
	// KindOpenAI is the OpenAI /chat/completions standard. It also covers Azure
	// OpenAI, DeepSeek, Qwen, Ollama, LM Studio, vLLM and any other server that
	// implements the same surface, via a configurable BaseURL.
	KindOpenAI ProviderKind = "openai"
	// KindAnthropic is the Anthropic Messages API (/v1/messages).
	KindAnthropic ProviderKind = "anthropic"
	// KindGemini is the Google Gemini generateContent API.
	KindGemini ProviderKind = "gemini"
)

// Provider is a configured, bindable model endpoint. APIKey is stored verbatim on
// disk but never serialized to clients (see ProviderView).
type Provider struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Kind      ProviderKind `json:"kind"`
	BaseURL   string       `json:"baseUrl"`
	APIKey    string       `json:"apiKey,omitempty"`
	Model     string       `json:"model"`
	Enabled   bool         `json:"enabled"`
	IsDefault bool         `json:"isDefault"`
	CreatedAt time.Time    `json:"createdAt"`
}

// ProviderView is the client-facing projection of a Provider. It omits the raw
// APIKey and exposes only a KeyHint (last 4 characters) plus whether a key is set.
type ProviderView struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Kind      ProviderKind `json:"kind"`
	BaseURL   string       `json:"baseUrl"`
	Model     string       `json:"model"`
	Enabled   bool         `json:"enabled"`
	IsDefault bool         `json:"isDefault"`
	HasKey    bool         `json:"hasKey"`
	KeyHint   string       `json:"keyHint,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
}

// ProviderInput is the mutable subset accepted on create/update. A nil pointer
// means "leave unchanged" on update; an empty APIKey string also preserves the
// existing key so the masked view can be saved back without wiping the secret.
type ProviderInput struct {
	Name      *string       `json:"name,omitempty"`
	Kind      *ProviderKind `json:"kind,omitempty"`
	BaseURL   *string       `json:"baseUrl,omitempty"`
	APIKey    *string       `json:"apiKey,omitempty"`
	Model     *string       `json:"model,omitempty"`
	Enabled   *bool         `json:"enabled,omitempty"`
	IsDefault *bool         `json:"isDefault,omitempty"`
}

// ChatMessage is one turn in a conversation. Role is "system" | "user" |
// "assistant" | "tool". ToolCalls is set on assistant turns that request tools;
// ToolCallID is set on tool-result turns to link them to the originating call.
type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"toolCalls,omitempty"`
	ToolCallID string     `json:"toolCallId,omitempty"`
}

// ToolDef describes a function the model may call. Parameters is a JSON Schema
// object.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// ToolCall is a function invocation requested by the model. Arguments is the raw
// JSON-encoded argument object.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ChatRequest is the provider-agnostic chat completion request. Model overrides
// the provider's default model when non-empty.
type ChatRequest struct {
	Model       string        `json:"model,omitempty"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"maxTokens,omitempty"`
}

// StreamChunk is one incremental piece of a streamed completion. Exactly one of
// Delta (more text) or Done (stream finished) is meaningful per chunk.
type StreamChunk struct {
	Delta string `json:"delta,omitempty"`
	Done  bool   `json:"done,omitempty"`
}

// toView builds the masked, client-facing projection of a provider.
func (p Provider) toView() ProviderView {
	return ProviderView{
		ID:        p.ID,
		Name:      p.Name,
		Kind:      p.Kind,
		BaseURL:   p.BaseURL,
		Model:     p.Model,
		Enabled:   p.Enabled,
		IsDefault: p.IsDefault,
		HasKey:    p.APIKey != "",
		KeyHint:   maskKey(p.APIKey),
		CreatedAt: p.CreatedAt,
	}
}

// maskKey returns the last 4 characters of a secret, prefixed with an ellipsis,
// or "" when there is no key.
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "····"
	}
	return "····" + key[len(key)-4:]
}
