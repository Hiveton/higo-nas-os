package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// TasksListInput is the input for higo.tasks.list.
type TasksListInput struct {
	Kind string `json:"kind,omitempty" jsonschema:"optional task kind filter"`
}

// TaskGetInput is the input for higo.tasks.get.
type TaskGetInput struct {
	ID string `json:"id" jsonschema:"background task id"`
}

// TaskCancelInput is the input for higo.tasks.cancel.
type TaskCancelInput struct {
	ID string `json:"id" jsonschema:"background task id to cancel (only queued tasks)"`
}

func registerTasks(r *registry) {
	addTool(r, "tasks", "higo.tasks.list",
		"List background tasks from the task runtime, optionally filtered by kind.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in TasksListInput) (json.RawMessage, error) {
			return c.Tasks(ctx, in.Kind)
		})
	addTool(r, "tasks", "higo.tasks.get",
		"Get a single background task's status by id.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in TaskGetInput) (json.RawMessage, error) {
			return c.Task(ctx, in.ID)
		})
	addTool(r, "tasks", "higo.tasks.cancel",
		"Cancel a queued background task by id.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in TaskCancelInput) (json.RawMessage, error) {
			return c.TaskCancel(ctx, in.ID)
		})
}
