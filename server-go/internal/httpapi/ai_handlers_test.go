package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"higoos/server-go/internal/devstub"
	"higoos/server-go/internal/httpapi"
	"higoos/server-go/internal/platform"
)

func testRouter() http.Handler {
	return httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})
}

func TestAIProviderCRUDMasksKey(t *testing.T) {
	router := testRouter()

	createRec := httptest.NewRecorder()
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/ai/providers",
		strings.NewReader(`{"name":"OpenAI","kind":"openai","model":"gpt-4o-mini","apiKey":"sk-abcd1234"}`))
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusOK {
		t.Fatalf("create provider HTTP %d: %s", createRec.Code, createRec.Body.String())
	}
	body := createRec.Body.String()
	if strings.Contains(body, "sk-abcd1234") {
		t.Fatalf("raw api key leaked in response: %s", body)
	}
	if !strings.Contains(body, "1234") || !strings.Contains(body, "keyHint") {
		t.Fatalf("expected masked keyHint, got %s", body)
	}

	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/v1/ai/providers", nil))
	if listRec.Code != http.StatusOK || !strings.Contains(listRec.Body.String(), "OpenAI") {
		t.Fatalf("list providers HTTP %d: %s", listRec.Code, listRec.Body.String())
	}
}

func TestAssistantMessageStreamsSSE(t *testing.T) {
	router := testRouter()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/assistant/threads/thread-current/messages",
		strings.NewReader(`{"role":"user","text":"整理下载目录"}`))
	req.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("expected SSE content type, got %q", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "data:") {
		t.Fatalf("expected SSE data frames, got %s", body)
	}
	// With no provider bound the canned fallback still streams a delta + done frame.
	if !strings.Contains(body, "执行草案") || !strings.Contains(body, "\"done\":true") {
		t.Fatalf("expected canned delta and done frame, got %s", body)
	}
}
