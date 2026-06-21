package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// SharedFolderCreateInput creates a shared folder under a storage space.
type SharedFolderCreateInput struct {
	SpaceID string `json:"spaceId" jsonschema:"storage space id the folder lives under"`
	Name    string `json:"name" jsonschema:"display name of the shared folder"`
	RelPath string `json:"relPath,omitempty" jsonschema:"path relative to the space root; empty means the space root"`
	Actor   string `json:"actor,omitempty" jsonschema:"actor performing the action"`
}

// SharedFolderPermissionEntryInput is one row of a folder's permission table.
type SharedFolderPermissionEntryInput struct {
	SubjectType string `json:"subjectType" jsonschema:"user or group"`
	SubjectID   string `json:"subjectId" jsonschema:"account/group id"`
	Access      string `json:"access" jsonschema:"none, read, read_write, or deny (deny overrides group access)"`
}

// SharedFolderSubjectPermInput sets one subject's access on one folder.
type SharedFolderSubjectPermInput struct {
	FolderID string `json:"folderId" jsonschema:"shared folder id"`
	Access   string `json:"access" jsonschema:"none, read, read_write, or deny"`
}

// SharedFolderSetSubjectPermissionsInput sets one subject's access across folders.
type SharedFolderSetSubjectPermissionsInput struct {
	SubjectType string                         `json:"subjectType" jsonschema:"user or group"`
	SubjectID   string                         `json:"subjectId" jsonschema:"account/group id"`
	Perms       []SharedFolderSubjectPermInput `json:"perms" jsonschema:"per-folder access for this subject"`
	Actor       string                         `json:"actor,omitempty"`
}

// SharedFolderSetPermissionsInput reconciles a folder's permission table.
type SharedFolderSetPermissionsInput struct {
	ID      string                             `json:"id" jsonschema:"shared folder id"`
	Entries []SharedFolderPermissionEntryInput `json:"entries" jsonschema:"the full desired permission table"`
	Actor   string                             `json:"actor,omitempty" jsonschema:"actor performing the action"`
}

// SharedFolderSetServiceInput toggles a folder's protocol export.
type SharedFolderSetServiceInput struct {
	ID       string `json:"id" jsonschema:"shared folder id"`
	Protocol string `json:"protocol" jsonschema:"smb or nfs"`
	Enabled  bool   `json:"enabled" jsonschema:"whether the export is enabled"`
	Guest    bool   `json:"guest,omitempty" jsonschema:"allow guest access (SMB)"`
	Actor    string `json:"actor,omitempty" jsonschema:"actor performing the action"`
}

// SharedFolderSetAdvancedInput updates a folder's advanced settings.
type SharedFolderSetAdvancedInput struct {
	ID         string `json:"id" jsonschema:"shared folder id"`
	Recycle    *bool  `json:"recycle,omitempty" jsonschema:"enable the recycle bin (deleted SMB files go to #recycle)"`
	QuotaBytes *int64 `json:"quotaBytes,omitempty" jsonschema:"per-folder quota in bytes (Btrfs subvolume only; 0 = unlimited)"`
	Encrypted  *bool  `json:"encrypted,omitempty" jsonschema:"folder-level encryption intent"`
	Actor      string `json:"actor,omitempty" jsonschema:"actor performing the action"`
}

// SharedFolderSnapshotInput identifies a folder for snapshot operations.
type SharedFolderSnapshotInput struct {
	ID    string `json:"id" jsonschema:"shared folder id"`
	Name  string `json:"name,omitempty" jsonschema:"snapshot name (auto-generated from timestamp if empty)"`
	Actor string `json:"actor,omitempty" jsonschema:"actor performing the action"`
}

// SharedFolderSnapshotDeleteInput removes a folder snapshot by name.
type SharedFolderSnapshotDeleteInput struct {
	ID   string `json:"id" jsonschema:"shared folder id"`
	Name string `json:"name" jsonschema:"snapshot name to delete"`
}

// SharedFolderDeletePreviewInput previews removing a shared folder.
type SharedFolderDeletePreviewInput struct {
	ID    string `json:"id" jsonschema:"shared folder id"`
	Actor string `json:"actor,omitempty" jsonschema:"actor performing the action"`
}

// SharedFolderDeleteConfirmInput removes a shared folder.
type SharedFolderDeleteConfirmInput struct {
	ID             string `json:"id" jsonschema:"shared folder id"`
	ConfirmationID string `json:"confirmationId" jsonschema:"single-use confirmation id from the delete preview"`
	RemoveDir      bool   `json:"removeDir,omitempty" jsonschema:"also delete the directory and its data (irreversible)"`
	Actor          string `json:"actor,omitempty" jsonschema:"actor performing the action"`
}

func registerSharedFolders(r *registry) {
	addTool(r, "sharedfolders", "higo.sharedfolders.list",
		"List shared folders with their permission tables (user/group access) and SMB/NFS service status.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SharedFoldersList(ctx)
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.create",
		"Create a shared folder under a storage space (space root or sub-folder).",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderCreateInput) (json.RawMessage, error) {
			return c.SharedFolderCreate(ctx, in)
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.permissions.set",
		"Set a shared folder's full permission table; SMB valid users / write list and POSIX ACLs are auto-derived.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderSetPermissionsInput) (json.RawMessage, error) {
			return c.SharedFolderSetPermissions(ctx, in.ID, map[string]any{"entries": in.Entries, "actor": in.Actor})
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.permissions.set-subject",
		"Set one user/group's access (none/read/read_write/deny) across many shared folders at once — the user/group editor's folder-permission tab.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderSetSubjectPermissionsInput) (json.RawMessage, error) {
			return c.SharedFolderSetSubjectPermissions(ctx, map[string]any{
				"subjectType": in.SubjectType, "subjectId": in.SubjectID, "perms": in.Perms, "actor": in.Actor,
			})
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.service.set",
		"Enable or disable a shared folder's SMB/NFS export (auto-derived from its permission table).",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderSetServiceInput) (json.RawMessage, error) {
			return c.SharedFolderSetService(ctx, in.ID, map[string]any{"protocol": in.Protocol, "enabled": in.Enabled, "guest": in.Guest, "actor": in.Actor})
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.advanced.set",
		"Update a shared folder's advanced settings: recycle bin (SMB), per-folder quota (Btrfs), encryption intent. Btrfs-only features degrade gracefully on ext4.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderSetAdvancedInput) (json.RawMessage, error) {
			body := map[string]any{"actor": in.Actor}
			if in.Recycle != nil {
				body["recycle"] = *in.Recycle
			}
			if in.QuotaBytes != nil {
				body["quotaBytes"] = *in.QuotaBytes
			}
			if in.Encrypted != nil {
				body["encrypted"] = *in.Encrypted
			}
			return c.SharedFolderSetAdvanced(ctx, in.ID, body)
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.snapshots.list",
		"List a shared folder's point-in-time snapshots (Btrfs).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderDeletePreviewInput) (json.RawMessage, error) {
			return c.SharedFolderSnapshots(ctx, in.ID)
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.snapshots.create",
		"Take a read-only snapshot of a shared folder (requires a Btrfs subvolume).",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderSnapshotInput) (json.RawMessage, error) {
			return c.SharedFolderSnapshotCreate(ctx, in.ID, map[string]any{"name": in.Name, "actor": in.Actor})
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.snapshots.delete",
		"Delete a shared folder snapshot by name (Btrfs).",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderSnapshotDeleteInput) (json.RawMessage, error) {
			return c.SharedFolderSnapshotDelete(ctx, in.ID, in.Name)
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.delete.preview",
		"Preview removing a shared folder: returns a single-use confirmationId and impact summary.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderDeletePreviewInput) (json.RawMessage, error) {
			return c.SharedFolderDeletePreview(ctx, in.ID, map[string]any{"actor": in.Actor})
		})

	addTool(r, "sharedfolders", "higo.sharedfolders.delete.confirm",
		"Confirm removing a shared folder (clears grants + shares; optionally deletes the directory).",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in SharedFolderDeleteConfirmInput) (json.RawMessage, error) {
			return c.SharedFolderDeleteConfirm(ctx, in.ID, map[string]any{"confirmationId": in.ConfirmationID, "removeDir": in.RemoveDir, "actor": in.Actor})
		})
}
