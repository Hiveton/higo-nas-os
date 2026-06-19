package httpapi

import (
	"net/http"

	"higoos/server-cloud/internal/platform"
)

// GET /healthz and GET /v1/health — liveness + build info. Open (no auth).
func (a *API) health(w http.ResponseWriter, r *http.Request) {
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"ok":      true,
		"app":     a.config.AppName,
		"version": a.config.Version,
		"env":     a.config.Environment,
	})
}
