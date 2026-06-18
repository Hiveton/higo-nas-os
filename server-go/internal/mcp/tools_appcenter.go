package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

type AppCenterAppInstallInput struct {
	ID string `json:"id" jsonschema:"app id to install"`
}

type AppCenterAppUpdateInput struct {
	ID string `json:"id" jsonschema:"app id to update"`
}

type AppCenterAppStartInput struct {
	ID string `json:"id" jsonschema:"app id to start"`
}

type AppCenterAppStopInput struct {
	ID string `json:"id" jsonschema:"app id to stop"`
}

func registerAppCenter(r *registry) {
	addTool(r, "app-center", "higo.app-center.apps.list",
		"List available and installed apps in the app center.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AppCenterApps(ctx)
		})

	addTool(r, "app-center", "higo.app-center.apps.install",
		"Install an app from the app center.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterAppInstallInput) (json.RawMessage, error) {
			return c.AppCenterAppInstall(ctx, in.ID)
		})

	addTool(r, "app-center", "higo.app-center.apps.update",
		"Update an installed app.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterAppUpdateInput) (json.RawMessage, error) {
			return c.AppCenterAppUpdate(ctx, in.ID)
		})

	addTool(r, "app-center", "higo.app-center.apps.start",
		"Start an installed app.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterAppStartInput) (json.RawMessage, error) {
			return c.AppCenterAppStart(ctx, in.ID)
		})

	addTool(r, "app-center", "higo.app-center.apps.stop",
		"Stop a running app.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterAppStopInput) (json.RawMessage, error) {
			return c.AppCenterAppStop(ctx, in.ID)
		})
}
