package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// DownloadsCreateInput is the input for higo.downloads.tasks.create.
type DownloadsCreateInput struct {
	Source   string `json:"source" jsonschema:"download source type"`
	Link     string `json:"link" jsonschema:"download link or magnet"`
	Name     string `json:"name" jsonschema:"download task name"`
	Category string `json:"category" jsonschema:"download category"`
}

// DownloadsDeleteInput is the input for higo.downloads.tasks.delete.
type DownloadsDeleteInput struct {
	ID         string `json:"id" jsonschema:"download task id"`
	DeleteFile bool   `json:"deleteFile,omitempty" jsonschema:"also delete the downloaded file"`
}

// DownloadsTaskInput is the input for download task actions targeting one task.
type DownloadsTaskInput struct {
	ID string `json:"id" jsonschema:"download task id"`
}

// DownloadsSetSpeedProfileInput is the input for higo.downloads.speed-profile.set.
type DownloadsSetSpeedProfileInput struct {
	Name string `json:"name" jsonschema:"speed profile name to activate"`
}

func registerDownloads(r *registry) {
	addTool(r, "downloads", "higo.downloads.tasks.list",
		"List download tasks.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DownloadTasks(ctx)
		})

	addTool(r, "downloads", "higo.downloads.tasks.create",
		"Create a download task from a link or magnet.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DownloadsCreateInput) (json.RawMessage, error) {
			return c.DownloadCreateTask(ctx, in)
		})

	addTool(r, "downloads", "higo.downloads.tasks.delete",
		"Delete a download task, optionally removing the file.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in DownloadsDeleteInput) (json.RawMessage, error) {
			return c.DownloadDeleteTask(ctx, in.ID, in.DeleteFile)
		})

	addTool(r, "downloads", "higo.downloads.tasks.pause",
		"Pause a running download task.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DownloadsTaskInput) (json.RawMessage, error) {
			return c.DownloadPauseTask(ctx, in.ID)
		})

	addTool(r, "downloads", "higo.downloads.tasks.resume",
		"Resume a paused download task.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DownloadsTaskInput) (json.RawMessage, error) {
			return c.DownloadResumeTask(ctx, in.ID)
		})

	addTool(r, "downloads", "higo.downloads.tasks.archive",
		"Archive a completed download task.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DownloadsTaskInput) (json.RawMessage, error) {
			return c.DownloadArchiveTask(ctx, in.ID)
		})

	addTool(r, "downloads", "higo.downloads.speed-profiles.list",
		"List download speed profiles.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DownloadSpeedProfiles(ctx)
		})

	addTool(r, "downloads", "higo.downloads.speed-profile.set",
		"Set the active download speed profile by name.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DownloadsSetSpeedProfileInput) (json.RawMessage, error) {
			return c.DownloadSetSpeedProfile(ctx, map[string]any{"name": in.Name})
		})
}
