package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"higoos/server-go/internal/files"
	"higoos/server-go/internal/monitoring"
	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/settings"
)

func (a *API) filesTree(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.files == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "files_unavailable", "files service is unavailable")
		return
	}
	tree, err := a.files.Tree(r.Context(), r.URL.Query().Get("space"))
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "space_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, tree)
}

func (a *API) filesSearch(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.files == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "files_unavailable", "files service is unavailable")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	rows, err := a.files.Search(r.Context(), files.SearchQuery{
		Query: r.URL.Query().Get("q"),
		Space: r.URL.Query().Get("space"),
		Type:  r.URL.Query().Get("type"),
		Tags:  splitCSV(r.URL.Query()["tags"]),
		Limit: limit,
	})
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "files_search_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, rows)
}

func (a *API) fileByID(w http.ResponseWriter, r *http.Request) {
	if a.files == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "files_unavailable", "files service is unavailable")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/files/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "file_not_found", "file id is required")
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		row, err := a.files.Get(r.Context(), id)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "file_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, row)
		return
	}
	if len(parts) != 2 {
		platform.WriteError(w, r, http.StatusNotFound, "file_route_not_found", "file route not found")
		return
	}
	switch parts[1] {
	case "preview":
		a.filePreview(w, r, id)
	case "tags":
		a.fileAddTags(w, r, id)
	case "shares":
		a.fileCreateShare(w, r, id)
	case "restore":
		a.fileRestore(w, r, id)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "file_route_not_found", "file route not found")
	}
}

func (a *API) filePreview(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	preview, err := a.files.Preview(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "file_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, preview)
}

func (a *API) fileAddTags(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body files.TagMutation
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	body.FileID = id
	row, err := a.files.AddTags(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "file_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, row)
}

func (a *API) fileCreateShare(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body struct {
		files.ShareLink
		ExpiresInDays int    `json:"expiresInDays"`
		Actor         string `json:"actor"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	shareRequest := body.ShareLink
	shareRequest.FileID = id
	if body.ExpiresInDays > 0 && shareRequest.ExpiresAt.IsZero() {
		shareRequest.ExpiresAt = time.Now().UTC().Add(time.Duration(body.ExpiresInDays) * 24 * time.Hour)
	}
	if body.Actor != "" && shareRequest.Audit == "" {
		shareRequest.Audit = body.Actor + " created share link"
	}
	share, err := a.files.CreateShare(r.Context(), shareRequest)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "file_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, share)
}

func (a *API) fileRestore(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"id":      "restore-" + id,
		"status":  "planned",
		"message": "restore plan created from recycle metadata",
	})
}

func (a *API) filesBatchMove(w http.ResponseWriter, r *http.Request) {
	a.filesBatch(w, r, "move")
}

func (a *API) filesBatchRename(w http.ResponseWriter, r *http.Request) {
	a.filesBatch(w, r, "rename")
}

func (a *API) filesBatchDelete(w http.ResponseWriter, r *http.Request) {
	a.filesBatch(w, r, "delete")
}

func (a *API) filesBatch(w http.ResponseWriter, r *http.Request, kind string) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.files == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "files_unavailable", "files service is unavailable")
		return
	}
	var body files.BatchOperation
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	var (
		task files.Task
		err  error
	)
	switch kind {
	case "move":
		task, err = a.files.BatchMove(r.Context(), body)
	case "rename":
		task, err = a.files.BatchRename(r.Context(), body)
	case "delete":
		task, err = a.files.BatchDelete(r.Context(), body)
	}
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "files_batch_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, task)
}

func (a *API) monitoringCurrentMetrics(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	snapshot, err := a.monitoring.CurrentMetrics(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "metrics_unavailable", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, snapshot.Metrics)
}

func (a *API) monitoringMetricTrend(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "cpu"
	}
	trend, err := a.monitoring.MetricTrend(r.Context(), metric, parseTimeRange(r.URL.Query().Get("range")))
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "metric_trend_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, trend.Points)
}

func (a *API) monitoringLogs(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	logs, err := a.monitoring.Logs(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "logs_unavailable", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, logs)
}

func (a *API) monitoringAlerts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		alerts, err := a.monitoring.Alerts(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "alerts_unavailable", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, alerts)
	case http.MethodPost:
		var body monitoring.CreateAlertRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		alert, err := a.monitoring.CreateAlert(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "alert_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, alert)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) monitoringAlertByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/monitoring/alerts/"), "/mute")
	if id == "" || id == r.URL.Path {
		platform.WriteError(w, r, http.StatusNotFound, "alert_route_not_found", "alert route not found")
		return
	}
	var body struct {
		Muted *bool `json:"muted"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	muted := true
	if body.Muted != nil {
		muted = *body.Muted
	}
	alert, err := a.monitoring.MuteAlert(r.Context(), strings.Trim(id, "/"), muted)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "alert_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, alert)
}

func (a *API) monitoringDiagnostics(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	run, err := a.monitoring.RunDiagnostics(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "diagnostics_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"id":      run.ID,
		"status":  run.Status,
		"message": run.Summary,
	})
}

func (a *API) settingsRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		platform.WriteJSON(w, r, http.StatusOK, a.settings.Get())
	case http.MethodPut:
		var body settings.Settings
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		next, err := a.settings.Update(body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "settings_invalid", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, next)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPut)
	}
}

func (a *API) settingsDefaults(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	settings, err := a.settings.RestoreDefaults()
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "settings_persist_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, settings)
}

func parseTimeRange(value string) monitoring.TimeRange {
	switch strings.ToLower(value) {
	case "6h":
		return monitoring.Range6H
	case "24h", "1d":
		return monitoring.Range24H
	case "7d":
		return monitoring.Range7D
	default:
		return monitoring.Range1H
	}
}

func splitCSV(values []string) []string {
	var out []string
	for _, value := range values {
		for _, item := range strings.Split(value, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}
