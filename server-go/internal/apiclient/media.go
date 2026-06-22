package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// MediaItems lists media library items, optionally filtered by dimension/facet
// (GET /api/v1/media/items).
func (c *Client) MediaItems(ctx context.Context, dimension, facet string) (json.RawMessage, error) {
	var query url.Values
	if dimension != "" || facet != "" {
		query = url.Values{}
		if dimension != "" {
			query.Set("dimension", dimension)
		}
		if facet != "" {
			query.Set("facet", facet)
		}
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/media/items", query, nil)
}

// MediaScan rescans the media library (POST /api/v1/media/scan).
func (c *Client) MediaScan(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/media/scan", nil, nil)
}

// MediaAlbums lists media albums (GET /api/v1/media/albums).
func (c *Client) MediaAlbums(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/media/albums", nil, nil)
}

// MediaCreateAlbum creates a media album (POST /api/v1/media/albums).
func (c *Client) MediaCreateAlbum(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/media/albums", nil, body)
}

// MediaCreateMemory generates a smart memory album (POST /api/v1/media/memories).
func (c *Client) MediaCreateMemory(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/media/memories", nil, body)
}

// MediaMergePeople merges face/person clusters (POST /api/v1/media/people/merge).
func (c *Client) MediaMergePeople(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/media/people/merge", nil, body)
}

// MediaSubtitleJob queues a subtitle generation job (POST /api/v1/media/subtitles/jobs).
func (c *Client) MediaSubtitleJob(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/media/subtitles/jobs", nil, body)
}

// MediaTranscodeJob queues a transcode job (POST /api/v1/media/transcode/jobs).
func (c *Client) MediaTranscodeJob(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/media/transcode/jobs", nil, body)
}

// MediaCreateShare creates an album share link (POST /api/v1/media/shares).
func (c *Client) MediaCreateShare(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/media/shares", nil, body)
}
