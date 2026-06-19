package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"

	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/relay"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// The agent is server-go, not a browser, so any origin is acceptable here;
	// device-secret authentication is what actually gates the connection.
	CheckOrigin: func(*http.Request) bool { return true },
}

// GET /v1/agent/connect — the NAS agent dials in and holds this WebSocket open.
// Authenticated by X-Device-Id + X-Device-Secret (or query params). Blocks for
// the life of the connection.
func (a *API) agentConnect(w http.ResponseWriter, r *http.Request) {
	deviceID, _, ok := a.authenticateDevice(w, r)
	if !ok {
		return
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote the error
	}
	_ = a.devices.Heartbeat(r.Context(), deviceID)
	a.logger.Info("relay agent connected", "deviceId", deviceID)
	a.relay.Serve(deviceID, ws) // blocks until the socket drops
	a.logger.Info("relay agent disconnected", "deviceId", deviceID)
}

// ANY /d/{deviceId}/{rest...} — forward an App request to a bound NAS over the
// relay. The cloud access token (Authorization) authenticates the account; the
// device access token travels in X-Device-Token and is rewritten to the upstream
// Authorization header so server-go authorizes it as the mapped local user.
func (a *API) relayForward(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	deviceID := r.PathValue("deviceId")

	// Authorization: the account must be bound to this device.
	if _, err := a.binding.IssueAccessTicket(r.Context(), userID, deviceID); err != nil {
		platform.WriteError(w, r, http.StatusForbidden, "not_bound", "device is not bound to this account")
		return
	}

	// Swap the cloud token for the device token before crossing the tunnel.
	devTok := r.Header.Get("X-Device-Token")
	r.Header.Del("X-Device-Token")
	if devTok != "" {
		r.Header.Set("Authorization", "Bearer "+devTok)
	} else {
		r.Header.Del("Authorization")
	}

	upstream := "/" + r.PathValue("rest")
	if r.URL.RawQuery != "" {
		upstream += "?" + r.URL.RawQuery
	}

	// Forward streams the NAS response directly to w (supports SSE / chunked).
	if err := a.relay.Forward(r.Context(), deviceID, r, upstream, w); err != nil {
		switch {
		case errors.Is(err, relay.ErrOffline):
			platform.WriteError(w, r, http.StatusServiceUnavailable, "device_offline", "device is offline")
		case errors.Is(err, relay.ErrTimeout):
			platform.WriteError(w, r, http.StatusGatewayTimeout, "device_timeout", "device did not respond")
		case errors.Is(err, context.Canceled):
			// Client disconnected mid-stream — nothing to write.
		default:
			platform.WriteError(w, r, http.StatusBadGateway, "relay_error", "relay error")
		}
		return
	}
}
