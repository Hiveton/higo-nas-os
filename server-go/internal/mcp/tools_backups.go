package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

type BackupsJobRunInput struct {
	ID string `json:"id" jsonschema:"backup job id to run now"`
}

type BackupsJobPauseInput struct {
	ID string `json:"id" jsonschema:"backup job id to pause"`
}

type BackupsJobResumeInput struct {
	ID string `json:"id" jsonschema:"backup job id to resume"`
}

type BackupsJobVerifyInput struct {
	ID string `json:"id" jsonschema:"backup job id to verify"`
}

func registerBackups(r *registry) {
	addTool(r, "backups", "higo.backups.jobs.list",
		"List configured backup jobs and their status.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.BackupJobs(ctx)
		})

	addTool(r, "backups", "higo.backups.jobs.run",
		"Run a backup job immediately.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in BackupsJobRunInput) (json.RawMessage, error) {
			return c.BackupJobRun(ctx, in.ID)
		})

	addTool(r, "backups", "higo.backups.jobs.pause",
		"Pause a running backup job.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in BackupsJobPauseInput) (json.RawMessage, error) {
			return c.BackupJobPause(ctx, in.ID)
		})

	addTool(r, "backups", "higo.backups.jobs.resume",
		"Resume a paused backup job.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in BackupsJobResumeInput) (json.RawMessage, error) {
			return c.BackupJobResume(ctx, in.ID)
		})

	addTool(r, "backups", "higo.backups.jobs.verify",
		"Verify the integrity of a backup job's data.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in BackupsJobVerifyInput) (json.RawMessage, error) {
			return c.BackupJobVerify(ctx, in.ID)
		})
}
