package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"higoos/server-go/internal/aianalysis"
	"higoos/server-go/internal/platform"
)

// aiAnalysisStatus reports the aggregate background-analysis status: current
// level, per-domain progress, provider/ffmpeg capability and pause state.
func (a *API) aiAnalysisStatus(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.aianalysis.Status())
}

// aiAnalysisRecords returns a page of analysis records, optionally filtered by
// domain and state.
func (a *API) aiAnalysisRecords(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	query := r.URL.Query()
	domain := aianalysis.DomainKind(strings.TrimSpace(query.Get("domain")))
	state := aianalysis.ItemState(strings.TrimSpace(query.Get("state")))
	q := strings.TrimSpace(query.Get("q"))
	page, _ := strconv.Atoi(query.Get("page"))
	size, _ := strconv.Atoi(query.Get("size"))
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 50
	}
	records, total := a.aianalysis.Records(domain, state, q, page, size)
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"records":  records,
		"total":    total,
		"page":     page,
		"pageSize": size,
	})
}

// aiAnalysisRecordByKey returns one record's full state by its domain-prefixed
// key (e.g. "file:abc"). The key arrives URL-encoded because it contains a colon.
func (a *API) aiAnalysisRecordByKey(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	raw := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/ai-analysis/records/"), "/")
	key, err := url.PathUnescape(raw)
	if err != nil {
		key = raw
	}
	if key == "" {
		platform.WriteError(w, r, http.StatusBadRequest, "missing_key", "缺少记录 key")
		return
	}
	rec, ok := a.aianalysis.RecordByKey(key)
	if !ok {
		platform.WriteError(w, r, http.StatusNotFound, "record_not_found", "未找到分析记录")
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, rec)
}

// aiAnalysisBatch performs a batch operation over a set of record keys. Today the
// only op is "reanalyze" (reset to pending and dispatch).
func (a *API) aiAnalysisBatch(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	var body struct {
		Op   string   `json:"op"`
		Keys []string `json:"keys"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	op := strings.TrimSpace(body.Op)
	if op == "" {
		op = "reanalyze"
	}
	if op != "reanalyze" {
		platform.WriteError(w, r, http.StatusBadRequest, "unsupported_op", "不支持的批量操作: "+op)
		return
	}
	reset := a.aianalysis.ReanalyzeKeys(body.Keys)
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"reset": reset, "status": a.aianalysis.Status()})
}

// aiAnalysisReanalyze forces re-analysis of one item, a whole domain or
// everything, then dispatches the work.
func (a *API) aiAnalysisReanalyze(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	var body struct {
		Scope  string `json:"scope"`
		ItemID string `json:"itemId"`
		Domain string `json:"domain"`
	}
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	var err error
	switch strings.TrimSpace(body.Scope) {
	case "item":
		if strings.TrimSpace(body.ItemID) == "" {
			platform.WriteError(w, r, http.StatusBadRequest, "missing_item", "缺少 itemId")
			return
		}
		err = a.aianalysis.Reanalyze("", body.ItemID)
	case "domain":
		err = a.aianalysis.Reanalyze(aianalysis.DomainKind(strings.TrimSpace(body.Domain)), "")
	default: // all
		err = a.aianalysis.Reanalyze("", "")
	}
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "reanalyze_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.aianalysis.Status())
}

// aiAnalysisRescan triggers an immediate enumerate + dispatch pass.
func (a *API) aiAnalysisRescan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	go a.aianalysis.Rescan(context.Background())
	platform.WriteJSON(w, r, http.StatusOK, a.aianalysis.Status())
}

// aiAnalysisPause halts dispatching of new analysis work.
func (a *API) aiAnalysisPause(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	a.aianalysis.Pause()
	platform.WriteJSON(w, r, http.StatusOK, a.aianalysis.Status())
}

// aiAnalysisResume resumes dispatching and triggers a rescan.
func (a *API) aiAnalysisResume(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	a.aianalysis.Resume()
	platform.WriteJSON(w, r, http.StatusOK, a.aianalysis.Status())
}

// aiAnalysisFaces reports the self-training framework: configured embedder /
// trainer, current face clusters, labeled-dataset stats, and trained models.
func (a *API) aiAnalysisFaces(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.aianalysis.FaceFramework())
}

// aiAnalysisFaceLabel names a face cluster, confirming its training samples.
func (a *API) aiAnalysisFaceLabel(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	var body struct {
		ClusterID string `json:"clusterId"`
		Name      string `json:"name"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	confirmed, err := a.aianalysis.LabelFace(body.ClusterID, body.Name)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "label_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{
		"confirmed": confirmed,
		"faces":     a.aianalysis.FaceFramework(),
	})
}

// aiAnalysisFaceRetrain enqueues a self-training run from the confirmed dataset.
func (a *API) aiAnalysisFaceRetrain(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	taskID, err := a.aianalysis.Retrain()
	if err != nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "trainer_unavailable", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"taskId": taskID})
}

// aiAnalysisProgressStream streams aggregate status snapshots over SSE as
// analysis tasks make progress, with a periodic heartbeat to keep the
// connection alive.
func (a *API) aiAnalysisProgressStream(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.aianalysis == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "aianalysis_unavailable", "AI 分析引擎不可用")
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		platform.WriteError(w, r, http.StatusInternalServerError, "stream_unsupported", "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	writeStatus := func(status aianalysis.EngineStatus) bool {
		payload, err := json.Marshal(status)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", payload); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	// Initial snapshot.
	if !writeStatus(a.aianalysis.Status()) {
		return
	}

	updates, cancel := a.aianalysis.Subscribe()
	defer cancel()

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case status, open := <-updates:
			if !open {
				return
			}
			if !writeStatus(status) {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
