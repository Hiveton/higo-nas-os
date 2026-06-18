package mcp

import (
	"context"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"higoos/server-go/internal/apiclient"
	"higoos/server-go/internal/platform"
)

// stubAPI returns an in-process API client that echoes a fixed envelope, so the
// catalog can be exercised without the real backend.
func stubClient() *apiclient.Client {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"ok":true}}`))
	})
	return apiclient.NewInProcess(handler, apiclient.Auth{})
}

func listTools(t *testing.T, srv *sdk.Server) []*sdk.Tool {
	t.Helper()
	ctx := context.Background()
	clientT, serverT := sdk.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()

	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	var tools []*sdk.Tool
	for tool, err := range cs.Tools(ctx, nil) {
		if err != nil {
			t.Fatalf("iterate tools: %v", err)
		}
		tools = append(tools, tool)
	}
	return tools
}

func TestBuildServerRegistersFullCatalog(t *testing.T) {
	srv := BuildServer(platform.Config{Version: "test", MCPEnabled: true}, stubClient())
	tools := listTools(t, srv)

	if len(tools) < 150 {
		t.Fatalf("tool count = %d, expected the full catalog (>=150)", len(tools))
	}

	seen := map[string]bool{}
	for _, tool := range tools {
		if !strings.HasPrefix(tool.Name, "higo.") {
			t.Errorf("tool %q does not use the higo. prefix", tool.Name)
		}
		if seen[tool.Name] {
			t.Errorf("duplicate tool name %q", tool.Name)
		}
		seen[tool.Name] = true
		if strings.TrimSpace(tool.Description) == "" {
			t.Errorf("tool %q has no description", tool.Name)
		}
	}

	// Spot-check that representative read and destructive tools exist and are
	// annotated correctly.
	if tool := seen["higo.system.info"]; !tool {
		t.Error("missing higo.system.info")
	}
	for _, tool := range tools {
		switch tool.Name {
		case "higo.system.info":
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Error("higo.system.info should be read-only")
			}
		case "higo.storage.disks.remove":
			if tool.Annotations == nil || tool.Annotations.DestructiveHint == nil || !*tool.Annotations.DestructiveHint {
				t.Error("higo.storage.disks.remove should be destructive")
			}
		}
	}
}

func TestBuildServerDomainFilter(t *testing.T) {
	srv := BuildServer(platform.Config{Version: "test", MCPEnabled: true, MCPDomains: "system,docker"}, stubClient())
	tools := listTools(t, srv)

	if len(tools) == 0 {
		t.Fatal("expected some tools with domain filter")
	}
	for _, tool := range tools {
		parts := strings.SplitN(tool.Name, ".", 3)
		domain := parts[1]
		if domain != "system" && domain != "docker" {
			t.Errorf("tool %q outside allowed domains", tool.Name)
		}
	}
}

func TestBuildServerCallTool(t *testing.T) {
	ctx := context.Background()
	srv := BuildServer(platform.Config{Version: "test", MCPEnabled: true}, stubClient())

	clientT, serverT := sdk.NewInMemoryTransports()
	ss, err := srv.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer ss.Close()
	client := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer cs.Close()

	res, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: "higo.system.info"})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if res.IsError {
		t.Fatalf("tool returned error: %+v", res.Content)
	}
	if len(res.Content) == 0 {
		t.Fatal("expected content")
	}
}
