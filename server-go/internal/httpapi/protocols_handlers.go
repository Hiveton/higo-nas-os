package httpapi

import (
	"net/http"
	"strings"

	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/protocols"
)

// protocolsList returns every sharing protocol with live running/installed state.
func (a *API) protocolsList(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	list, err := a.protocols.List(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "protocols_list_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, list)
}

// protocolsShares returns every configured share across all protocols.
func (a *API) protocolsShares(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	shares, err := a.protocols.Shares(r.Context(), "")
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "protocols_shares_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, shares)
}

// protocolsAudit returns the governance audit log (newest first).
func (a *API) protocolsAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	entries, err := a.protocols.Audit(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "protocols_audit_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entries)
}

// protocolsAuditByID handles POST /api/v1/protocols/audit/{id}/rollback.
func (a *API) protocolsAuditByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/protocols/audit/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "rollback" {
		platform.WriteError(w, r, http.StatusNotFound, "protocols_audit_route_not_found", "protocols audit route not found")
		return
	}
	var body protocols.RollbackRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	entry, err := a.protocols.Rollback(r.Context(), parts[0], body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "protocols_rollback_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entry)
}

// protocolShareByID handles POST /api/v1/protocols/shares/{id}/delete/{preview|confirm}.
func (a *API) protocolShareByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/protocols/shares/"), "/"), "/")
	if len(parts) != 3 || parts[1] != "delete" {
		platform.WriteError(w, r, http.StatusNotFound, "protocols_share_route_not_found", "protocols share route not found")
		return
	}
	id := parts[0]
	switch parts[2] {
	case "preview":
		var body protocols.ActorRequest
		if r.ContentLength != 0 {
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
		}
		preview, err := a.protocols.PreviewDeleteShare(r.Context(), id, body.Actor)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "protocols_preview_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, preview)
	case "confirm":
		a.protocolsConfirm(w, r)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "protocols_share_route_not_found", "protocols share route not found")
	}
}

// protocolByKey handles per-protocol routes:
//
//	GET  /api/v1/protocols/{key}
//	GET  /api/v1/protocols/{key}/shares
//	POST /api/v1/protocols/{key}/enable/{preview|confirm}
//	POST /api/v1/protocols/{key}/disable/{preview|confirm}
//	POST /api/v1/protocols/{key}/shares/{preview|confirm}
func (a *API) protocolByKey(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/protocols/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "protocols_route_not_found", "protocols route not found")
		return
	}
	key := protocols.ProtocolKey(parts[0])

	switch len(parts) {
	case 1: // GET /{key}
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		proto, err := a.protocols.Get(r.Context(), key)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "protocol_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, proto)
	case 2: // GET /{key}/shares  |  PUT /{key}/config
		switch parts[1] {
		case "shares":
			if !allowMethod(w, r, http.MethodGet) {
				return
			}
			shares, err := a.protocols.Shares(r.Context(), key)
			if err != nil {
				platform.WriteError(w, r, http.StatusInternalServerError, "protocols_shares_failed", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, shares)
		case "config":
			if !allowMethod(w, r, http.MethodPut) {
				return
			}
			var body protocols.ConfigUpdateRequest
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			proto, err := a.protocols.UpdateConfig(r.Context(), key, body)
			if err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "protocols_config_failed", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, proto)
		default:
			platform.WriteError(w, r, http.StatusNotFound, "protocols_route_not_found", "protocols route not found")
		}
	case 3: // POST /{key}/{action}/{verb}
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		a.protocolAction(w, r, key, parts[1], parts[2])
	default:
		platform.WriteError(w, r, http.StatusNotFound, "protocols_route_not_found", "protocols route not found")
	}
}

func (a *API) protocolAction(w http.ResponseWriter, r *http.Request, key protocols.ProtocolKey, action, verb string) {
	switch action {
	case "enable", "disable":
		switch verb {
		case "preview":
			var body protocols.ActorRequest
			if r.ContentLength != 0 {
				if err := decodeJSON(r, &body); err != nil {
					platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
					return
				}
			}
			var (
				preview protocols.ProtocolPreview
				err     error
			)
			if action == "enable" {
				preview, err = a.protocols.PreviewEnable(r.Context(), key, body.Actor)
			} else {
				preview, err = a.protocols.PreviewDisable(r.Context(), key, body.Actor)
			}
			if err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "protocols_preview_failed", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, preview)
		case "confirm":
			a.protocolsConfirm(w, r)
		default:
			platform.WriteError(w, r, http.StatusNotFound, "protocols_route_not_found", "protocols route not found")
		}
	case "shares":
		switch verb {
		case "preview":
			var body protocols.CreateShareRequest
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			preview, err := a.protocols.PreviewCreateShare(r.Context(), key, body)
			if err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "protocols_preview_failed", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, preview)
		case "confirm":
			a.protocolsConfirm(w, r)
		default:
			platform.WriteError(w, r, http.StatusNotFound, "protocols_route_not_found", "protocols route not found")
		}
	default:
		platform.WriteError(w, r, http.StatusNotFound, "protocols_route_not_found", "protocols route not found")
	}
}

// protocolsConfirm applies any previewed change — enable/disable/share-create/
// share-delete are all disambiguated by the confirmation id in the body.
func (a *API) protocolsConfirm(w http.ResponseWriter, r *http.Request) {
	var body protocols.ConfirmRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	result, err := a.protocols.Confirm(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "protocols_confirm_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, result)
}
