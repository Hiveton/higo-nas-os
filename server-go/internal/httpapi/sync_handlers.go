package httpapi

import (
	"net/http"
	"strings"

	"higoos/server-go/internal/foldersync"
	"higoos/server-go/internal/platform"
)

// actorFromRequest derives an audit actor from the authenticated principal.
func actorFromRequest(r *http.Request) string {
	if p, ok := platform.PrincipalFromContext(r.Context()); ok && p.Username != "" {
		return p.Username
	}
	return "sync-ui"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// syncPairs handles GET (list) and POST (create) on /api/v1/sync/pairs.
func (a *API) syncPairs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		pairs, err := a.sync.List(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "sync_list_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, pairs)
	case http.MethodPost:
		var body foldersync.CreatePairRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		pair, err := a.sync.Create(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "sync_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, pair)
	default:
		platform.WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

// syncPairByID handles /api/v1/sync/pairs/{id} and /{id}/{action}.
func (a *API) syncPairByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/sync/pairs/"), "/")
	parts := strings.Split(rest, "/")
	id := parts[0]
	if id == "" {
		platform.WriteError(w, r, http.StatusNotFound, "sync_not_found", "sync pair id required")
		return
	}
	actor := actorFromRequest(r)

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			pair, err := a.sync.Get(r.Context(), id)
			if err != nil {
				platform.WriteError(w, r, http.StatusNotFound, "sync_not_found", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, pair)
		case http.MethodPut:
			var body foldersync.UpdatePairRequest
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
			pair, err := a.sync.Update(r.Context(), id, body)
			if err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "sync_update_failed", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, pair)
		case http.MethodDelete:
			if err := a.sync.Delete(r.Context(), id, actor); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "sync_delete_failed", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, map[string]any{"id": id, "deleted": true})
		default:
			platform.WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
		return
	}

	// /{id}/{action}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var pair foldersync.SyncPair
	var err error
	switch parts[1] {
	case "run":
		pair, err = a.sync.Run(r.Context(), id, actor)
	case "verify":
		pair, err = a.sync.Verify(r.Context(), id, actor)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "sync_route_not_found", "unknown sync action")
		return
	}
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "sync_action_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, pair)
}

// syncConflicts handles GET /api/v1/sync/conflicts.
func (a *API) syncConflicts(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	conflicts, err := a.sync.Conflicts(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "sync_conflicts_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, conflicts)
}

// syncConflictByID handles POST /api/v1/sync/conflicts/{id}/resolve with body {side}.
func (a *API) syncConflictByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/sync/conflicts/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) < 2 || parts[1] != "resolve" {
		platform.WriteError(w, r, http.StatusNotFound, "sync_route_not_found", "expected /conflicts/{id}/resolve")
		return
	}
	var body struct {
		Side  string `json:"side"`
		Actor string `json:"actor,omitempty"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	conflict, err := a.sync.ResolveConflict(r.Context(), parts[0], body.Side, firstNonEmpty(body.Actor, actorFromRequest(r)))
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "sync_resolve_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, conflict)
}

// syncAudit handles GET /api/v1/sync/audit.
func (a *API) syncAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	entries, err := a.sync.Audit(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "sync_audit_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entries)
}
