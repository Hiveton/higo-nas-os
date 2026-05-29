package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"

	"higoos/server-go/internal/music"
	"higoos/server-go/internal/platform"
)

func (a *API) musicLibrary(w http.ResponseWriter, r *http.Request) {
	if a.music == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "music_unavailable", "music service is unavailable")
		return
	}
	switch r.Method {
	case http.MethodGet:
		settings, err := a.music.Settings(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "music_library_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, settings)
	case http.MethodPut:
		var body music.LibraryUpdateRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		settings, err := a.music.UpdateSettings(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "music_library_update_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, settings)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPut)
	}
}

func (a *API) musicScan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	if a.music == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "music_unavailable", "music service is unavailable")
		return
	}
	result, err := a.music.Scan(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "music_scan_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, result)
}

func (a *API) musicTracks(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.music == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "music_unavailable", "music service is unavailable")
		return
	}
	tracks, err := a.music.Tracks(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "music_tracks_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, tracks)
}

func (a *API) musicAlbums(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if a.music == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "music_unavailable", "music service is unavailable")
		return
	}
	albums, err := a.music.Albums(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "music_albums_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, albums)
}

func (a *API) musicTrackByID(w http.ResponseWriter, r *http.Request) {
	if a.music == nil {
		platform.WriteError(w, r, http.StatusServiceUnavailable, "music_unavailable", "music service is unavailable")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/music/tracks/"), "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "music_route_not_found", "music route not found")
		return
	}
	id := parts[0]
	switch parts[1] {
	case "stream":
		a.musicTrackStream(w, r, id)
	case "cover":
		a.musicTrackCover(w, r, id)
	case "lyrics":
		a.musicTrackLyrics(w, r, id)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "music_route_not_found", "music route not found")
	}
}

func (a *API) musicTrackStream(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	file, track, contentType, err := a.music.OpenTrack(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "music_track_not_found", err.Error())
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "music_stream_failed", err.Error())
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline; filename="+strconvQuote(filepath.Base(track.FileName)))
	http.ServeContent(w, r, filepath.Base(track.FileName), info.ModTime(), file)
}

func (a *API) musicTrackCover(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	cover, contentType, name, modTime, err := a.music.OpenCover(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "music_cover_not_found", err.Error())
		return
	}
	defer cover.Close()
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, name, modTime, cover)
}

func (a *API) musicTrackLyrics(w http.ResponseWriter, r *http.Request, id string) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	lyrics, err := a.music.Lyrics(r.Context(), id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "music_lyrics_not_found", err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(lyrics))
}

func strconvQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `\"`) + `"`
}
