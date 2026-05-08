package httpapi

import (
	"fmt"
	"net/http"
	"time"

	"higoos/server-go/internal/platform"
)

func (a *API) systemUpdates(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"channel":        "stable",
		"current":        a.config.Version,
		"latest":         a.config.Version,
		"updateStatus":   "已是最新",
		"lastCheckedAt":  time.Now().UTC(),
		"requiresReboot": false,
	})
}

func (a *API) systemUpdateCheck(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(
		"system-update-check",
		"queued",
		"system update check queued for the Linux adapter",
	))
}

func (a *API) systemBackups(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(
		fmt.Sprintf("system-backup-%d", time.Now().UTC().Unix()),
		"queued",
		"configuration and metadata backup queued",
	))
}

func (a *API) eventsStream(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "event: system.ready\n")
	_, _ = fmt.Fprintf(w, "data: {\"status\":\"ready\",\"version\":%q}\n\n", a.config.Version)
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}
