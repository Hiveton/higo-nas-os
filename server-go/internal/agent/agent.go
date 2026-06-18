// Package agent defines the catalog of NAS tools the assistant's LLM may call,
// and the executor that runs them against the HiGoOS /api/v1 surface via the
// in-process apiclient.
//
// This v1 catalog is intentionally READ-ONLY: it lets the model ground answers in
// real device state (storage, disks, metrics, files, containers, downloads)
// without performing any mutation. Write/destructive actions remain gated behind
// the existing assistant confirmation flow rather than being callable autonomously.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"higoos/server-go/internal/apiclient"
)

// Tool is one callable function exposed to the model.
type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	// Write marks a mutating tool. Write tools are never executed inline by the
	// agent loop; they produce a pending action the user must confirm first.
	Write bool
	// Impact returns a human-readable description of what a confirmed call will
	// do, shown on the confirmation card. Optional; falls back to Description.
	Impact func(args map[string]any) string
	// Run executes the tool against the API and returns the raw JSON result.
	Run func(ctx context.Context, c *apiclient.Client, args map[string]any) (json.RawMessage, error)
}

// Catalog returns the full tool set: read-only tools plus confirmation-gated
// write tools.
func Catalog() []Tool {
	return append(ReadOnlyCatalog(), WriteCatalog()...)
}

// ByName indexes a catalog by tool name.
func ByName(tools []Tool) map[string]Tool {
	out := make(map[string]Tool, len(tools))
	for _, t := range tools {
		out[t.Name] = t
	}
	return out
}

func emptyParams() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func stringArg(args map[string]any, key string) string {
	if args == nil {
		return ""
	}
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func get(ctx context.Context, c *apiclient.Client, path string, query url.Values) (json.RawMessage, error) {
	return c.Do(ctx, "GET", path, query, nil)
}

// ReadOnlyCatalog returns the read-only NAS tools available to the assistant.
func ReadOnlyCatalog() []Tool {
	return []Tool{
		{
			Name:        "get_system_info",
			Description: "获取 NAS 系统信息：名称、版本、运行环境与基础状态。",
			Parameters:  emptyParams(),
			Run: func(ctx context.Context, c *apiclient.Client, _ map[string]any) (json.RawMessage, error) {
				return get(ctx, c, "/api/v1/system/info", nil)
			},
		},
		{
			Name:        "list_storage_pools",
			Description: "列出存储池及其容量、已用与剩余空间、健康状态。回答‘还剩多少空间’这类问题。",
			Parameters:  emptyParams(),
			Run: func(ctx context.Context, c *apiclient.Client, _ map[string]any) (json.RawMessage, error) {
				return get(ctx, c, "/api/v1/storage/pools", nil)
			},
		},
		{
			Name:        "list_disks",
			Description: "列出物理磁盘及其型号、容量、SMART 健康状态。",
			Parameters:  emptyParams(),
			Run: func(ctx context.Context, c *apiclient.Client, _ map[string]any) (json.RawMessage, error) {
				return get(ctx, c, "/api/v1/storage/disks", nil)
			},
		},
		{
			Name:        "get_current_metrics",
			Description: "获取当前实时监控指标：CPU、内存、网络、温度等。",
			Parameters:  emptyParams(),
			Run: func(ctx context.Context, c *apiclient.Client, _ map[string]any) (json.RawMessage, error) {
				return get(ctx, c, "/api/v1/monitoring/metrics/current", nil)
			},
		},
		{
			Name:        "list_docker_containers",
			Description: "列出 Docker 容器及其运行状态、镜像和端口。",
			Parameters:  emptyParams(),
			Run: func(ctx context.Context, c *apiclient.Client, _ map[string]any) (json.RawMessage, error) {
				return get(ctx, c, "/api/v1/docker/containers", nil)
			},
		},
		{
			Name:        "list_download_tasks",
			Description: "列出当前下载任务及其进度、速度与状态。",
			Parameters:  emptyParams(),
			Run: func(ctx context.Context, c *apiclient.Client, _ map[string]any) (json.RawMessage, error) {
				return get(ctx, c, "/api/v1/downloads/tasks", nil)
			},
		},
		{
			Name:        "search_files",
			Description: "按关键字搜索 NAS 文件，返回匹配的文件名、路径与摘要。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "搜索关键字"},
				},
				"required": []string{"query"},
			},
			Run: func(ctx context.Context, c *apiclient.Client, args map[string]any) (json.RawMessage, error) {
				q := stringArg(args, "query")
				if q == "" {
					return nil, fmt.Errorf("query is required")
				}
				return get(ctx, c, "/api/v1/files/search", url.Values{"q": {q}})
			},
		},
		{
			Name:        "semantic_search",
			Description: "对文件内容做语义搜索，回答‘帮我找某份文档’这类问题，返回带引用的结果。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{"type": "string", "description": "自然语言查询"},
				},
				"required": []string{"query"},
			},
			Run: func(ctx context.Context, c *apiclient.Client, args map[string]any) (json.RawMessage, error) {
				q := stringArg(args, "query")
				if q == "" {
					return nil, fmt.Errorf("query is required")
				}
				return c.Do(ctx, "POST", "/api/v1/search/semantic", nil, map[string]any{"query": q})
			},
		},
	}
}
