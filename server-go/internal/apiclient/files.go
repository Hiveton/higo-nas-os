package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
)

// FilesTree returns the file/folder tree for a space (GET /api/v1/files/tree).
func (c *Client) FilesTree(ctx context.Context, space string) (json.RawMessage, error) {
	query := url.Values{}
	if space != "" {
		query.Set("space", space)
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/files/tree", query, nil)
}

// FilesSearch searches files by query, space, type, tags and limit
// (GET /api/v1/files/search).
func (c *Client) FilesSearch(ctx context.Context, q, space, fileType, tags string, limit int) (json.RawMessage, error) {
	query := url.Values{}
	if q != "" {
		query.Set("q", q)
	}
	if space != "" {
		query.Set("space", space)
	}
	if fileType != "" {
		query.Set("type", fileType)
	}
	if tags != "" {
		query.Set("tags", tags)
	}
	if limit != 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/files/search", query, nil)
}

// FilesCreateFolder creates a folder (POST /api/v1/files/folders).
func (c *Client) FilesCreateFolder(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/folders", nil, body)
}

// FilesUpload creates a file from an inline JSON body (POST /api/v1/files/upload).
func (c *Client) FilesUpload(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/upload", nil, body)
}

// FilesBatchMove moves a batch of files (POST /api/v1/files/batch/move).
func (c *Client) FilesBatchMove(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/batch/move", nil, body)
}

// FilesBatchRename renames a batch of files (POST /api/v1/files/batch/rename).
func (c *Client) FilesBatchRename(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/batch/rename", nil, body)
}

// FilesBatchDelete deletes a batch of files (POST /api/v1/files/batch/delete).
func (c *Client) FilesBatchDelete(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/batch/delete", nil, body)
}

// FilesGet returns a single file by id (GET /api/v1/files/{id}).
func (c *Client) FilesGet(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/files/"+url.PathEscape(id), nil, nil)
}

// FilesAddTags adds tags to a file (POST /api/v1/files/{id}/tags).
func (c *Client) FilesAddTags(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/"+url.PathEscape(id)+"/tags", nil, body)
}

// FilesCreateShare creates a share link for a file (POST /api/v1/files/{id}/shares).
func (c *Client) FilesCreateShare(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/"+url.PathEscape(id)+"/shares", nil, body)
}

// FilesRename renames a file (POST /api/v1/files/{id}/rename).
func (c *Client) FilesRename(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/"+url.PathEscape(id)+"/rename", nil, body)
}

// FilesMove moves a file to a destination (POST /api/v1/files/{id}/move).
func (c *Client) FilesMove(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/"+url.PathEscape(id)+"/move", nil, body)
}

// FilesDelete deletes a file (POST /api/v1/files/{id}/delete).
func (c *Client) FilesDelete(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/"+url.PathEscape(id)+"/delete", nil, body)
}

// FilesRestore restores a deleted file (POST /api/v1/files/{id}/restore).
func (c *Client) FilesRestore(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/files/"+url.PathEscape(id)+"/restore", nil, nil)
}
