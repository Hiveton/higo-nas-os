package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// VideoLibrary returns video library settings (GET /api/v1/videos/library).
func (c *Client) VideoLibrary(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/library", nil, nil)
}

// VideoLibraryUpdate replaces video library settings (PUT /api/v1/videos/library).
func (c *Client) VideoLibraryUpdate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/videos/library", nil, body)
}

// VideoLibraryCreate creates a video library (POST /api/v1/videos/library).
func (c *Client) VideoLibraryCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/videos/library", nil, body)
}

// VideoLibraryDelete removes a video library (DELETE /api/v1/videos/library/{id}).
func (c *Client) VideoLibraryDelete(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/videos/library/"+url.PathEscape(id), nil, nil)
}

// VideoScan triggers a video library scan (POST /api/v1/videos/scan).
func (c *Client) VideoScan(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/videos/scan", nil, nil)
}

// VideoItems lists video items (GET /api/v1/videos/items).
func (c *Client) VideoItems(ctx context.Context, q, libraryID, kind string) (json.RawMessage, error) {
	var query url.Values
	if q != "" || libraryID != "" || kind != "" {
		query = url.Values{}
		if q != "" {
			query.Set("q", q)
		}
		if libraryID != "" {
			query.Set("libraryId", libraryID)
		}
		if kind != "" {
			query.Set("kind", kind)
		}
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/items", query, nil)
}

// VideoItem returns a single video item (GET /api/v1/videos/items/{id}).
func (c *Client) VideoItem(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/items/"+url.PathEscape(id), nil, nil)
}

// VideoTasks lists video processing tasks (GET /api/v1/videos/tasks).
func (c *Client) VideoTasks(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/tasks", nil, nil)
}

// VideoScrapeTask creates a metadata scrape task (POST /api/v1/videos/tasks/scrape).
func (c *Client) VideoScrapeTask(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/videos/tasks/scrape", nil, body)
}

// VideoSubtitleTask creates a subtitle task (POST /api/v1/videos/tasks/subtitle).
func (c *Client) VideoSubtitleTask(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/videos/tasks/subtitle", nil, body)
}

// VideoTranscodeTask creates a transcode task (POST /api/v1/videos/tasks/transcode).
func (c *Client) VideoTranscodeTask(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/videos/tasks/transcode", nil, body)
}

// VideoLiveSources lists live TV sources (GET /api/v1/videos/live/sources).
func (c *Client) VideoLiveSources(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/live/sources", nil, nil)
}

// VideoLiveSourceCreate creates a live TV source (POST /api/v1/videos/live/sources).
func (c *Client) VideoLiveSourceCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/videos/live/sources", nil, body)
}

// VideoLiveChannels lists live TV channels (GET /api/v1/videos/live/channels).
func (c *Client) VideoLiveChannels(ctx context.Context, sourceID string) (json.RawMessage, error) {
	var query url.Values
	if sourceID != "" {
		query = url.Values{}
		query.Set("sourceId", sourceID)
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/live/channels", query, nil)
}

// VideoLiveGuideSources lists EPG guide sources (GET /api/v1/videos/live/guide-sources).
func (c *Client) VideoLiveGuideSources(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/live/guide-sources", nil, nil)
}

// VideoLiveGuideSourceCreate creates an EPG guide source (POST /api/v1/videos/live/guide-sources).
func (c *Client) VideoLiveGuideSourceCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/videos/live/guide-sources", nil, body)
}

// VideoLivePrograms lists EPG programs (GET /api/v1/videos/live/programs).
func (c *Client) VideoLivePrograms(ctx context.Context, sourceID, channelID, from, to string) (json.RawMessage, error) {
	var query url.Values
	if sourceID != "" || channelID != "" || from != "" || to != "" {
		query = url.Values{}
		if sourceID != "" {
			query.Set("sourceId", sourceID)
		}
		if channelID != "" {
			query.Set("channelId", channelID)
		}
		if from != "" {
			query.Set("from", from)
		}
		if to != "" {
			query.Set("to", to)
		}
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/live/programs", query, nil)
}

// VideoLiveDVRSettings returns DVR settings (GET /api/v1/videos/live/dvr/settings).
func (c *Client) VideoLiveDVRSettings(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/live/dvr/settings", nil, nil)
}

// VideoLiveDVRSettingsUpdate replaces DVR settings (PUT /api/v1/videos/live/dvr/settings).
func (c *Client) VideoLiveDVRSettingsUpdate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/videos/live/dvr/settings", nil, body)
}

// VideoLiveRecordingTimers lists DVR recording timers (GET /api/v1/videos/live/recording-timers).
func (c *Client) VideoLiveRecordingTimers(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/live/recording-timers", nil, nil)
}

// VideoLiveRecordingTimerCreate creates a DVR recording timer (POST /api/v1/videos/live/recording-timers).
func (c *Client) VideoLiveRecordingTimerCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/videos/live/recording-timers", nil, body)
}

// VideoLiveRecordingTimerCancel cancels a DVR recording timer (DELETE /api/v1/videos/live/recording-timers/{id}).
func (c *Client) VideoLiveRecordingTimerCancel(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/videos/live/recording-timers/"+url.PathEscape(id), nil, nil)
}

// VideoLiveRecordings lists completed DVR recordings (GET /api/v1/videos/live/recordings).
func (c *Client) VideoLiveRecordings(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/videos/live/recordings", nil, nil)
}
