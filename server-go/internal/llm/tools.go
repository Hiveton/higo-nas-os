package llm

import (
	"context"
	"encoding/json"
	"sort"
)

// StreamWithTools runs one OpenAI-compatible chat-completion round with function
// calling enabled. Content deltas are forwarded to emit as they arrive; any tool
// calls the model requests are accumulated and returned (empty slice when the
// model answered directly). Pass tools=nil to force a plain textual answer.
//
// Only the OpenAI-compatible wire format is supported here — it is the format the
// agent loop targets. Anthropic/Gemini continue to use the plain Stream path.
func StreamWithTools(ctx context.Context, p Provider, messages []ChatMessage, tools []ToolDef, emit func(StreamChunk)) ([]ToolCall, error) {
	base := p.BaseURL
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	headers := map[string]string{}
	if p.APIKey != "" {
		headers["Authorization"] = "Bearer " + p.APIKey
	}

	payload := map[string]any{
		"model":    p.Model,
		"messages": openaiWireMessages(messages),
		"stream":   true,
	}
	if len(tools) > 0 {
		payload["tools"] = openaiWireTools(tools)
		payload["tool_choice"] = "auto"
	}

	acc := map[int]*ToolCall{}
	var order []int
	err := postSSE(ctx, base+"/chat/completions", headers, payload, func(data string) bool {
		if data == "[DONE]" {
			return false
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(data), &chunk) != nil || len(chunk.Choices) == 0 {
			return true
		}
		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			emit(StreamChunk{Delta: delta.Content})
		}
		for _, tc := range delta.ToolCalls {
			cur := acc[tc.Index]
			if cur == nil {
				cur = &ToolCall{}
				acc[tc.Index] = cur
				order = append(order, tc.Index)
			}
			if tc.ID != "" {
				cur.ID = tc.ID
			}
			if tc.Function.Name != "" {
				cur.Name = tc.Function.Name
			}
			cur.Arguments += tc.Function.Arguments
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	sort.Ints(order)
	calls := make([]ToolCall, 0, len(order))
	for _, i := range order {
		calls = append(calls, *acc[i])
	}
	return calls, nil
}

// openaiWireMessages converts internal chat messages into the OpenAI wire shape,
// expanding tool calls and tool results into their nested representations.
func openaiWireMessages(messages []ChatMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		wm := map[string]any{"role": m.Role, "content": m.Content}
		if m.ToolCallID != "" {
			wm["tool_call_id"] = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			calls := make([]map[string]any, 0, len(m.ToolCalls))
			for _, tc := range m.ToolCalls {
				calls = append(calls, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Name,
						"arguments": tc.Arguments,
					},
				})
			}
			wm["tool_calls"] = calls
		}
		out = append(out, wm)
	}
	return out
}

func openaiWireTools(tools []ToolDef) []map[string]any {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		params := t.Parameters
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		out = append(out, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  params,
			},
		})
	}
	return out
}
