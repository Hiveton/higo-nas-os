package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"higoos/server-go/internal/cloud"
	"higoos/server-go/internal/platform"
)

// resolveDeviceToken verifies a cloud-signed device access token from the
// Authorization header and maps it to the NAS-local principal it names. Returns
// false when there is no Bearer token, the device has not registered (no cloud
// public key), or verification fails — so the caller falls through to the
// existing auth fallbacks.
func (a *API) resolveDeviceToken(r *http.Request) (platform.Principal, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return platform.Principal{}, false
	}
	if a.identity == nil {
		return platform.Principal{}, false
	}
	pub := cloud.LoadPublicKey(a.config.StateDir)
	if pub == "" {
		return platform.Principal{}, false
	}
	tok := strings.TrimSpace(header[len("Bearer "):])
	claims, err := cloud.VerifyDeviceToken(pub, tok, a.identity.DeviceID())
	if err != nil {
		return platform.Principal{}, false
	}
	role := claims.Role
	if role == "" {
		role = "user"
	}
	return platform.Principal{
		UserID:   claims.LocalUserID,
		Username: claims.LocalUserID,
		Role:     role,
		DeviceID: claims.DeviceID,
	}, true
}

// cloudProvision links a HiGoOS cloud account to a NAS-local user during the
// account-binding flow. It is reachable ONLY over the relay tunnel (the cloud
// holds this device's secret); a direct LAN call lacks the relay-origin marker
// and is rejected. server-cloud's relay.Provisioner calls this.
//
// POST /api/v1/cloud/provision { cloudUserId, role } -> { localUserId }
func (a *API) cloudProvision(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if !cloud.IsRelayOrigin(r.Context()) {
		platform.WriteError(w, r, http.StatusForbidden, "forbidden", "cloud provisioning is only reachable over the relay tunnel")
		return
	}
	if a.accounts == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "unavailable", "accounts service unavailable")
		return
	}
	var body struct {
		CloudUserID string `json:"cloudUserId"`
		Role        string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CloudUserID == "" {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_body", "cloudUserId is required")
		return
	}
	user, err := a.accounts.LinkCloudAccount(r.Context(), body.CloudUserID, body.Role)
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "provision_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"localUserId": user.ID})
}

// cloudUnprovision removes the local user backing a cloud account on unbind.
// POST /api/v1/cloud/unprovision { cloudUserId }
func (a *API) cloudUnprovision(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if !cloud.IsRelayOrigin(r.Context()) {
		platform.WriteError(w, r, http.StatusForbidden, "forbidden", "cloud provisioning is only reachable over the relay tunnel")
		return
	}
	if a.accounts == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "unavailable", "accounts service unavailable")
		return
	}
	var body struct {
		CloudUserID string `json:"cloudUserId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.CloudUserID == "" {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_body", "cloudUserId is required")
		return
	}
	if err := a.accounts.UnlinkCloudAccount(r.Context(), body.CloudUserID); err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "unprovision_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true})
}
