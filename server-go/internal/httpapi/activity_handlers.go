package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"higoos/server-go/internal/activity"
	"higoos/server-go/internal/platform"
)

// activityLog serves the user-activity ledger:
//   - GET    /api/v1/activity?type=&limit=  → { entries: [...] }
//   - POST   /api/v1/activity               → append a batch { entries: [...] } (or a bare array)
//   - DELETE /api/v1/activity               → clear all entries
func (a *API) activityLog(w http.ResponseWriter, r *http.Request) {
	if a.activity == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "activity_unavailable", "activity log is not configured")
		return
	}
	switch r.Method {
	case http.MethodGet:
		typ := strings.TrimSpace(r.URL.Query().Get("type"))
		limit := 0
		if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				limit = n
			}
		}
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"entries": a.activity.List(typ, limit)})
	case http.MethodPost:
		var payload struct {
			Entries []activity.Entry `json:"entries"`
		}
		if err := decodeJSON(r, &payload); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "activity_invalid", "invalid activity payload")
			return
		}
		stored := a.activity.Append(payload.Entries)
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"entries": stored, "count": len(stored)})
	case http.MethodDelete:
		a.activity.Clear()
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"cleared": true})
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost, http.MethodDelete)
	}
}
