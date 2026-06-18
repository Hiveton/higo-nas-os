package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

func registerSystem(r *registry) {
	addTool(r, "system", "higo.system.info",
		"Get HiGoOS system identity, version and runtime environment.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SystemInfo(ctx)
		})

	addTool(r, "system", "higo.system.updates.status",
		"Get the current system software update availability and status.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SystemUpdates(ctx)
		})

	addTool(r, "system", "higo.system.updates.check",
		"Trigger a check for available system software updates.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SystemUpdateCheck(ctx)
		})

	addTool(r, "system", "higo.system.backup.create",
		"Create a system configuration backup snapshot.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SystemBackup(ctx, map[string]any{})
		})
}
