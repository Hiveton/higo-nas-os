package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/tasks"
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

// eventsStream is the desktop's global SSE feed. It opens with a system.ready
// frame, then stays live, multiplexing task lifecycle updates (queued → running
// → done/failed) so the shell's notifications and progress badges react in real
// time. A periodic comment keeps the connection alive through proxies, and the
// loop unwinds as soon as the client disconnects.
func (a *API) eventsStream(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		platform.WriteError(w, r, http.StatusInternalServerError, "stream_unsupported", "streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	fmt.Fprintf(w, "event: system.ready\n")
	fmt.Fprintf(w, "data: {\"status\":\"ready\",\"version\":%q}\n\n", a.config.Version)
	flusher.Flush()

	// Without a task manager there are no live events to relay; keep the
	// connection open with heartbeats so the client's reconnect logic rests.
	var ch <-chan tasks.Task
	if a.tasks != nil {
		c, unsubscribe := a.tasks.Subscribe()
		defer unsubscribe()
		ch = c
	}

	writeEvent := func(name string, payload any) {
		buf, err := json.Marshal(payload)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "event: %s\n", name)
		fmt.Fprintf(w, "data: %s\n\n", buf)
		flusher.Flush()
	}

	ctx := r.Context()
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-ch:
			if !ok {
				return
			}
			writeEvent("task.update", task)
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}
