package httpapi_test

import (
	"bufio"
	"encoding/base64"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestRelayStreaming proves an SSE-style streamed response (response-head →
// response-chunk* → response-end) flows through the relay to the App as it is
// produced, rather than being buffered.
func TestRelayStreaming(t *testing.T) {
	srv := testServer(t)
	access := loginViaSMS(t, srv, "13600004444")

	_, env := do(t, srv, http.MethodPost, "/v1/devices/register", "",
		map[string]string{"deviceId": "hg-sse01", "model": "HiGoOS"}, nil)
	var reg struct{ DeviceSecret string `json:"deviceSecret"` }
	mustData(t, env, &reg)

	_, env = do(t, srv, http.MethodPost, "/v1/bindings/pairing/issue", "",
		map[string]string{"kind": "code"},
		map[string]string{"X-Device-Id": "hg-sse01", "X-Device-Secret": reg.DeviceSecret})
	var pairing struct{ Secret string `json:"secret"` }
	mustData(t, env, &pairing)

	_, env = do(t, srv, http.MethodPost, "/v1/bindings/pairing/claim", access,
		map[string]string{"code": pairing.Secret}, nil)
	var bound struct{ DeviceToken string `json:"deviceToken"` }
	mustData(t, env, &bound)

	// Agent: on a request, emit a head then three SSE chunks then end.
	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1) + "/v1/agent/connect"
	header := http.Header{"X-Device-Id": {"hg-sse01"}, "X-Device-Secret": {reg.DeviceSecret}}
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("agent dial: %v", err)
	}
	defer ws.Close()
	go func() {
		var req struct {
			Type string `json:"type"`
			ID   string `json:"id"`
		}
		if err := ws.ReadJSON(&req); err != nil {
			return
		}
		_ = ws.WriteJSON(map[string]any{"type": "response-head", "id": req.ID, "status": 200,
			"headers": map[string]string{"Content-Type": "text/event-stream"}})
		for i := 0; i < 3; i++ {
			_ = ws.WriteJSON(map[string]any{"type": "response-chunk", "id": req.ID,
				"body": base64.StdEncoding.EncodeToString([]byte("data: tick\n\n"))})
		}
		_ = ws.WriteJSON(map[string]any{"type": "response-end", "id": req.ID})
	}()

	// Wait for the agent to register.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, env := do(t, srv, http.MethodGet, "/v1/bindings", access, nil, nil)
		var list struct {
			Devices []struct{ Device struct{ Online bool `json:"online"` } `json:"device"` } `json:"devices"`
		}
		mustData(t, env, &list)
		if len(list.Devices) > 0 && list.Devices[0].Device.Online {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// Stream the response and count SSE events.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/d/hg-sse01/api/v1/events/stream", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	req.Header.Set("X-Device-Token", bound.DeviceToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}
	scanner := bufio.NewScanner(resp.Body)
	ticks := 0
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "data: tick") {
			ticks++
		}
	}
	if ticks != 3 {
		t.Fatalf("got %d SSE ticks, want 3", ticks)
	}
}
