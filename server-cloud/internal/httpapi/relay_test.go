package httpapi_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeAgent connects as a NAS, then echoes one forwarded request back as a 200
// carrying the path it received and the rewritten Authorization header. This
// proves the App->cloud->NAS round trip and the device-token rewrite.
func TestRelayForwardRoundTrip(t *testing.T) {
	srv := testServer(t)
	access := loginViaSMS(t, srv, "13700003333")

	// Register + bind a device.
	_, env := do(t, srv, http.MethodPost, "/v1/devices/register", "",
		map[string]string{"deviceId": "hg-relay01", "model": "HiGoOS"}, nil)
	var reg struct {
		DeviceSecret string `json:"deviceSecret"`
	}
	mustData(t, env, &reg)

	_, env = do(t, srv, http.MethodPost, "/v1/bindings/pairing/issue", "",
		map[string]string{"kind": "code"},
		map[string]string{"X-Device-Id": "hg-relay01", "X-Device-Secret": reg.DeviceSecret})
	var pairing struct {
		Secret string `json:"secret"`
	}
	mustData(t, env, &pairing)

	_, env = do(t, srv, http.MethodPost, "/v1/bindings/pairing/claim", access,
		map[string]string{"code": pairing.Secret}, nil)
	var bound struct {
		DeviceToken string `json:"deviceToken"`
	}
	mustData(t, env, &bound)

	// Dial in as the NAS agent.
	wsURL := strings.Replace(srv.URL, "http://", "ws://", 1) + "/v1/agent/connect"
	header := http.Header{"X-Device-Id": {"hg-relay01"}, "X-Device-Secret": {reg.DeviceSecret}}
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		t.Fatalf("agent dial: %v", err)
	}
	defer ws.Close()

	// Agent loop: read one request frame, reply with a response frame.
	agentDone := make(chan map[string]string, 1)
	go func() {
		var req struct {
			Type    string            `json:"type"`
			ID      string            `json:"id"`
			Path    string            `json:"path"`
			Headers map[string]string `json:"headers"`
		}
		if err := ws.ReadJSON(&req); err != nil {
			return
		}
		agentDone <- map[string]string{"path": req.Path, "auth": req.Headers["Authorization"]}
		payload, _ := json.Marshal(map[string]any{"echoedPath": req.Path})
		_ = ws.WriteJSON(map[string]any{
			"type":    "response",
			"id":      req.ID,
			"status":  200,
			"headers": map[string]string{"Content-Type": "application/json"},
			"body":    base64.StdEncoding.EncodeToString(payload),
		})
	}()

	// Wait for the hub to register the agent (online flag flips true).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		_, env := do(t, srv, http.MethodGet, "/v1/bindings", access, nil, nil)
		var list struct {
			Devices []struct {
				Device struct {
					Online bool `json:"online"`
				} `json:"device"`
			} `json:"devices"`
		}
		mustData(t, env, &list)
		if len(list.Devices) > 0 && list.Devices[0].Device.Online {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// App forwards a request through the relay, carrying the device token.
	status, _ := do(t, srv, http.MethodGet, "/d/hg-relay01/api/v1/system/health", access, nil,
		map[string]string{"X-Device-Token": bound.DeviceToken})
	if status != http.StatusOK {
		t.Fatalf("relay forward status=%d", status)
	}

	select {
	case got := <-agentDone:
		if got["path"] != "/api/v1/system/health" {
			t.Fatalf("agent saw wrong path: %q", got["path"])
		}
		if got["auth"] != "Bearer "+bound.DeviceToken {
			t.Fatalf("device token not rewritten into Authorization: %q", got["auth"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("agent never received the forwarded request")
	}
}
