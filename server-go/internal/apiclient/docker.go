package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// DockerStacks lists compose stacks (GET /api/v1/docker/stacks).
func (c *Client) DockerStacks(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/docker/stacks", nil, nil)
}

// DockerContainers lists containers (GET /api/v1/docker/containers).
func (c *Client) DockerContainers(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/docker/containers", nil, nil)
}

// DockerCreateContainer creates a container (POST /api/v1/docker/containers).
func (c *Client) DockerCreateContainer(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/containers", nil, body)
}

// DockerRemoveContainer removes a container (DELETE /api/v1/docker/containers/{id}).
func (c *Client) DockerRemoveContainer(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodDelete, "/api/v1/docker/containers/"+url.PathEscape(id), nil, body)
}

// DockerContainerLogs returns container logs (GET /api/v1/docker/containers/{id}/logs).
func (c *Client) DockerContainerLogs(ctx context.Context, id string, tail string) (json.RawMessage, error) {
	var query url.Values
	if tail != "" {
		query = url.Values{}
		query.Set("tail", tail)
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/docker/containers/"+url.PathEscape(id)+"/logs", query, nil)
}

// DockerStartContainer starts a container (POST /api/v1/docker/containers/{id}/start).
func (c *Client) DockerStartContainer(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/containers/"+url.PathEscape(id)+"/start", nil, nil)
}

// DockerStopContainer stops a container (POST /api/v1/docker/containers/{id}/stop).
func (c *Client) DockerStopContainer(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/containers/"+url.PathEscape(id)+"/stop", nil, nil)
}

// DockerRestartContainer restarts a container (POST /api/v1/docker/containers/{id}/restart).
func (c *Client) DockerRestartContainer(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/containers/"+url.PathEscape(id)+"/restart", nil, nil)
}

// DockerCompleteRestartContainer recreates and restarts a container
// (POST /api/v1/docker/containers/{id}/complete-restart).
func (c *Client) DockerCompleteRestartContainer(ctx context.Context, id string) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/containers/"+url.PathEscape(id)+"/complete-restart", nil, nil)
}

// DockerExecContainer runs a command inside a container (POST /api/v1/docker/containers/{id}/exec).
func (c *Client) DockerExecContainer(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/containers/"+url.PathEscape(id)+"/exec", nil, body)
}

// DockerUpdateContainerLimits updates a container's CPU/memory limits
// (PUT /api/v1/docker/containers/{id}/limits).
func (c *Client) DockerUpdateContainerLimits(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPut, "/api/v1/docker/containers/"+url.PathEscape(id)+"/limits", nil, body)
}

// DockerImages lists images (GET /api/v1/docker/images).
func (c *Client) DockerImages(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/docker/images", nil, nil)
}

// DockerSearchImages searches the registry for images (GET /api/v1/docker/images/search).
func (c *Client) DockerSearchImages(ctx context.Context, q string) (json.RawMessage, error) {
	var query url.Values
	if q != "" {
		query = url.Values{}
		query.Set("q", q)
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/docker/images/search", query, nil)
}

// DockerPullImage starts an image pull (POST /api/v1/docker/images/pull).
func (c *Client) DockerPullImage(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/images/pull", nil, body)
}

// DockerImagePulls lists in-progress image pull tasks (GET /api/v1/docker/images/pulls).
func (c *Client) DockerImagePulls(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/docker/images/pulls", nil, nil)
}

// DockerRemoveImage removes an image (POST /api/v1/docker/images/remove).
func (c *Client) DockerRemoveImage(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/images/remove", nil, body)
}

// DockerVolumes lists volumes (GET /api/v1/docker/volumes).
func (c *Client) DockerVolumes(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/docker/volumes", nil, nil)
}

// DockerCreateVolume creates a volume (POST /api/v1/docker/volumes).
func (c *Client) DockerCreateVolume(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/volumes", nil, body)
}

// DockerRemoveVolume removes a volume (POST /api/v1/docker/volumes/remove).
func (c *Client) DockerRemoveVolume(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/volumes/remove", nil, body)
}

// DockerNetworks lists networks (GET /api/v1/docker/networks).
func (c *Client) DockerNetworks(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/docker/networks", nil, nil)
}

// DockerCreateNetwork creates a network (POST /api/v1/docker/networks).
func (c *Client) DockerCreateNetwork(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/networks", nil, body)
}

// DockerRemoveNetwork removes a network (POST /api/v1/docker/networks/remove).
func (c *Client) DockerRemoveNetwork(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/networks/remove", nil, body)
}

// DockerConnectNetwork attaches a container to a network (POST /api/v1/docker/networks/connect).
func (c *Client) DockerConnectNetwork(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/networks/connect", nil, body)
}

// DockerDisconnectNetwork detaches a container from a network (POST /api/v1/docker/networks/disconnect).
func (c *Client) DockerDisconnectNetwork(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/docker/networks/disconnect", nil, body)
}
