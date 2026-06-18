package mcp

import (
	"context"
	"encoding/json"
	"net/url"

	"higoos/server-go/internal/apiclient"
)

// --- input types ------------------------------------------------------------

type MusicLibraryUpdateInput struct {
	Paths    []string `json:"paths" jsonschema:"filesystem paths scanned for music"`
	AutoScan *bool    `json:"autoScan,omitempty" jsonschema:"whether libraries are rescanned automatically"`
}

type MusicTracksInput struct {
	Q string `json:"q,omitempty" jsonschema:"free-text search query"`
}

type MusicTrackStreamInput struct {
	ID string `json:"id" jsonschema:"identifier of the track to stream"`
}

type MusicTrackCoverInput struct {
	ID string `json:"id" jsonschema:"identifier of the track cover"`
}

type MusicTrackLyricsInput struct {
	ID string `json:"id" jsonschema:"identifier of the track lyrics"`
}

// --- registration -----------------------------------------------------------

func registerMusic(r *registry) {
	addTool(r, "music", "higo.music.library.get",
		"Get music library settings.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MusicLibrary(ctx)
		})

	addTool(r, "music", "higo.music.library.update",
		"Replace music library settings.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MusicLibraryUpdateInput) (json.RawMessage, error) {
			body := map[string]any{"paths": in.Paths}
			if in.AutoScan != nil {
				body["autoScan"] = *in.AutoScan
			}
			return c.MusicLibraryUpdate(ctx, body)
		})

	addTool(r, "music", "higo.music.scan",
		"Trigger a scan of the music library.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MusicScan(ctx)
		})

	addTool(r, "music", "higo.music.tracks.list",
		"List music tracks, optionally filtered by a search query.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in MusicTracksInput) (json.RawMessage, error) {
			return c.MusicTracks(ctx, in.Q)
		})

	addTool(r, "music", "higo.music.albums.list",
		"List music albums.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MusicAlbums(ctx)
		})

	addTool(r, "music", "higo.music.tracks.stream.url",
		"Get the direct stream URL for a music track.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in MusicTrackStreamInput) (json.RawMessage, error) {
			return apiclient.URLResult("/api/v1/music/tracks/" + url.PathEscape(in.ID) + "/stream"), nil
		})

	addTool(r, "music", "higo.music.tracks.cover.url",
		"Get the cover art URL for a music track.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in MusicTrackCoverInput) (json.RawMessage, error) {
			return apiclient.URLResult("/api/v1/music/tracks/" + url.PathEscape(in.ID) + "/cover"), nil
		})

	addTool(r, "music", "higo.music.tracks.lyrics.url",
		"Get the lyrics URL for a music track.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in MusicTrackLyricsInput) (json.RawMessage, error) {
			return apiclient.URLResult("/api/v1/music/tracks/" + url.PathEscape(in.ID) + "/lyrics"), nil
		})
}
