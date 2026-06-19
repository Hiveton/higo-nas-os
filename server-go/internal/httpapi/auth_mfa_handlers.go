package httpapi

import (
	"net/http"

	"higoos/server-go/internal/audit"
	"higoos/server-go/internal/platform"
)

// authMFASetup begins TOTP enrollment for the current user, returning the
// secret and an otpauth:// URI for authenticator apps.
func (a *API) authMFASetup(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok {
		platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "no active session")
		return
	}
	secret, uri, err := a.accounts.MFASetup(r.Context(), principal.UserID, principal.Username)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "mfa_setup_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]string{"secret": secret, "otpauthUri": uri})
}

// authMFAEnable confirms enrollment with a TOTP code.
func (a *API) authMFAEnable(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok {
		platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "no active session")
		return
	}
	var body struct {
		Code string `json:"code"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := a.accounts.MFAEnable(r.Context(), principal.UserID, body.Code); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "mfa_enable_failed", err.Error())
		return
	}
	a.auditAuth(r, principal.Username, principal.SessionID, "mfa_enable", audit.ResultAllowed, audit.RiskMedium)
	platform.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "mfa_enabled"})
}

// authMFADisable turns off TOTP after verifying the caller's password.
func (a *API) authMFADisable(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok {
		platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "no active session")
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if _, err := a.accounts.VerifyPassword(r.Context(), principal.Username, body.Password); err != nil {
		platform.WriteError(w, r, http.StatusForbidden, "invalid_credentials", "password is incorrect")
		return
	}
	if err := a.accounts.MFADisable(r.Context(), principal.UserID); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "mfa_disable_failed", err.Error())
		return
	}
	a.auditAuth(r, principal.Username, principal.SessionID, "mfa_disable", audit.ResultAllowed, audit.RiskMedium)
	platform.WriteJSON(w, r, http.StatusOK, map[string]string{"status": "mfa_disabled"})
}
