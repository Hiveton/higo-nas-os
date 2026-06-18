package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"higoos/server-go/internal/assistant"
	"higoos/server-go/internal/llm"
	"higoos/server-go/internal/platform"
)

// aiProviders handles list/create on /api/v1/ai/providers.
func (a *API) aiProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		platform.WriteJSON(w, r, http.StatusOK, a.llm.List())
	case http.MethodPost:
		var body llm.ProviderInput
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		view, err := a.llm.Create(body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "ai_provider_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, view)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

// aiProviderByID handles update/delete on /api/v1/ai/providers/{id} and the
// connection test on /api/v1/ai/providers/{id}/test.
func (a *API) aiProviderByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/ai/providers/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "ai_provider_route_not_found", "ai provider route not found")
		return
	}
	id := parts[0]

	if len(parts) == 2 && parts[1] == "test" {
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		a.aiProviderTest(w, r, id)
		return
	}
	if len(parts) != 1 {
		platform.WriteError(w, r, http.StatusNotFound, "ai_provider_route_not_found", "ai provider route not found")
		return
	}

	switch r.Method {
	case http.MethodPut, http.MethodPatch:
		var body llm.ProviderInput
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		view, err := a.llm.Update(id, body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "ai_provider_update_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, view)
	case http.MethodDelete:
		if err := a.llm.Delete(id); err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "ai_provider_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"id": id, "deleted": true})
	default:
		allowMethod(w, r, http.MethodPut, http.MethodPatch, http.MethodDelete)
	}
}

// aiProviderTest sends a tiny non-streamed prompt to verify the binding works.
func (a *API) aiProviderTest(w http.ResponseWriter, r *http.Request, id string) {
	provider, err := a.llm.Get(id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "ai_provider_not_found", err.Error())
		return
	}
	client, err := llm.NewClient(provider.Kind)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "ai_provider_unsupported", err.Error())
		return
	}
	req := llm.ChatRequest{
		MaxTokens: 16,
		Messages:  []llm.ChatMessage{{Role: "user", Content: "Reply with the single word: OK"}},
	}
	start := time.Now()
	reply, err := llm.Complete(r.Context(), client, provider, req)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadGateway, "ai_provider_test_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"ok":        true,
		"model":     provider.Model,
		"reply":     strings.TrimSpace(reply),
		"latencyMs": time.Since(start).Milliseconds(),
	})
}

// wantsEventStream reports whether the client asked for an SSE response, either
// via the Accept header or a ?stream=true query parameter.
func wantsEventStream(r *http.Request) bool {
	if strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
		return true
	}
	return r.URL.Query().Get("stream") == "true"
}

// streamAssistantMessage records a user turn and streams the model reply as SSE.
// Each delta is sent as a `data: {"delta":"..."}` frame; the stream ends with a
// `data: {"done":true,...}` frame carrying the persisted message + pending action.
func (a *API) streamAssistantMessage(w http.ResponseWriter, r *http.Request, threadID string, req assistant.MessageRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		platform.WriteError(w, r, http.StatusInternalServerError, "stream_unsupported", "streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	writeEvent := func(payload any) {
		buf, err := json.Marshal(payload)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", buf)
		flusher.Flush()
	}

	result, err := a.assistant.AddMessageStream(r.Context(), threadID, req, func(chunk llm.StreamChunk) {
		if chunk.Delta != "" {
			writeEvent(map[string]any{"delta": chunk.Delta})
		}
	})
	if err != nil {
		writeEvent(map[string]any{"error": err.Error(), "done": true})
		return
	}
	writeEvent(map[string]any{
		"done":    true,
		"message": mapAssistantMessage(result.AssistantMessage),
	})
}
