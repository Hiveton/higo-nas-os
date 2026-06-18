package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"higoos/server-go/internal/apiclient"
)

// intArg coerces a JSON argument (number or numeric string) to an int.
func intArg(args map[string]any, key string) (int, bool) {
	if args == nil {
		return 0, false
	}
	switch v := args[key].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n, true
		}
	}
	return 0, false
}

func boolArg(args map[string]any, key string) bool {
	if args == nil {
		return false
	}
	b, _ := args[key].(bool)
	return b
}

// stringSliceArg accepts a JSON array of strings (or a single string).
func stringSliceArg(args map[string]any, key string) []string {
	if args == nil {
		return nil
	}
	switch v := args[key].(type) {
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if strings.TrimSpace(v) != "" {
			return []string{v}
		}
	}
	return nil
}

func post(ctx context.Context, c *apiclient.Client, path string, body any) (json.RawMessage, error) {
	return c.Do(ctx, "POST", path, nil, body)
}

// WriteCatalog returns the confirmation-gated mutating tools.
func WriteCatalog() []Tool {
	return []Tool{
		// --- Downloads ---
		{
			Name:        "pause_download_task",
			Description: "暂停一个下载任务。",
			Write:       true,
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"taskId": map[string]any{"type": "integer", "description": "下载任务 ID"}},
				"required":   []string{"taskId"},
			},
			Impact: func(a map[string]any) string {
				id, _ := intArg(a, "taskId")
				return fmt.Sprintf("暂停下载任务 #%d", id)
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				id, ok := intArg(a, "taskId")
				if !ok {
					return nil, fmt.Errorf("taskId is required")
				}
				return post(ctx, c, fmt.Sprintf("/api/v1/downloads/tasks/%d/pause", id), nil)
			},
		},
		{
			Name:        "resume_download_task",
			Description: "继续一个已暂停的下载任务。",
			Write:       true,
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"taskId": map[string]any{"type": "integer", "description": "下载任务 ID"}},
				"required":   []string{"taskId"},
			},
			Impact: func(a map[string]any) string {
				id, _ := intArg(a, "taskId")
				return fmt.Sprintf("继续下载任务 #%d", id)
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				id, ok := intArg(a, "taskId")
				if !ok {
					return nil, fmt.Errorf("taskId is required")
				}
				return post(ctx, c, fmt.Sprintf("/api/v1/downloads/tasks/%d/resume", id), nil)
			},
		},
		{
			Name:        "delete_download_task",
			Description: "删除一个下载任务，可选是否同时删除已下载文件。",
			Write:       true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"taskId":     map[string]any{"type": "integer", "description": "下载任务 ID"},
					"deleteFile": map[string]any{"type": "boolean", "description": "是否同时删除已下载的文件"},
				},
				"required": []string{"taskId"},
			},
			Impact: func(a map[string]any) string {
				id, _ := intArg(a, "taskId")
				if boolArg(a, "deleteFile") {
					return fmt.Sprintf("删除下载任务 #%d 及其文件", id)
				}
				return fmt.Sprintf("删除下载任务 #%d（保留文件）", id)
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				id, ok := intArg(a, "taskId")
				if !ok {
					return nil, fmt.Errorf("taskId is required")
				}
				q := url.Values{}
				if boolArg(a, "deleteFile") {
					q.Set("deleteFile", "true")
				}
				return c.Do(ctx, "DELETE", fmt.Sprintf("/api/v1/downloads/tasks/%d", id), q, nil)
			},
		},
		{
			Name:        "add_download_task",
			Description: "新增一个下载任务（HTTP/磁力/BT 链接）。",
			Write:       true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"link":     map[string]any{"type": "string", "description": "下载链接或磁力链接"},
					"name":     map[string]any{"type": "string", "description": "任务名称（可选）"},
					"category": map[string]any{"type": "string", "description": "分类（可选）"},
				},
				"required": []string{"link"},
			},
			Impact: func(a map[string]any) string {
				return "新增下载任务：" + stringArg(a, "link")
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				link := stringArg(a, "link")
				if link == "" {
					return nil, fmt.Errorf("link is required")
				}
				body := map[string]any{"link": link, "name": stringArg(a, "name"), "category": stringArg(a, "category")}
				return post(ctx, c, "/api/v1/downloads/tasks", body)
			},
		},
		// --- Docker ---
		dockerActionTool("start_container", "启动", "start"),
		dockerActionTool("stop_container", "停止", "stop"),
		dockerActionTool("restart_container", "重启", "restart"),
		// --- Files ---
		{
			Name:        "move_files",
			Description: "把一批文件移动到目标目录。fileIds 来自 search_files / semantic_search 的结果。",
			Write:       true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fileIds":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "文件 ID 列表"},
					"destination": map[string]any{"type": "string", "description": "目标目录路径"},
				},
				"required": []string{"fileIds", "destination"},
			},
			Impact: func(a map[string]any) string {
				return fmt.Sprintf("移动 %d 个文件到 %s", len(stringSliceArg(a, "fileIds")), stringArg(a, "destination"))
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				ids := stringSliceArg(a, "fileIds")
				dest := stringArg(a, "destination")
				if len(ids) == 0 || dest == "" {
					return nil, fmt.Errorf("fileIds and destination are required")
				}
				return post(ctx, c, "/api/v1/files/batch/move", map[string]any{"type": "move", "fileIds": ids, "destination": dest})
			},
		},
		{
			Name:        "rename_file",
			Description: "重命名一个文件。fileId 来自搜索结果。",
			Write:       true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fileId":  map[string]any{"type": "string", "description": "文件 ID"},
					"newName": map[string]any{"type": "string", "description": "新文件名"},
				},
				"required": []string{"fileId", "newName"},
			},
			Impact: func(a map[string]any) string {
				return fmt.Sprintf("重命名文件 %s → %s", stringArg(a, "fileId"), stringArg(a, "newName"))
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				id := stringArg(a, "fileId")
				name := stringArg(a, "newName")
				if id == "" || name == "" {
					return nil, fmt.Errorf("fileId and newName are required")
				}
				return post(ctx, c, "/api/v1/files/batch/rename", map[string]any{
					"type": "rename", "fileIds": []string{id}, "rename": map[string]string{id: name},
				})
			},
		},
		{
			Name:        "delete_files",
			Description: "删除一批文件（移入回收站/删除）。不可逆，需谨慎。fileIds 来自搜索结果。",
			Write:       true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"fileIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "文件 ID 列表"},
				},
				"required": []string{"fileIds"},
			},
			Impact: func(a map[string]any) string {
				return fmt.Sprintf("删除 %d 个文件", len(stringSliceArg(a, "fileIds")))
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				ids := stringSliceArg(a, "fileIds")
				if len(ids) == 0 {
					return nil, fmt.Errorf("fileIds is required")
				}
				return post(ctx, c, "/api/v1/files/batch/delete", map[string]any{"type": "delete", "fileIds": ids})
			},
		},
		// --- Storage ---
		{
			Name:        "create_snapshot",
			Description: "为某个存储池创建快照（可选指定 targetPool）。",
			Write:       true,
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"targetPool": map[string]any{"type": "string", "description": "目标存储池（可选）"}},
			},
			Impact: func(a map[string]any) string {
				if p := stringArg(a, "targetPool"); p != "" {
					return "为存储池 " + p + " 创建快照"
				}
				return "创建存储快照"
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				return post(ctx, c, "/api/v1/storage/tasks/snapshot", map[string]any{"targetPool": stringArg(a, "targetPool")})
			},
		},
		{
			Name:        "run_smart_scan",
			Description: "对磁盘运行 SMART 健康扫描（可选指定 targetSlot）。",
			Write:       true,
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"targetSlot": map[string]any{"type": "string", "description": "目标磁盘槽位（可选）"}},
			},
			Impact: func(a map[string]any) string {
				if s := stringArg(a, "targetSlot"); s != "" {
					return "对磁盘 " + s + " 运行 SMART 扫描"
				}
				return "运行 SMART 健康扫描"
			},
			Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
				return post(ctx, c, "/api/v1/storage/tasks/smart-scan", map[string]any{"targetSlot": stringArg(a, "targetSlot")})
			},
		},
	}
}

func dockerActionTool(name, verb, action string) Tool {
	return Tool{
		Name:        name,
		Description: verb + "一个 Docker 容器。containerId 来自 list_docker_containers。",
		Write:       true,
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"containerId": map[string]any{"type": "string", "description": "容器 ID 或名称"}},
			"required":   []string{"containerId"},
		},
		Impact: func(a map[string]any) string {
			return fmt.Sprintf("%s容器 %s", verb, stringArg(a, "containerId"))
		},
		Run: func(ctx context.Context, c *apiclient.Client, a map[string]any) (json.RawMessage, error) {
			id := stringArg(a, "containerId")
			if id == "" {
				return nil, fmt.Errorf("containerId is required")
			}
			return post(ctx, c, fmt.Sprintf("/api/v1/docker/containers/%s/%s", url.PathEscape(id), action), nil)
		},
	}
}
