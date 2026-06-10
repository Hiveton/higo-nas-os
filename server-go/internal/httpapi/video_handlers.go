package httpapi

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"

	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/video"
)

func (a *API) videoLibrary(w http.ResponseWriter, r *http.Request) {
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		settings, err := a.video.Settings(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "video_library_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, settings)
	case http.MethodPut:
		var body video.LibraryUpdateRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		settings, err := a.video.UpdateSettings(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "video_library_update_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, settings)
	case http.MethodPost:
		var body video.CreateLibraryRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		library, err := a.video.CreateLibrary(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "video_library_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusCreated, library)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPut, http.MethodPost)
	}
}

func (a *API) videoLibraryByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/videos/library/"), "/")
	if id == "" {
		platform.WriteError(w, r, http.StatusNotFound, "video_library_not_found", "video library not found")
		return
	}
	result, err := a.video.DeleteLibrary(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "video_library_delete_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, result)
}

func (a *API) videoScan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	result, err := a.video.Scan(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "video_scan_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, result)
}

func (a *API) videoItems(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	query := r.URL.Query()
	items, err := a.video.Items(r.Context(), query.Get("q"), query.Get("libraryId"), query.Get("kind"))
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "video_items_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, items)
}

func (a *API) videoItemByID(w http.ResponseWriter, r *http.Request) {
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/videos/items/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "video_route_not_found", "video route not found")
		return
	}
	id := parts[0]
	if len(parts) == 1 {
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		item, err := a.video.Item(r.Context(), id)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "video_item_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, item)
		return
	}
	if len(parts) != 2 {
		platform.WriteError(w, r, http.StatusNotFound, "video_route_not_found", "video route not found")
		return
	}
	switch parts[1] {
	case "stream":
		a.videoItemStream(w, r, id)
	case "poster":
		a.videoItemPoster(w, r, id)
	case "subtitle":
		a.videoItemSubtitle(w, r, id)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "video_route_not_found", "video route not found")
	}
}

func (a *API) videoItemStream(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	file, item, err := a.video.OpenItem(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "video_item_not_found", err.Error())
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "video_stream_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", videoContentType(item.FileName))
	w.Header().Set("Content-Disposition", "inline; filename="+strconvQuote(filepath.Base(item.FileName)))
	http.ServeContent(w, r, filepath.Base(item.FileName), info.ModTime(), file)
}

func (a *API) videoItemPoster(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	data, contentType, modTime, err := a.video.Poster(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "video_poster_not_found", err.Error())
		return
	}
	video.ServeBytes(w, r, id+"-poster", data, contentType, modTime)
}

func (a *API) videoItemSubtitle(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	data, contentType, modTime, err := a.video.Subtitle(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "video_subtitle_not_found", err.Error())
		return
	}
	video.ServeBytes(w, r, id+"-subtitle", data, contentType, modTime)
}

func (a *API) videoTasks(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	tasks, err := a.video.Tasks(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "video_tasks_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, tasks)
}

func (a *API) videoScrapeTask(w http.ResponseWriter, r *http.Request) {
	a.videoCreateTask(w, r, a.video.CreateScrapeTask, "video_scrape_failed")
}

func (a *API) videoSubtitleTask(w http.ResponseWriter, r *http.Request) {
	a.videoCreateTask(w, r, a.video.CreateSubtitleTask, "video_subtitle_failed")
}

func (a *API) videoTranscodeTask(w http.ResponseWriter, r *http.Request) {
	a.videoCreateTask(w, r, a.video.CreateTranscodeTask, "video_transcode_failed")
}

func (a *API) videoCreateTask(w http.ResponseWriter, r *http.Request, create func(context.Context, video.CreateTaskRequest) (video.Task, error), code string) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	var body video.CreateTaskRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	task, err := create(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, code, err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusCreated, task)
}

func (a *API) videoLiveSources(w http.ResponseWriter, r *http.Request) {
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		sources, err := a.video.LiveSources(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "video_live_sources_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, sources)
	case http.MethodPost:
		var body video.CreateLiveSourceRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		source, err := a.video.CreateLiveSource(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "video_live_source_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusCreated, source)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) videoLiveChannels(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	channels, err := a.video.LiveChannels(r.Context(), r.URL.Query().Get("sourceId"))
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "video_live_channels_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, channels)
}

func (a *API) videoLiveGuideSources(w http.ResponseWriter, r *http.Request) {
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		sources, err := a.video.GuideSources(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "video_guide_sources_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, sources)
	case http.MethodPost:
		var body video.CreateGuideSourceRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		source, err := a.video.CreateGuideSource(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "video_guide_source_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusCreated, source)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) videoLivePrograms(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	programs, err := a.video.LivePrograms(r.Context(), video.LiveProgramQuery{
		SourceID:  r.URL.Query().Get("sourceId"),
		ChannelID: r.URL.Query().Get("channelId"),
		From:      r.URL.Query().Get("from"),
		To:        r.URL.Query().Get("to"),
	})
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "video_programs_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, programs)
}

func (a *API) videoLiveDVRSettings(w http.ResponseWriter, r *http.Request) {
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		settings, err := a.video.DVRSettings(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "video_dvr_settings_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, settings)
	case http.MethodPut:
		var body video.DVRSettings
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		settings, err := a.video.UpdateDVRSettings(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "video_dvr_settings_update_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, settings)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPut)
	}
}

func (a *API) videoLiveRecordingTimers(w http.ResponseWriter, r *http.Request) {
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		timers, err := a.video.RecordingTimers(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "video_recording_timers_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, timers)
	case http.MethodPost:
		var body video.CreateRecordingTimerRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		timer, err := a.video.CreateRecordingTimer(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "video_recording_timer_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusCreated, timer)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) videoLiveRecordingTimerByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/videos/live/recording-timers/")
	timer, err := a.video.CancelRecordingTimer(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "video_recording_timer_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, timer)
}

func (a *API) videoLiveRecordings(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.video == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "video_unavailable", "video service is unavailable")
		return
	}
	recordings, err := a.video.Recordings(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "video_recordings_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, recordings)
}

func videoContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	case ".ts", ".m2ts":
		return "video/mp2t"
	default:
		return "application/octet-stream"
	}
}
