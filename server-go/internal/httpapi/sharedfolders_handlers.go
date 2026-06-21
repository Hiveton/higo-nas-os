package httpapi

import (
	"net/http"
	"os"
	"strings"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/sharedfolders"
)

// serverHostFor derives a best-effort host for SMB/NFS mount hints. The frontend
// typically overrides this with the browser's own hostname.
func serverHostFor(cfg platform.Config) string {
	if h, err := os.Hostname(); err == nil && strings.TrimSpace(h) != "" {
		return h
	}
	return "nas.local"
}

func (a *API) sharedFoldersList(w http.ResponseWriter, r *http.Request) {
	if a.sharedFolders == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "shared_folders_unavailable", "shared folders service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		views, err := a.sharedFolders.List(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "shared_folders_list_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, views)
	case http.MethodPost:
		var body sharedfolders.CreateRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		view, err := a.sharedFolders.Create(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, view)
	default:
		platform.WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *API) sharedFolderByID(w http.ResponseWriter, r *http.Request) {
	if a.sharedFolders == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "shared_folders_unavailable", "shared folders service is unavailable")
		return
	}
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/shared-folders/"), "/")
	parts := strings.Split(rest, "/")
	if len(parts) == 0 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "shared_folder_route_not_found", "shared folder id is required")
		return
	}
	id := parts[0]
	switch {
	case len(parts) == 2 && parts[1] == "permissions":
		if !allowMethod(w, r, http.MethodPut) {
			return
		}
		var body sharedfolders.SetPermissionsRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		view, err := a.sharedFolders.SetPermissions(r.Context(), id, body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_permissions_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, view)
	case len(parts) == 2 && parts[1] == "services":
		if !allowMethod(w, r, http.MethodPut) {
			return
		}
		var body sharedfolders.SetServiceRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		view, err := a.sharedFolders.SetService(r.Context(), id, body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_service_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, view)
	case len(parts) == 2 && parts[1] == "advanced":
		if !allowMethod(w, r, http.MethodPut) {
			return
		}
		var body sharedfolders.SetAdvancedRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		view, err := a.sharedFolders.SetAdvanced(r.Context(), id, body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_advanced_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, view)
	case len(parts) == 2 && parts[1] == "snapshots":
		switch r.Method {
		case http.MethodGet:
			snaps, err := a.sharedFolders.ListSnapshots(r.Context(), id)
			if err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_snapshots_failed", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, snaps)
		case http.MethodPost:
			var body sharedfolders.SnapshotRequest
			if r.ContentLength != 0 {
				_ = decodeJSON(r, &body)
			}
			snap, err := a.sharedFolders.CreateSnapshot(r.Context(), id, body)
			if err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_snapshot_create_failed", err.Error())
				return
			}
			platform.WriteJSON(w, r, http.StatusOK, snap)
		default:
			platform.WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		}
	case len(parts) == 3 && parts[1] == "snapshots":
		if !allowMethod(w, r, http.MethodDelete) {
			return
		}
		if err := a.sharedFolders.DeleteSnapshot(r.Context(), id, parts[2]); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_snapshot_delete_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"deleted": true})
	case len(parts) == 3 && parts[1] == "delete" && parts[2] == "preview":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		var body struct {
			Actor string `json:"actor"`
		}
		if r.ContentLength != 0 {
			_ = decodeJSON(r, &body)
		}
		preview, err := a.sharedFolders.PreviewDelete(r.Context(), id, body.Actor)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_delete_preview_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, preview)
	case len(parts) == 3 && parts[1] == "delete" && parts[2] == "confirm":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		var body sharedfolders.ConfirmDeleteRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		if err := a.sharedFolders.ConfirmDelete(r.Context(), body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_delete_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"deleted": true})
	default:
		platform.WriteError(w, r, http.StatusNotFound, "shared_folder_route_not_found", "shared folder route not found")
	}
}

// sharedFolderSubjectPermissions sets one subject's access across many folders
// at once (the "shared-folder permissions" tab of a user/group editor).
func (a *API) sharedFolderSubjectPermissions(w http.ResponseWriter, r *http.Request) {
	if a.sharedFolders == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "shared_folders_unavailable", "shared folders service is unavailable")
		return
	}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		SubjectType string                                  `json:"subjectType"`
		SubjectID   string                                  `json:"subjectId"`
		Perms       []sharedfolders.SubjectFolderPermission `json:"perms"`
		Actor       string                                  `json:"actor"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := a.sharedFolders.SetSubjectPermissions(r.Context(), accounts.SubjectType(body.SubjectType), body.SubjectID, body.Perms, body.Actor); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "shared_folder_subject_permissions_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) accountsSambaSync(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.accounts == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "accounts_unavailable", "accounts service is unavailable")
		return
	}
	report, err := a.accounts.SyncSamba(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "samba_sync_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, report)
}
