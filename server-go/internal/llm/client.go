package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client streams a chat completion from one provider kind. emit is called once
// per incremental delta and once more with Done=true when the stream finishes.
// Implementations must not call emit after returning.
type Client interface {
	Stream(ctx context.Context, p Provider, req ChatRequest, emit func(StreamChunk)) error
}

// NewClient returns the streaming client for a provider kind.
func NewClient(kind ProviderKind) (Client, error) {
	switch kind {
	case KindOpenAI:
		return openAIClient{}, nil
	case KindAnthropic:
		return anthropicClient{}, nil
	case KindGemini:
		return geminiClient{}, nil
	default:
		return nil, fmt.Errorf("unsupported llm provider kind: %s", kind)
	}
}

// Factory resolves a Client for a provider kind. assistant.Service depends on
// this rather than NewClient directly so tests can inject a fake.
type Factory func(kind ProviderKind) (Client, error)

// Complete runs a streamed request to completion and returns the full text. It is
// used by the connection-test endpoint and anywhere a single string is wanted.
func Complete(ctx context.Context, c Client, p Provider, req ChatRequest) (string, error) {
	var sb strings.Builder
	err := c.Stream(ctx, p, req, func(chunk StreamChunk) {
		sb.WriteString(chunk.Delta)
	})
	return sb.String(), err
}

// modelFor returns the request model, falling back to the provider default.
func modelFor(p Provider, req ChatRequest) string {
	if strings.TrimSpace(req.Model) != "" {
		return req.Model
	}
	return p.Model
}

// postSSE issues a streaming POST and invokes onLine for each non-empty `data:`
// payload line, stopping when onLine returns false or the body ends.
func postSSE(ctx context.Context, url string, headers map[string]string, body any, onLine func(payload string) bool) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(buf))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("call provider: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("provider returned %d: %s", resp.StatusCode, strings.TrimSpace(string(detail)))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64<<10), 4<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if !onLine(payload) {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}
	return nil
}

// --- OpenAI-compatible -------------------------------------------------------

type openAIClient struct{}

func (openAIClient) Stream(ctx context.Context, p Provider, req ChatRequest, emit func(StreamChunk)) error {
	base := p.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	headers := map[string]string{}
	if p.APIKey != "" {
		headers["Authorization"] = "Bearer " + p.APIKey
	}
	payload := map[string]any{
		"model":    modelFor(p, req),
		"messages": req.Messages,
		"stream":   true,
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		payload["max_tokens"] = req.MaxTokens
	}

	err := postSSE(ctx, base+"/chat/completions", headers, payload, func(data string) bool {
		if data == "[DONE]" {
			return false
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil {
			return true
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			emit(StreamChunk{Delta: chunk.Choices[0].Delta.Content})
		}
		return true
	})
	if err != nil {
		return err
	}
	emit(StreamChunk{Done: true})
	return nil
}

// --- Anthropic Messages ------------------------------------------------------

type anthropicClient struct{}

func (anthropicClient) Stream(ctx context.Context, p Provider, req ChatRequest, emit func(StreamChunk)) error {
	base := p.BaseURL
	if base == "" {
		base = "https://api.anthropic.com/v1"
	}
	headers := map[string]string{
		"anthropic-version": "2023-06-01",
	}
	if p.APIKey != "" {
		headers["x-api-key"] = p.APIKey
	}

	// Anthropic takes the system prompt as a top-level field, not a message.
	var system string
	messages := make([]ChatMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			if system != "" {
				system += "\n\n"
			}
			system += m.Content
			continue
		}
		messages = append(messages, m)
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1024
	}
	payload := map[string]any{
		"model":      modelFor(p, req),
		"messages":   messages,
		"max_tokens": maxTokens,
		"stream":     true,
	}
	if system != "" {
		payload["system"] = system
	}
	if req.Temperature > 0 {
		payload["temperature"] = req.Temperature
	}

	err := postSSE(ctx, base+"/messages", headers, payload, func(data string) bool {
		var evt struct {
			Type  string `json:"type"`
			Delta struct {
				Text string `json:"text"`
			} `json:"delta"`
		}
		if json.Unmarshal([]byte(data), &evt) != nil {
			return true
		}
		if evt.Type == "content_block_delta" && evt.Delta.Text != "" {
			emit(StreamChunk{Delta: evt.Delta.Text})
		}
		return evt.Type != "message_stop"
	})
	if err != nil {
		return err
	}
	emit(StreamChunk{Done: true})
	return nil
}

// --- Google Gemini -----------------------------------------------------------

type geminiClient struct{}

func (geminiClient) Stream(ctx context.Context, p Provider, req ChatRequest, emit func(StreamChunk)) error {
	base := p.BaseURL
	if base == "" {
		base = "https://generativelanguage.googleapis.com/v1beta"
	}
	model := modelFor(p, req)
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse", base, model)
	if p.APIKey != "" {
		url += "&key=" + p.APIKey
	}

	// Gemini uses "user"/"model" roles and a top-level systemInstruction.
	var systemParts []map[string]string
	contents := make([]map[string]any, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			systemParts = append(systemParts, map[string]string{"text": m.Content})
			continue
		}
		role := "user"
		if m.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, map[string]any{
			"role":  role,
			"parts": []map[string]string{{"text": m.Content}},
		})
	}
	payload := map[string]any{"contents": contents}
	if len(systemParts) > 0 {
		payload["systemInstruction"] = map[string]any{"parts": systemParts}
	}
	genConfig := map[string]any{}
	if req.Temperature > 0 {
		genConfig["temperature"] = req.Temperature
	}
	if req.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = req.MaxTokens
	}
	if len(genConfig) > 0 {
		payload["generationConfig"] = genConfig
	}

	err := postSSE(ctx, url, nil, payload, func(data string) bool {
		var resp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				} `json:"content"`
			} `json:"candidates"`
		}
		if json.Unmarshal([]byte(data), &resp) != nil {
			return true
		}
		for _, cand := range resp.Candidates {
			for _, part := range cand.Content.Parts {
				if part.Text != "" {
					emit(StreamChunk{Delta: part.Text})
				}
			}
		}
		return true
	})
	if err != nil {
		return err
	}
	emit(StreamChunk{Done: true})
	return nil
}
