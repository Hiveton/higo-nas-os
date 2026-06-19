package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

type AppCenterCatalogGetInput struct {
	ID string `json:"id" jsonschema:"catalog app id"`
}

type AppCenterPreviewInput struct {
	ID     string            `json:"id" jsonschema:"app id"`
	Action string            `json:"action" jsonschema:"lifecycle action: install|update|start|stop|uninstall"`
	Config map[string]string `json:"config,omitempty" jsonschema:"optional install config values"`
}

type AppCenterConfirmInput struct {
	ID             string            `json:"id" jsonschema:"app id"`
	Action         string            `json:"action" jsonschema:"lifecycle action: install|update|start|stop|uninstall"`
	ConfirmationID string            `json:"confirmationId" jsonschema:"confirmationId returned by the matching preview"`
	Actor          string            `json:"actor,omitempty" jsonschema:"actor performing the action"`
	Config         map[string]string `json:"config,omitempty" jsonschema:"optional install config values"`
}

type AppCenterRollbackInput struct {
	AuditID string `json:"auditId" jsonschema:"app-center audit id to reverse"`
	Actor   string `json:"actor,omitempty" jsonschema:"actor performing the rollback"`
}

type AppCenterRegistryAddInput struct {
	Name string `json:"name" jsonschema:"registry name"`
	URL  string `json:"url" jsonschema:"registry index url"`
}

type AppCenterRegistryRemoveInput struct {
	Name string `json:"name" jsonschema:"registry name to remove"`
}

func registerAppCenter(r *registry) {
	addTool(r, "app-center", "higo.app-center.apps.list",
		"List installed and installable apps with their runtime state.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AppCenterApps(ctx)
		})

	addTool(r, "app-center", "higo.app-center.catalog.list",
		"List every installable app manifest across builtin, local and remote sources.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AppCenterCatalog(ctx)
		})

	addTool(r, "app-center", "higo.app-center.catalog.get",
		"Fetch one catalog entry including its full manifest.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterCatalogGetInput) (json.RawMessage, error) {
			return c.AppCenterCatalogItem(ctx, in.ID)
		})

	addTool(r, "app-center", "higo.app-center.catalog.refresh",
		"Re-fetch the configured remote registries.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AppCenterCatalogRefresh(ctx)
		})

	addTool(r, "app-center", "higo.app-center.apps.preview",
		"Preview a governed lifecycle action (install|update|start|stop|uninstall) and get its confirmationId + impact summary. No side effect.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterPreviewInput) (json.RawMessage, error) {
			return c.AppCenterPreview(ctx, in.ID, in.Action, in.Config)
		})

	addTool(r, "app-center", "higo.app-center.apps.confirm",
		"Execute a previously-previewed lifecycle action by echoing its confirmationId. Uninstall is destructive.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterConfirmInput) (json.RawMessage, error) {
			return c.AppCenterConfirm(ctx, in.ID, in.Action, in.ConfirmationID, in.Actor, in.Config)
		})

	addTool(r, "app-center", "higo.app-center.audit.list",
		"List the app-center audit log of confirmed lifecycle actions.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AppCenterAudit(ctx)
		})

	addTool(r, "app-center", "higo.app-center.audit.rollback",
		"Reverse a previously-confirmed app-center action.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterRollbackInput) (json.RawMessage, error) {
			return c.AppCenterRollback(ctx, in.AuditID, in.Actor)
		})

	addTool(r, "app-center", "higo.app-center.registries.list",
		"List configured remote app registries.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AppCenterRegistries(ctx)
		})

	addTool(r, "app-center", "higo.app-center.registries.add",
		"Add (or re-enable) a remote app registry and refresh its index.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterRegistryAddInput) (json.RawMessage, error) {
			return c.AppCenterAddRegistry(ctx, in.Name, in.URL)
		})

	addTool(r, "app-center", "higo.app-center.registries.remove",
		"Remove a remote app registry.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in AppCenterRegistryRemoveInput) (json.RawMessage, error) {
			return c.AppCenterRemoveRegistry(ctx, in.Name)
		})
}
