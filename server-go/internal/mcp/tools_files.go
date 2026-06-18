package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// FilesTreeInput selects which space's file tree to return.
type FilesTreeInput struct {
	Space string `json:"space,omitempty" jsonschema:"Space whose file tree to return (empty for default)"`
}

// FilesSearchInput parameterizes a file search.
type FilesSearchInput struct {
	Q     string `json:"q,omitempty" jsonschema:"Free-text search query"`
	Space string `json:"space,omitempty" jsonschema:"Restrict results to this space"`
	Type  string `json:"type,omitempty" jsonschema:"Restrict results to this file type"`
	Tags  string `json:"tags,omitempty" jsonschema:"Comma-separated tags to filter by"`
	Limit int    `json:"limit,omitempty" jsonschema:"Maximum number of results to return"`
}

// FilesCreateFolderInput describes a new folder.
type FilesCreateFolderInput struct {
	Space string `json:"space" jsonschema:"Space to create the folder in"`
	Path  string `json:"path" jsonschema:"Parent path the folder is created under"`
	Name  string `json:"name" jsonschema:"Name of the new folder"`
	Actor string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// FilesUploadInput describes a file to create from inline content.
type FilesUploadInput struct {
	Space   string `json:"space" jsonschema:"Space to create the file in"`
	Path    string `json:"path" jsonschema:"Parent path the file is created under"`
	Name    string `json:"name" jsonschema:"Name of the new file"`
	Content string `json:"content,omitempty" jsonschema:"Inline text content of the file"`
	Actor   string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// FilesBatchInput describes a batch move/rename/delete operation.
type FilesBatchInput struct {
	Type        string            `json:"type,omitempty" jsonschema:"Batch operation type"`
	FileIDs     []string          `json:"fileIds" jsonschema:"IDs of the files to operate on"`
	Destination string            `json:"destination,omitempty" jsonschema:"Destination path for move operations"`
	Rename      map[string]string `json:"rename,omitempty" jsonschema:"Map of file id to new name for rename operations"`
	Actor       string            `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// FilesGetInput identifies a single file.
type FilesGetInput struct {
	ID string `json:"id" jsonschema:"ID of the file to fetch"`
}

// FilesAddTagsInput adds tags to a file.
type FilesAddTagsInput struct {
	ID     string   `json:"id" jsonschema:"ID of the file to tag"`
	Tags   []string `json:"tags" jsonschema:"Tags to add to the file"`
	Actor  string   `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
	Source string   `json:"source,omitempty" jsonschema:"Source that produced the tags"`
	Audit  string   `json:"audit,omitempty" jsonschema:"Audit note for the tag change"`
}

// FilesCreateShareInput creates a share link for a file.
type FilesCreateShareInput struct {
	ID            string `json:"id" jsonschema:"ID of the file to share"`
	Password      string `json:"password,omitempty" jsonschema:"Optional password protecting the share"`
	DownloadLimit int    `json:"downloadLimit,omitempty" jsonschema:"Maximum number of downloads allowed"`
	ExpiresInDays int    `json:"expiresInDays,omitempty" jsonschema:"Number of days until the share expires"`
	Actor         string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// FilesRenameInput renames a file.
type FilesRenameInput struct {
	ID    string `json:"id" jsonschema:"ID of the file to rename"`
	Name  string `json:"name" jsonschema:"New name for the file"`
	Actor string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// FilesMoveInput moves a file to a new destination.
type FilesMoveInput struct {
	ID          string `json:"id" jsonschema:"ID of the file to move"`
	Destination string `json:"destination" jsonschema:"Destination path to move the file to"`
	Actor       string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// FilesDeleteInput deletes a file.
type FilesDeleteInput struct {
	ID    string `json:"id" jsonschema:"ID of the file to delete"`
	Actor string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// FilesRestoreInput restores a deleted file.
type FilesRestoreInput struct {
	ID string `json:"id" jsonschema:"ID of the file to restore"`
}

// FilesPreviewURLInput identifies a file to preview.
type FilesPreviewURLInput struct {
	ID string `json:"id" jsonschema:"ID of the file to preview"`
}

// FilesDownloadURLInput identifies a file to download.
type FilesDownloadURLInput struct {
	ID string `json:"id" jsonschema:"ID of the file to download"`
}

func registerFiles(r *registry) {
	addTool(r, "files", "higo.files.tree",
		"List the file and folder tree for a space.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in FilesTreeInput) (json.RawMessage, error) {
			return c.FilesTree(ctx, in.Space)
		})

	addTool(r, "files", "higo.files.search",
		"Search files by query, space, type and tags.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in FilesSearchInput) (json.RawMessage, error) {
			return c.FilesSearch(ctx, in.Q, in.Space, in.Type, in.Tags, in.Limit)
		})

	addTool(r, "files", "higo.files.folder.create",
		"Create a new folder in a space.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesCreateFolderInput) (json.RawMessage, error) {
			return c.FilesCreateFolder(ctx, in)
		})

	addTool(r, "files", "higo.files.upload",
		"Create a file from inline JSON content.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesUploadInput) (json.RawMessage, error) {
			return c.FilesUpload(ctx, in)
		})

	addTool(r, "files", "higo.files.batch.move",
		"Move a batch of files to a destination.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesBatchInput) (json.RawMessage, error) {
			return c.FilesBatchMove(ctx, in)
		})

	addTool(r, "files", "higo.files.batch.rename",
		"Rename a batch of files.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesBatchInput) (json.RawMessage, error) {
			return c.FilesBatchRename(ctx, in)
		})

	addTool(r, "files", "higo.files.batch.delete",
		"Delete a batch of files.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in FilesBatchInput) (json.RawMessage, error) {
			return c.FilesBatchDelete(ctx, in)
		})

	addTool(r, "files", "higo.files.get",
		"Get metadata for a single file by id.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in FilesGetInput) (json.RawMessage, error) {
			return c.FilesGet(ctx, in.ID)
		})

	addTool(r, "files", "higo.files.preview.url",
		"Get the URL to fetch a file preview (binary/stream endpoint).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in FilesPreviewURLInput) (json.RawMessage, error) {
			return apiclient.URLResult("/api/v1/files/" + in.ID + "/preview"), nil
		})

	addTool(r, "files", "higo.files.download.url",
		"Get the URL to download a file (binary/stream endpoint).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in FilesDownloadURLInput) (json.RawMessage, error) {
			return apiclient.URLResult("/api/v1/files/" + in.ID + "/download"), nil
		})

	addTool(r, "files", "higo.files.tags.add",
		"Add tags to a file.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesAddTagsInput) (json.RawMessage, error) {
			return c.FilesAddTags(ctx, in.ID, in)
		})

	addTool(r, "files", "higo.files.share.create",
		"Create a share link for a file.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesCreateShareInput) (json.RawMessage, error) {
			return c.FilesCreateShare(ctx, in.ID, in)
		})

	addTool(r, "files", "higo.files.rename",
		"Rename a single file.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesRenameInput) (json.RawMessage, error) {
			return c.FilesRename(ctx, in.ID, in)
		})

	addTool(r, "files", "higo.files.move",
		"Move a single file to a destination.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesMoveInput) (json.RawMessage, error) {
			return c.FilesMove(ctx, in.ID, in)
		})

	addTool(r, "files", "higo.files.delete",
		"Delete a single file.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in FilesDeleteInput) (json.RawMessage, error) {
			return c.FilesDelete(ctx, in.ID, in)
		})

	addTool(r, "files", "higo.files.restore",
		"Restore a previously deleted file.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in FilesRestoreInput) (json.RawMessage, error) {
			return c.FilesRestore(ctx, in.ID)
		})
}
