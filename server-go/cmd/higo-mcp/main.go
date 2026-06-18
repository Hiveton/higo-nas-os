// Command higo-mcp is the standalone HiGoOS MCP server. It speaks MCP over
// stdio (for desktop clients such as Claude Desktop / Claude Code) and proxies
// every tool call to a running higo-api over HTTP.
//
// Configuration (environment):
//
//	HIGO_MCP_API_BASE   base URL of the HiGoOS API   (default http://127.0.0.1:8080)
//	HIGO_MCP_API_TOKEN  bearer token sent as Authorization on every call (optional)
//	HIGO_MCP_DOMAINS    optional comma-separated domain allowlist
//	HIGO_VERSION        version string advertised to clients (default dev)
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"higoos/server-go/internal/apiclient"
	higomcp "higoos/server-go/internal/mcp"
	"higoos/server-go/internal/platform"
)

func main() {
	base := getenv("HIGO_MCP_API_BASE", "http://127.0.0.1:8080")
	token := strings.TrimSpace(os.Getenv("HIGO_MCP_API_TOKEN"))

	auth := apiclient.Auth{}
	if token != "" {
		if strings.Contains(token, " ") {
			auth.Authorization = token // already a scheme-qualified value
		} else {
			auth.Authorization = "Bearer " + token
		}
	}

	cfg := platform.Config{
		Version:    getenv("HIGO_VERSION", "dev"),
		MCPEnabled: true,
		MCPDomains: os.Getenv("HIGO_MCP_DOMAINS"),
	}

	client := apiclient.NewRemote(base, auth, nil)
	server := higomcp.BuildServer(cfg, client)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil && ctx.Err() == nil {
		log.Fatalf("higo-mcp: %v", err)
	}
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
