package httpapi

import (
	"errors"
	"net/http"

	"higoos/server-cloud/internal/bindings"
	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/store"
)

// POST /v1/bindings/pairing/issue { kind } — device-authenticated. The NAS calls
// this to publish a pairing code (kind="code", flow A) or PIN (kind="pin", flow
// C) on its screen/web desktop.
func (a *API) pairingIssue(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	deviceID, _, ok := a.authenticateDevice(w, r)
	if !ok {
		return
	}
	var body struct {
		Kind string `json:"kind"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	kind := store.PairingCode
	if body.Kind == string(store.PairingPIN) {
		kind = store.PairingPIN
	}
	secret, expires, err := a.binding.IssuePairing(r.Context(), deviceID, kind)
	if err != nil {
		writeBindingErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"secret":    secret, // the code/PIN the NAS displays
		"kind":      kind,
		"expiresAt": expires.UTC().Format(http.TimeFormat),
	})
}

// POST /v1/bindings/pairing/claim { code } — flow A.
func (a *API) pairingClaim(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	res, err := a.binding.ClaimPairing(r.Context(), userID, body.Code)
	if err != nil {
		writeBindingErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusCreated, res)
}

// POST /v1/bindings/lan-confirm { deviceId } — flow B.
func (a *API) lanConfirm(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		DeviceID string `json:"deviceId"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	res, err := a.binding.LANConfirm(r.Context(), userID, body.DeviceID)
	if err != nil {
		writeBindingErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusCreated, res)
}

// POST /v1/bindings/serial { serial, pin } — flow C.
func (a *API) bindSerial(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Serial string `json:"serial"`
		PIN    string `json:"pin"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	res, err := a.binding.BindSerial(r.Context(), userID, body.Serial, body.PIN)
	if err != nil {
		writeBindingErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusCreated, res)
}

// GET /v1/bindings — list the account's bound devices.
func (a *API) bindingsList(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	list, err := a.binding.List(r.Context(), userID)
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "internal_error", "could not list bindings")
		return
	}
	// Mark online state against the live relay registry.
	for i := range list {
		list[i].Device.Online = a.relay.Online(list[i].Device.ID)
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"devices": list})
}

// DELETE /v1/bindings/{deviceId} — unbind.
func (a *API) bindingUnbind(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	deviceID := r.PathValue("deviceId")
	if err := a.binding.Unbind(r.Context(), userID, deviceID); err != nil {
		writeBindingErr(w, r, err)
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true})
}

// POST /v1/devices/{deviceId}/access-ticket — mint a fresh device access token.
func (a *API) accessTicket(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	deviceID := r.PathValue("deviceId")
	res, err := a.binding.IssueAccessTicket(r.Context(), userID, deviceID)
	if err != nil {
		writeBindingErr(w, r, err)
		return
	}
	res.Device.Online = a.relay.Online(deviceID)
	platform.WriteJSON(w, r, http.StatusOK, res)
}

func writeBindingErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, bindings.ErrChallengeInvalid):
		platform.WriteError(w, r, http.StatusBadRequest, "pairing_invalid", "pairing challenge invalid or expired")
	case errors.Is(err, bindings.ErrDeviceUnknown):
		platform.WriteError(w, r, http.StatusNotFound, "device_not_found", "device not found")
	case errors.Is(err, bindings.ErrNotBound):
		platform.WriteError(w, r, http.StatusForbidden, "not_bound", "device is not bound to this account")
	case errors.Is(err, bindings.ErrAlreadyBound):
		platform.WriteError(w, r, http.StatusConflict, "already_bound", "device already bound to this account")
	default:
		platform.WriteError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
