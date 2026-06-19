package httpapi

import (
	"net/http"
	"strings"

	"higoos/server-go/internal/network"
	"higoos/server-go/internal/platform"
)

// networkInterfaces returns the live interface inventory (GET /api/v1/network/interfaces).
func (a *API) networkInterfaces(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	ifaces, err := a.network.Interfaces(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "network_interfaces_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, ifaces)
}

// networkConfig handles GET (current config) and PUT (preview a change) on
// /api/v1/network/config. Changing the address is high risk, so PUT only
// previews — the caller must POST the confirmation to apply.
func (a *API) networkConfig(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet, http.MethodPut) {
		return
	}
	if r.Method == http.MethodGet {
		cfg, err := a.network.Config(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "network_config_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, cfg)
		return
	}
	var body network.ConfigRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	body.Actor = actorFor(r, body.Actor)
	preview, err := a.network.PreviewConfig(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "network_preview_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, preview)
}

// networkConfigConfirm applies a previewed change (POST /api/v1/network/config/confirm).
func (a *API) networkConfigConfirm(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body network.ConfirmRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if strings.TrimSpace(body.ConfirmationID) == "" {
		platform.WriteError(w, r, http.StatusBadRequest, "confirmation_required", "confirmationId is required")
		return
	}
	body.Actor = actorFor(r, body.Actor)
	entry, err := a.network.Confirm(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "network_confirm_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entry)
}

// networkAudit returns the governance audit log (GET /api/v1/network/audit).
func (a *API) networkAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	entries, err := a.network.Audit(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "network_audit_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entries)
}

// networkAuditByID handles POST /api/v1/network/audit/{id}/rollback.
func (a *API) networkAuditByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/network/audit/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "rollback" {
		platform.WriteError(w, r, http.StatusNotFound, "network_audit_route_not_found", "network audit route not found")
		return
	}
	var body network.RollbackRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	body.Actor = actorFor(r, body.Actor)
	entry, err := a.network.Rollback(r.Context(), parts[0], body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "network_rollback_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entry)
}

// systemIdentity returns the unauthenticated device fingerprint
// (GET /api/v1/system/identity) — the HTTP twin of the UDP discovery announce.
func (a *API) systemIdentity(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.identity.Snapshot())
}

// actorFor resolves the audit actor from the request principal, falling back to
// a provided value.
func actorFor(r *http.Request, fallback string) string {
	if principal, ok := platform.PrincipalFromContext(r.Context()); ok && principal.Username != "" {
		return principal.Username
	}
	return fallback
}
