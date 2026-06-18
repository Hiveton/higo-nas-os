package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// MediaItemsInput filters the media library listing.
type MediaItemsInput struct {
	Dimension string `json:"dimension,omitempty" jsonschema:"dimension to group items by (timeline, people, places, devices, albums)"`
	Facet     string `json:"facet,omitempty" jsonschema:"value within the dimension to filter to"`
}

// MediaCreateAlbumInput is the body for creating an album.
type MediaCreateAlbumInput struct {
	Name    string `json:"name" jsonschema:"album display name"`
	Type    string `json:"type,omitempty" jsonschema:"album type (家庭相册, 共享相册, 智能回忆)"`
	ItemIDs []int  `json:"itemIds,omitempty" jsonschema:"media item ids to include in the album"`
	Privacy string `json:"privacy,omitempty" jsonschema:"album privacy setting"`
}

// MediaCreateMemoryInput is the body for generating a smart memory.
type MediaCreateMemoryInput struct {
	Dimension string `json:"dimension" jsonschema:"dimension to build the memory from (timeline, people, places, devices, albums)"`
	Facet     string `json:"facet,omitempty" jsonschema:"value within the dimension to focus the memory on"`
}

// MediaMergePeopleInput is the body for merging person clusters.
type MediaMergePeopleInput struct {
	SourceNames []string `json:"sourceNames" jsonschema:"person names to merge into the target"`
	TargetName  string   `json:"targetName" jsonschema:"person name to keep as the merged identity"`
}

// MediaSubtitleJobInput is the body for queuing a subtitle job.
type MediaSubtitleJobInput struct {
	ItemID  int    `json:"itemId" jsonschema:"media item id to generate subtitles for"`
	Profile string `json:"profile,omitempty" jsonschema:"subtitle generation profile"`
}

// MediaTranscodeJobInput is the body for queuing a transcode job.
type MediaTranscodeJobInput struct {
	ItemID  int    `json:"itemId" jsonschema:"media item id to transcode"`
	Profile string `json:"profile,omitempty" jsonschema:"transcode output profile"`
}

// MediaCreateShareInput is the body for creating an album share link.
type MediaCreateShareInput struct {
	AlbumID       int `json:"albumId" jsonschema:"album id to share"`
	ExpiresInDays int `json:"expiresInDays,omitempty" jsonschema:"number of days the share link stays valid"`
}

func registerMedia(r *registry) {
	addTool(r, "media", "higo.media.items.list",
		"List media library items, optionally grouped by dimension and facet.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in MediaItemsInput) (json.RawMessage, error) {
			return c.MediaItems(ctx, in.Dimension, in.Facet)
		})

	addTool(r, "media", "higo.media.albums.list",
		"List media albums.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MediaAlbums(ctx)
		})

	addTool(r, "media", "higo.media.albums.create",
		"Create a media album from selected items.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MediaCreateAlbumInput) (json.RawMessage, error) {
			return c.MediaCreateAlbum(ctx, in)
		})

	addTool(r, "media", "higo.media.memories.create",
		"Generate a smart memory album for a dimension/facet.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MediaCreateMemoryInput) (json.RawMessage, error) {
			return c.MediaCreateMemory(ctx, in)
		})

	addTool(r, "media", "higo.media.people.merge",
		"Merge one or more person clusters into a target identity.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MediaMergePeopleInput) (json.RawMessage, error) {
			return c.MediaMergePeople(ctx, in)
		})

	addTool(r, "media", "higo.media.subtitles.jobs.create",
		"Queue a subtitle generation job for a media item.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MediaSubtitleJobInput) (json.RawMessage, error) {
			return c.MediaSubtitleJob(ctx, in)
		})

	addTool(r, "media", "higo.media.transcode.jobs.create",
		"Queue a transcode job for a media item.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MediaTranscodeJobInput) (json.RawMessage, error) {
			return c.MediaTranscodeJob(ctx, in)
		})

	addTool(r, "media", "higo.media.shares.create",
		"Create a share link for a media album.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MediaCreateShareInput) (json.RawMessage, error) {
			return c.MediaCreateShare(ctx, in)
		})
}
