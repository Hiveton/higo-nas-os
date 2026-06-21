package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// SharedFoldersList lists shared folders with their permission tables and
// service status (GET /api/v1/shared-folders).
func (c *Client) SharedFoldersList(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/shared-folders", nil, nil)
}

// SharedFolderCreate creates a shared folder (POST /api/v1/shared-folders).
func (c *Client) SharedFolderCreate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/shared-folders", nil, body)
}

// SharedFolderSetPermissions reconciles a folder's permission table
// (PUT /api/v1/shared-folders/{id}/permissions).
func (c *Client) SharedFolderSetPermissions(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/shared-folders/"+url.PathEscape(id)+"/permissions", nil, body)
}

// SharedFolderSetService toggles a folder's protocol export
// (PUT /api/v1/shared-folders/{id}/services).
func (c *Client) SharedFolderSetService(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/shared-folders/"+url.PathEscape(id)+"/services", nil, body)
}

// SharedFolderSetSubjectPermissions sets one subject's access across many
// folders (POST /api/v1/shared-folders/permissions/by-subject).
func (c *Client) SharedFolderSetSubjectPermissions(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/shared-folders/permissions/by-subject", nil, body)
}

// SharedFolderSetAdvanced updates a folder's advanced settings — recycle bin,
// quota, encryption intent (PUT /api/v1/shared-folders/{id}/advanced).
func (c *Client) SharedFolderSetAdvanced(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/shared-folders/"+url.PathEscape(id)+"/advanced", nil, body)
}

// SharedFolderSnapshots lists a folder's snapshots
// (GET /api/v1/shared-folders/{id}/snapshots).
func (c *Client) SharedFolderSnapshots(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/shared-folders/"+url.PathEscape(id)+"/snapshots", nil, nil)
}

// SharedFolderSnapshotCreate takes a new snapshot of a folder
// (POST /api/v1/shared-folders/{id}/snapshots).
func (c *Client) SharedFolderSnapshotCreate(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/shared-folders/"+url.PathEscape(id)+"/snapshots", nil, body)
}

// SharedFolderSnapshotDelete removes a folder snapshot by name
// (DELETE /api/v1/shared-folders/{id}/snapshots/{name}).
func (c *Client) SharedFolderSnapshotDelete(ctx context.Context, id, name string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/shared-folders/"+url.PathEscape(id)+"/snapshots/"+url.PathEscape(name), nil, nil)
}

// SharedFolderDeletePreview previews removing a shared folder
// (POST /api/v1/shared-folders/{id}/delete/preview).
func (c *Client) SharedFolderDeletePreview(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/shared-folders/"+url.PathEscape(id)+"/delete/preview", nil, body)
}

// SharedFolderDeleteConfirm removes a shared folder
// (POST /api/v1/shared-folders/{id}/delete/confirm).
func (c *Client) SharedFolderDeleteConfirm(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/shared-folders/"+url.PathEscape(id)+"/delete/confirm", nil, body)
}

// AccountsSambaSync reports which managed users have a Samba account
// (POST /api/v1/accounts/samba-sync).
func (c *Client) AccountsSambaSync(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/accounts/samba-sync", nil, nil)
}
