// Package mcp builds the HiGoOS Model Context Protocol tool catalog.
//
// Every backend REST endpoint is exposed as one MCP tool. Tools are thin: each
// maps its typed input to an internal/apiclient call and returns the API's JSON
// response as text content. The same catalog is served two ways:
//
//   - embedded in higo-api at /mcp (Streamable HTTP), bound to an in-process
//     client, and
//   - by the standalone higo-mcp stdio binary, bound to a remote HTTP client.
//
// All governance (auth, risk confirmation, audit) lives in the HTTP handlers the
// client calls, so the MCP layer adds none of its own.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	mcp "github.com/modelcontextprotocol/go-sdk/mcp"

	"higoos/server-go/internal/apiclient"
	"higoos/server-go/internal/platform"
)

const serverInstructions = "HiGoOS NAS control surface. Tools are grouped by domain " +
	"(higo.<domain>.<action>). Read tools are annotated read-only; destructive tools " +
	"(delete, remove, format, revoke) are annotated accordingly. Medium/high-risk " +
	"actions return a confirmationId and an impact summary instead of executing; call " +
	"the matching *.confirm tool with that id to proceed."

// noInput is the input type for tools that take no parameters.
type noInput struct{}

// BuildServer constructs an MCP server exposing the HiGoOS tool catalog, with
// every tool bound to the given API client. Domains may be filtered via
// cfg.MCPDomains (empty = expose all).
func BuildServer(cfg platform.Config, client *apiclient.Client) *mcp.Server {
	version := strings.TrimSpace(cfg.Version)
	if version == "" {
		version = "dev"
	}
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    "higoos",
		Title:   "HiGoOS NAS",
		Version: version,
	}, &mcp.ServerOptions{Instructions: serverInstructions})

	reg := &registry{server: srv, client: client, domains: parseDomains(cfg.MCPDomains)}

	registerSystem(reg)
	registerSearch(reg)
	registerDesktop(reg)
	registerFiles(reg)
	registerMonitoring(reg)
	registerHardware(reg)
	registerSettings(reg)
	registerStorage(reg)
	registerDownloads(reg)
	registerDocker(reg)
	registerRemote(reg)
	registerVideo(reg)
	registerMusic(reg)
	registerMedia(reg)
	registerAI(reg)
	registerAssistant(reg)
	registerSteward(reg)
	registerSecurity(reg)
	registerProtocols(reg)
	registerNetwork(reg)
	registerAccounts(reg)
	registerAuth(reg)
	registerBackups(reg)
	registerAppCenter(reg)
	registerTasks(reg)

	return srv
}

// registry holds the wiring shared by every domain registration helper.
type registry struct {
	server  *mcp.Server
	client  *apiclient.Client
	domains map[string]bool // nil/empty => expose all domains
}

func (r *registry) enabled(domain string) bool {
	if len(r.domains) == 0 {
		return true
	}
	return r.domains[domain]
}

// addTool registers one MCP tool. In is reflected into the input JSON schema by
// the SDK; the handler maps it to an apiclient call and returns the API JSON as
// text. Out is intentionally `any` so the SDK performs no output-schema
// validation on arbitrary API payloads.
func addTool[In any](r *registry, domain, name, description string, ann *mcp.ToolAnnotations, call func(context.Context, *apiclient.Client, In) (json.RawMessage, error)) {
	if !r.enabled(domain) {
		return
	}
	handler := func(ctx context.Context, _ *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
		data, err := call(ctx, r.client, in)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(data), nil, nil
	}
	mcp.AddTool(r.server, &mcp.Tool{Name: name, Description: description, Annotations: ann}, handler)
}

// jsonResult wraps an API JSON payload as a single text content block,
// pretty-printed when it parses as JSON.
func jsonResult(data json.RawMessage) *mcp.CallToolResult {
	text := string(data)
	var buf bytes.Buffer
	if json.Indent(&buf, data, "", "  ") == nil {
		text = buf.String()
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

// --- annotation helpers -----------------------------------------------------

func boolPtr(b bool) *bool { return &b }

// readOnly marks a tool that does not modify state.
func readOnly() *mcp.ToolAnnotations { return &mcp.ToolAnnotations{ReadOnlyHint: true} }

// mutating marks a non-destructive write (create/update/start/pause/etc.).
func mutating() *mcp.ToolAnnotations { return &mcp.ToolAnnotations{DestructiveHint: boolPtr(false)} }

// destructive marks an irreversible/dangerous write (delete/remove/format/revoke).
func destructive() *mcp.ToolAnnotations { return &mcp.ToolAnnotations{DestructiveHint: boolPtr(true)} }

func parseDomains(csv string) map[string]bool {
	csv = strings.TrimSpace(csv)
	if csv == "" {
		return nil
	}
	out := make(map[string]bool)
	for _, part := range strings.Split(csv, ",") {
		if d := strings.TrimSpace(part); d != "" {
			out[d] = true
		}
	}
	return out
}
