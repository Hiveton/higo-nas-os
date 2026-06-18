package assistant

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"higoos/server-go/internal/agent"
	"higoos/server-go/internal/apiclient"
	"higoos/server-go/internal/llm"
)

// TestAgentLoopCallsToolThenAnswers drives the full read-only agent loop: a fake
// OpenAI provider asks for a tool on round 1, the in-process API returns data,
// and the provider answers in text on round 2.
func TestAgentLoopCallsToolThenAnswers(t *testing.T) {
	var toolHit bool

	// In-process API the agent's tools call into.
	api := http.NewServeMux()
	api.HandleFunc("/api/v1/storage/pools", func(w http.ResponseWriter, r *http.Request) {
		toolHit = true
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true,"data":[{"name":"tank","freeBytes":500}]}`)
	})
	loopback := apiclient.NewInProcess(api, apiclient.Auth{})

	// Fake OpenAI-compatible provider: request 1 (no tool result yet) -> tool call;
	// request 2 (carries a tool result) -> final text.
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		if strings.Contains(string(body), `"role":"tool"`) {
			io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"存储池 tank 还剩 500 字节。\"}}]}\n\n")
			io.WriteString(w, "data: [DONE]\n\n")
			return
		}
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_1\",\"function\":{\"name\":\"list_storage_pools\",\"arguments\":\"\"}}]}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{}\"}}]}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer provider.Close()

	store := llm.NewStore()
	kind := llm.KindOpenAI
	if _, err := store.Create(llm.ProviderInput{
		Name: ptr("fake"), Kind: &kind, Model: ptr("m"), BaseURL: ptr(provider.URL), APIKey: ptr("k"),
	}); err != nil {
		t.Fatalf("create provider: %v", err)
	}

	svc := NewService().
		WithLLM(store, llm.NewClient).
		WithTools(loopback, agent.ReadOnlyCatalog())

	var streamed strings.Builder
	var toolPhases []string
	res, err := svc.AddMessageStream(context.Background(), "thread-current",
		MessageRequest{Text: "还剩多少空间？"}, func(e StreamEvent) {
			streamed.WriteString(e.Delta)
			if e.Tool != nil {
				toolPhases = append(toolPhases, e.Tool.Phase+":"+e.Tool.Name)
			}
		})
	if err != nil {
		t.Fatalf("AddMessageStream: %v", err)
	}
	if !toolHit {
		t.Fatal("expected the storage tool to be called")
	}
	want := "存储池 tank 还剩 500 字节。"
	if streamed.String() != want {
		t.Fatalf("streamed = %q, want %q", streamed.String(), want)
	}
	if res.AssistantMessage.Text != want {
		t.Fatalf("persisted = %q, want %q", res.AssistantMessage.Text, want)
	}
	// Tool activity must be surfaced as start+done events and persisted as a trace.
	if len(toolPhases) != 2 || toolPhases[0] != "start:list_storage_pools" || toolPhases[1] != "done:list_storage_pools" {
		t.Fatalf("unexpected tool phases: %v", toolPhases)
	}
	if len(res.AssistantMessage.Tools) != 1 || res.AssistantMessage.Tools[0].Name != "list_storage_pools" {
		t.Fatalf("expected persisted tool trace, got %+v", res.AssistantMessage.Tools)
	}
}

// TestWriteToolGatedByConfirmation verifies a write tool is NOT executed during
// the agent loop (it becomes a pending action) and only runs on ConfirmAction.
func TestWriteToolGatedByConfirmation(t *testing.T) {
	var executed bool

	api := http.NewServeMux()
	api.HandleFunc("/api/v1/downloads/tasks/3/pause", func(w http.ResponseWriter, r *http.Request) {
		executed = true
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true,"data":{"id":3,"state":"paused"}}`)
	})
	loopback := apiclient.NewInProcess(api, apiclient.Auth{})

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		if strings.Contains(string(body), "awaiting_user_confirmation") {
			io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"我已准备好暂停任务，请确认。\"}}]}\n\n")
			io.WriteString(w, "data: [DONE]\n\n")
			return
		}
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"c1\",\"function\":{\"name\":\"pause_download_task\",\"arguments\":\"\"}}]}}]}\n\n")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"taskId\\\":3}\"}}]}}]}\n\n")
		io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer provider.Close()

	store := llm.NewStore()
	kind := llm.KindOpenAI
	store.Create(llm.ProviderInput{Name: ptr("fake"), Kind: &kind, Model: ptr("m"), BaseURL: ptr(provider.URL), APIKey: ptr("k")})

	svc := NewService().WithLLM(store, llm.NewClient).WithTools(loopback, agent.Catalog())

	res, err := svc.AddMessageStream(context.Background(), "thread-current",
		MessageRequest{Text: "暂停下载任务 3"}, nil)
	if err != nil {
		t.Fatalf("AddMessageStream: %v", err)
	}
	if executed {
		t.Fatal("write tool must NOT execute before confirmation")
	}
	actionID := res.AssistantMessage.ActionID
	if actionID == "" || res.Action == nil || res.Action.ToolName != "pause_download_task" {
		t.Fatalf("expected a pending write action, got %+v", res.Action)
	}

	confirmed, err := svc.ConfirmAction(context.Background(), actionID, ConfirmActionRequest{})
	if err != nil {
		t.Fatalf("ConfirmAction: %v", err)
	}
	if confirmed.Status != ActionConfirmed {
		t.Fatalf("expected confirmed, got %s", confirmed.Status)
	}
	if !executed {
		t.Fatal("write tool must execute on confirmation")
	}
	// A result message should have been appended to the thread.
	thread, _ := svc.GetThread(context.Background(), "thread-current")
	last := thread.Messages[len(thread.Messages)-1]
	if !strings.Contains(last.Text, "已执行") {
		t.Fatalf("expected execution result message, got %q", last.Text)
	}
}

func TestThreadLifecycle(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	created, err := svc.CreateThread(ctx, "测试会话")
	if err != nil {
		t.Fatalf("CreateThread: %v", err)
	}
	if created.ID == "" || created.Title != "测试会话" {
		t.Fatalf("unexpected created thread: %+v", created)
	}

	summaries := svc.ListThreads(ctx)
	// Seed thread-current + the new one.
	if len(summaries) < 2 {
		t.Fatalf("expected at least 2 threads, got %d", len(summaries))
	}

	if err := svc.DeleteThread(ctx, created.ID); err != nil {
		t.Fatalf("DeleteThread: %v", err)
	}
	if _, err := svc.GetThread(ctx, created.ID); err == nil {
		t.Fatal("expected deleted thread to be gone")
	}
}
