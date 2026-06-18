package llm

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sseServer returns a test server that writes the given raw SSE body and records
// the request path + decoded body for assertion.
func sseServer(t *testing.T, body string, capture *string, capturePath *string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if capturePath != nil {
			*capturePath = r.URL.Path
		}
		if capture != nil {
			raw, _ := io.ReadAll(r.Body)
			*capture = string(raw)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, body)
	}))
}

func collect(t *testing.T, c Client, p Provider) string {
	t.Helper()
	var sb strings.Builder
	var sawDone bool
	err := c.Stream(context.Background(), p, ChatRequest{Messages: []ChatMessage{
		{Role: "system", Content: "you are helpful"},
		{Role: "user", Content: "hi"},
	}}, func(chunk StreamChunk) {
		sb.WriteString(chunk.Delta)
		if chunk.Done {
			sawDone = true
		}
	})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	if !sawDone {
		t.Fatal("expected terminal Done chunk")
	}
	return sb.String()
}

func TestOpenAIClientStream(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"Hel\"}}]}\n\n" +
		"data: {\"choices\":[{\"delta\":{\"content\":\"lo\"}}]}\n\n" +
		"data: [DONE]\n\n"
	var reqBody, path string
	srv := sseServer(t, body, &reqBody, &path)
	defer srv.Close()

	got := collect(t, openAIClient{}, Provider{Kind: KindOpenAI, BaseURL: srv.URL, APIKey: "k", Model: "gpt"})
	if got != "Hello" {
		t.Fatalf("expected Hello, got %q", got)
	}
	if path != "/chat/completions" {
		t.Fatalf("unexpected path %q", path)
	}
	if !strings.Contains(reqBody, "\"stream\":true") {
		t.Fatalf("expected stream flag in body: %s", reqBody)
	}
}

func TestAnthropicClientStream(t *testing.T) {
	body := "data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"Hi \"}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"there\"}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"
	var reqBody, path string
	srv := sseServer(t, body, &reqBody, &path)
	defer srv.Close()

	got := collect(t, anthropicClient{}, Provider{Kind: KindAnthropic, BaseURL: srv.URL, APIKey: "k", Model: "claude"})
	if got != "Hi there" {
		t.Fatalf("expected 'Hi there', got %q", got)
	}
	if path != "/messages" {
		t.Fatalf("unexpected path %q", path)
	}
	// System prompt must be hoisted to a top-level field, not a message.
	if !strings.Contains(reqBody, "\"system\":\"you are helpful\"") {
		t.Fatalf("expected hoisted system prompt: %s", reqBody)
	}
}

func TestGeminiClientStream(t *testing.T) {
	body := "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Ge\"}]}}]}\n\n" +
		"data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"mini\"}]}}]}\n\n"
	var reqBody, path string
	srv := sseServer(t, body, &reqBody, &path)
	defer srv.Close()

	got := collect(t, geminiClient{}, Provider{Kind: KindGemini, BaseURL: srv.URL, APIKey: "k", Model: "gemini-1.5-pro"})
	if got != "Gemini" {
		t.Fatalf("expected Gemini, got %q", got)
	}
	if !strings.Contains(path, ":streamGenerateContent") {
		t.Fatalf("unexpected path %q", path)
	}
	if !strings.Contains(reqBody, "systemInstruction") {
		t.Fatalf("expected systemInstruction in body: %s", reqBody)
	}
}

func TestClientReturnsErrorOnHTTPFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusUnauthorized)
	}))
	defer srv.Close()

	err := openAIClient{}.Stream(context.Background(), Provider{Kind: KindOpenAI, BaseURL: srv.URL, Model: "m"},
		ChatRequest{Messages: []ChatMessage{{Role: "user", Content: "x"}}}, func(StreamChunk) {})
	if err == nil {
		t.Fatal("expected error on 401")
	}
}

func TestCompleteCollectsText(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\n" + "data: [DONE]\n\n"
	srv := sseServer(t, body, nil, nil)
	defer srv.Close()

	got, err := Complete(context.Background(), openAIClient{}, Provider{Kind: KindOpenAI, BaseURL: srv.URL, Model: "m"},
		ChatRequest{Messages: []ChatMessage{{Role: "user", Content: "x"}}})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if got != "OK" {
		t.Fatalf("expected OK, got %q", got)
	}
}

func TestNewClientUnsupported(t *testing.T) {
	if _, err := NewClient("bogus"); err == nil {
		t.Fatal("expected error for unsupported kind")
	}
}
