package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// DesktopUpdateSessionInput patches the desktop session. Only the provided
// fields are applied; omitted fields are left unchanged.
type DesktopUpdateSessionInput struct {
	OpenWindowIDs      []string `json:"openWindowIds,omitempty" jsonschema:"IDs of the currently open windows"`
	MinimizedWindowIDs []string `json:"minimizedWindowIds,omitempty" jsonschema:"IDs of the minimized windows"`
	ActiveWindowID     *string  `json:"activeWindowId,omitempty" jsonschema:"ID of the active window"`
	UtilityAppID       *string  `json:"utilityAppId,omitempty" jsonschema:"ID of the active utility app"`
	AssistantVisible   *bool    `json:"assistantVisible,omitempty" jsonschema:"Whether the assistant panel is visible"`
	IsCompact          *bool    `json:"isCompact,omitempty" jsonschema:"Whether the desktop is in compact mode"`
	MaximizedWindowID  *string  `json:"maximizedWindowId,omitempty" jsonschema:"ID of the maximized window"`
	DockOrder          []string `json:"dockOrder,omitempty" jsonschema:"Ordered list of dock app IDs"`
	PinnedDockAppIDs   []string `json:"pinnedDockAppIds,omitempty" jsonschema:"IDs of apps pinned to the dock"`
}

func registerDesktop(r *registry) {
	addTool(r, "desktop", "higo.desktop.apps",
		"List the registered desktop apps.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DesktopApps(ctx)
		})

	addTool(r, "desktop", "higo.desktop.windows",
		"List the open desktop windows.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DesktopWindows(ctx)
		})

	addTool(r, "desktop", "higo.desktop.session.get",
		"Get the current desktop session state.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DesktopSession(ctx)
		})

	addTool(r, "desktop", "higo.desktop.session.update",
		"Update the desktop session state.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DesktopUpdateSessionInput) (json.RawMessage, error) {
			return c.DesktopUpdateSession(ctx, in)
		})
}
