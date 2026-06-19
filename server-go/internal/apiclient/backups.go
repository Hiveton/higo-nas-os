package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// BackupJobs lists backup jobs (GET /api/v1/backups/jobs).
func (c *Client) BackupJobs(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/backups/jobs", nil, nil)
}

// BackupJobRun runs a backup job now (POST /api/v1/backups/jobs/{id}/run).
func (c *Client) BackupJobRun(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/backups/jobs/" + url.PathEscape(id) + "/run"
	return c.Do(ctx, http.MethodPost, path, nil, nil)
}

// BackupJobPause pauses a backup job (POST /api/v1/backups/jobs/{id}/pause).
func (c *Client) BackupJobPause(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/backups/jobs/" + url.PathEscape(id) + "/pause"
	return c.Do(ctx, http.MethodPost, path, nil, nil)
}

// BackupJobResume resumes a backup job (POST /api/v1/backups/jobs/{id}/resume).
func (c *Client) BackupJobResume(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/backups/jobs/" + url.PathEscape(id) + "/resume"
	return c.Do(ctx, http.MethodPost, path, nil, nil)
}

// BackupJobVerify verifies a backup job (POST /api/v1/backups/jobs/{id}/verify).
func (c *Client) BackupJobVerify(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/backups/jobs/" + url.PathEscape(id) + "/verify"
	return c.Do(ctx, http.MethodPost, path, nil, nil)
}

// BackupJobSchedule configures a job's automatic-backup policy
// (POST /api/v1/backups/jobs/{id}/schedule).
func (c *Client) BackupJobSchedule(ctx context.Context, id string, body any) (json.RawMessage, error) {
	path := "/api/v1/backups/jobs/" + url.PathEscape(id) + "/schedule"
	return c.Do(ctx, http.MethodPost, path, nil, body)
}
