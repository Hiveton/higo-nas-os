package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// DownloadTasks lists download tasks (GET /api/v1/downloads/tasks).
func (c *Client) DownloadTasks(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/downloads/tasks", nil, nil)
}

// DownloadCreateTask creates a download task (POST /api/v1/downloads/tasks).
func (c *Client) DownloadCreateTask(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/downloads/tasks", nil, body)
}

// DownloadDeleteTask deletes a download task (DELETE /api/v1/downloads/tasks/{id}).
func (c *Client) DownloadDeleteTask(ctx context.Context, id string, deleteFile bool) (json.RawMessage, error) {
	var query url.Values
	if deleteFile {
		query = url.Values{}
		query.Set("deleteFile", "true")
	}
	return c.Do(ctx, http.MethodDelete, "/api/v1/downloads/tasks/"+url.PathEscape(id), query, nil)
}

// DownloadPauseTask pauses a download task (POST /api/v1/downloads/tasks/{id}/pause).
func (c *Client) DownloadPauseTask(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/downloads/tasks/"+url.PathEscape(id)+"/pause", nil, nil)
}

// DownloadResumeTask resumes a download task (POST /api/v1/downloads/tasks/{id}/resume).
func (c *Client) DownloadResumeTask(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/downloads/tasks/"+url.PathEscape(id)+"/resume", nil, nil)
}

// DownloadArchiveTask archives a download task (POST /api/v1/downloads/tasks/{id}/archive).
func (c *Client) DownloadArchiveTask(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/downloads/tasks/"+url.PathEscape(id)+"/archive", nil, nil)
}

// DownloadSpeedProfiles lists download speed profiles (GET /api/v1/downloads/speed-profiles).
func (c *Client) DownloadSpeedProfiles(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/downloads/speed-profiles", nil, nil)
}

// DownloadSetSpeedProfile sets the active download speed profile (PUT /api/v1/downloads/speed-profile).
func (c *Client) DownloadSetSpeedProfile(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/downloads/speed-profile", nil, body)
}
