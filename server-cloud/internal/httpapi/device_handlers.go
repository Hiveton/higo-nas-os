package httpapi

import (
	"net/http"

	"higoos/server-cloud/internal/platform"
)

// POST /v1/devices/register { deviceId, model, version, agentPublicKey }
// Open endpoint: a NAS self-registers on first boot. Returns a one-time secret +
// serial + the cloud public key the NAS trusts for device access tokens.
func (a *API) deviceRegister(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		DeviceID       string `json:"deviceId"`
		Model          string `json:"model"`
		Version        string `json:"version"`
		AgentPublicKey string `json:"agentPublicKey"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	reg, err := a.devices.Register(r.Context(), body.DeviceID, body.Model, body.Version, body.AgentPublicKey)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "register_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"device":         reg.Device,
		"deviceSecret":   reg.DeviceSecret,
		"serial":         reg.Serial,
		"cloudPublicKey": a.tokens.PublicKeyHex(),
		"relayPath":      "/v1/agent/connect",
	})
}

// POST /v1/devices/heartbeat — device-authenticated keepalive (X-Device-Id +
// X-Device-Secret). Cheap liveness signal independent of the relay socket.
func (a *API) deviceHeartbeat(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	deviceID, _, ok := a.authenticateDevice(w, r)
	if !ok {
		return
	}
	if err := a.devices.Heartbeat(r.Context(), deviceID); err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "internal_error", "heartbeat failed")
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true})
}

// authenticateDevice resolves a device from X-Device-Id + X-Device-Secret headers
// (or query params, for the WebSocket upgrade). Writes 401 and returns false on
// failure.
func (a *API) authenticateDevice(w http.ResponseWriter, r *http.Request) (deviceID, secret string, ok bool) {
	deviceID = r.Header.Get("X-Device-Id")
	secret = r.Header.Get("X-Device-Secret")
	if deviceID == "" {
		deviceID = r.URL.Query().Get("deviceId")
	}
	if secret == "" {
		secret = r.URL.Query().Get("secret")
	}
	if _, err := a.devices.Authenticate(r.Context(), deviceID, secret); err != nil {
		platform.WriteError(w, r, http.StatusUnauthorized, "device_unauthorized", "device authentication failed")
		return "", "", false
	}
	return deviceID, secret, true
}
