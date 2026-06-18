package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// SettingsModelInput mirrors the model policy settings.
type SettingsModelInput struct {
	Mode         string `json:"mode,omitempty" jsonschema:"Model mode (e.g. local or cloud)"`
	Provider     string `json:"provider,omitempty" jsonschema:"Model provider identifier"`
	LocalModel   string `json:"localModel,omitempty" jsonschema:"Local model identifier"`
	CloudModel   string `json:"cloudModel,omitempty" jsonschema:"Cloud model identifier"`
	CloudEnabled bool   `json:"cloudEnabled,omitempty" jsonschema:"Whether cloud model usage is enabled"`
}

// SettingsPrivacyInput mirrors the privacy policy settings.
type SettingsPrivacyInput struct {
	SensitiveDataLocalOnly bool `json:"sensitiveDataLocalOnly,omitempty" jsonschema:"Keep sensitive data on the local model only"`
	AuditRetentionDays     int  `json:"auditRetentionDays,omitempty" jsonschema:"Number of days to retain audit records"`
}

// SettingsUIInput mirrors the UI policy settings.
type SettingsUIInput struct {
	Theme        string `json:"theme,omitempty" jsonschema:"UI theme name"`
	WindowRadius string `json:"windowRadius,omitempty" jsonschema:"Window corner radius preset"`
	DockPosition string `json:"dockPosition,omitempty" jsonschema:"Dock position on screen"`
	DockStyle    string `json:"dockStyle,omitempty" jsonschema:"Dock visual style"`
	DockIconSize string `json:"dockIconSize,omitempty" jsonschema:"Dock icon size preset"`
}

// SettingsUpdateInput replaces the full settings document.
type SettingsUpdateInput struct {
	Model   SettingsModelInput   `json:"model,omitempty" jsonschema:"Model policy settings"`
	Privacy SettingsPrivacyInput `json:"privacy,omitempty" jsonschema:"Privacy policy settings"`
	UI      SettingsUIInput      `json:"ui,omitempty" jsonschema:"UI policy settings"`
}

func registerSettings(r *registry) {
	addTool(r, "settings", "higo.settings.get",
		"Get the current HiGoOS settings.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SettingsGet(ctx)
		})

	addTool(r, "settings", "higo.settings.update",
		"Update the HiGoOS settings document.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SettingsUpdateInput) (json.RawMessage, error) {
			return c.SettingsUpdate(ctx, in)
		})

	addTool(r, "settings", "higo.settings.defaults.restore",
		"Restore the HiGoOS settings to their defaults.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SettingsRestoreDefaults(ctx)
		})
}
