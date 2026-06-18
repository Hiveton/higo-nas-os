package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// MusicLibrary returns music library settings (GET /api/v1/music/library).
func (c *Client) MusicLibrary(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/music/library", nil, nil)
}

// MusicLibraryUpdate replaces music library settings (PUT /api/v1/music/library).
func (c *Client) MusicLibraryUpdate(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/music/library", nil, body)
}

// MusicScan triggers a music library scan (POST /api/v1/music/scan).
func (c *Client) MusicScan(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/music/scan", nil, nil)
}

// MusicTracks lists music tracks (GET /api/v1/music/tracks).
func (c *Client) MusicTracks(ctx context.Context, q string) (json.RawMessage, error) {
	var query url.Values
	if q != "" {
		query = url.Values{}
		query.Set("q", q)
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/music/tracks", query, nil)
}

// MusicAlbums lists music albums (GET /api/v1/music/albums).
func (c *Client) MusicAlbums(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/music/albums", nil, nil)
}
