package httpapi

import (
	"net/http"
	"strings"

	"higoos/server-go/internal/iscsi"
	"higoos/server-go/internal/platform"
)

func (a *API) iscsiTargets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		targets, err := a.iscsi.Targets(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "iscsi_list_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, targets)
	case http.MethodPost:
		var body iscsi.CreateTargetRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		target, err := a.iscsi.CreateTarget(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "iscsi_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, target)
	default:
		platform.WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *API) iscsiCapabilities(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	caps, err := a.iscsi.Capabilities(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "iscsi_caps_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, caps)
}

func (a *API) iscsiAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	entries, err := a.iscsi.Audit(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "iscsi_audit_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entries)
}

// iscsiTargetByID handles DELETE /api/v1/iscsi/targets/{iqn} and
// POST /api/v1/iscsi/targets/{iqn}/{luns|acls}.
func (a *API) iscsiTargetByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/iscsi/targets/"), "/")
	parts := strings.SplitN(rest, "/", 2)
	iqn := parts[0]
	if iqn == "" {
		platform.WriteError(w, r, http.StatusNotFound, "iscsi_not_found", "target iqn required")
		return
	}
	actor := actorFromRequest(r)

	if len(parts) == 1 {
		if !allowMethod(w, r, http.MethodDelete) {
			return
		}
		if err := a.iscsi.DeleteTarget(r.Context(), iqn, actor); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "iscsi_delete_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"iqn": iqn, "deleted": true})
		return
	}

	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	switch parts[1] {
	case "luns":
		var body iscsi.AddLUNRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		body.Actor = firstNonEmpty(body.Actor, actor)
		if err := a.iscsi.AddLUN(r.Context(), iqn, body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "iscsi_lun_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"iqn": iqn, "lun": body.Name})
	case "acls":
		var body iscsi.AddACLRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		body.Actor = firstNonEmpty(body.Actor, actor)
		if err := a.iscsi.AddACL(r.Context(), iqn, body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "iscsi_acl_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"iqn": iqn, "initiator": body.Initiator})
	default:
		platform.WriteError(w, r, http.StatusNotFound, "iscsi_route_not_found", "unknown iscsi sub-resource")
	}
}
