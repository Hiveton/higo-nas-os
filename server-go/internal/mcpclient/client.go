// Package mcpclient lets the in-app assistant drive the full HiGoOS tool catalog
// through the Model Context Protocol, exactly like an external MCP client would.
//
// It connects an in-process MCP client to the embedded MCP server over an
// in-memory transport, lists the server's tools, and adapts them into the
// agent.Tool shape the assistant's tool-calling loop already understands. Write
// safety is driven by each tool's MCP ReadOnly annotation: read-only tools run
// inline; everything else is gated behind the existing confirmation flow.
package mcpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"higoos/server-go/internal/agent"
	"higoos/server-go/internal/apiclient"
	higomcp "higoos/server-go/internal/mcp"
	"higoos/server-go/internal/platform"
)

// Session holds the live MCP client session bound to the embedded server.
type Session struct {
	mu sync.Mutex
	cs *mcpsdk.ClientSession
}

// Connect builds the embedded MCP server bound to loopback, wires an in-process
// client to it, and returns the live session.
func Connect(ctx context.Context, cfg platform.Config, loopback *apiclient.Client) (*Session, error) {
	server := higomcp.BuildServer(cfg, loopback)
	serverT, clientT := mcpsdk.NewInMemoryTransports()
	go func() { _ = server.Run(context.Background(), serverT) }()

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "higo-assistant", Version: "dev"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		return nil, fmt.Errorf("connect in-process mcp: %w", err)
	}
	return &Session{cs: cs}, nil
}

// Catalog lists the MCP tools and adapts them to agent.Tool. The assistant domain
// is excluded to avoid the assistant recursively calling itself.
func (s *Session) Catalog(ctx context.Context) ([]agent.Tool, error) {
	res, err := s.cs.ListTools(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("list mcp tools: %w", err)
	}
	tools := make([]agent.Tool, 0, len(res.Tools))
	for _, t := range res.Tools {
		if strings.HasPrefix(t.Name, "higo.assistant.") {
			continue
		}
		realName := t.Name
		llmName := sanitize(realName)

		params, _ := t.InputSchema.(map[string]any)
		if params == nil {
			params = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		readOnly := t.Annotations != nil && t.Annotations.ReadOnlyHint
		desc := t.Description

		tools = append(tools, agent.Tool{
			Name:        llmName,
			Description: desc,
			Parameters:  params,
			Write:       !readOnly,
			Impact: func(args map[string]any) string {
				return impactText(desc, args)
			},
			Run: func(ctx context.Context, _ *apiclient.Client, args map[string]any) (json.RawMessage, error) {
				return s.call(ctx, realName, args)
			},
		})
	}
	return tools, nil
}

// call invokes a tool by its real MCP name and returns its text content as JSON.
func (s *Session) call(ctx context.Context, name string, args map[string]any) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.cs.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(contentText(res.Content))
	if res.IsError {
		if text == "" {
			text = "tool error"
		}
		return nil, fmt.Errorf("%s", text)
	}
	if text == "" {
		return json.RawMessage("null"), nil
	}
	return json.RawMessage(text), nil
}

func contentText(content []mcpsdk.Content) string {
	var b strings.Builder
	for _, c := range content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// sanitize maps an MCP tool name (e.g. "higo.storage.pools.list") to an
// OpenAI-function-safe name ("higo_storage_pools_list"). Names are capped at 64
// chars to satisfy provider limits.
func sanitize(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	s := b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

func impactText(desc string, args map[string]any) string {
	if len(args) == 0 {
		return desc
	}
	raw, _ := json.Marshal(args)
	compact := string(raw)
	if len(compact) > 120 {
		compact = compact[:120] + "…"
	}
	return desc + "（参数：" + compact + "）"
}
