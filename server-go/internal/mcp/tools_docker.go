package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// DockerContainersCreateInput is the body for creating a container.
type DockerContainersCreateInput struct {
	Image         string   `json:"image" jsonschema:"image reference to run, e.g. nginx:latest"`
	Name          string   `json:"name,omitempty" jsonschema:"container name"`
	Ports         []string `json:"ports,omitempty" jsonschema:"port mappings, e.g. 8080:80"`
	Mounts        []string `json:"mounts,omitempty" jsonschema:"volume/bind mounts"`
	Env           []string `json:"env,omitempty" jsonschema:"environment variables as KEY=VALUE"`
	Labels        []string `json:"labels,omitempty" jsonschema:"labels as key=value"`
	ExtraHosts    []string `json:"extraHosts,omitempty" jsonschema:"extra host entries"`
	DNS           []string `json:"dns,omitempty" jsonschema:"custom DNS servers"`
	Command       string   `json:"command,omitempty" jsonschema:"override container command"`
	Entrypoint    string   `json:"entrypoint,omitempty" jsonschema:"override container entrypoint"`
	RestartPolicy string   `json:"restartPolicy,omitempty" jsonschema:"restart policy, e.g. always"`
	Network       string   `json:"network,omitempty" jsonschema:"network to attach"`
	Hostname      string   `json:"hostname,omitempty" jsonschema:"container hostname"`
	User          string   `json:"user,omitempty" jsonschema:"user to run as"`
	WorkingDir    string   `json:"workingDir,omitempty" jsonschema:"working directory inside the container"`
	Privileged    bool     `json:"privileged,omitempty" jsonschema:"run in privileged mode"`
	AutoRemove    bool     `json:"autoRemove,omitempty" jsonschema:"remove the container when it exits"`
	LimitCPU      int      `json:"limitCpu,omitempty" jsonschema:"CPU limit"`
	LimitMemory   int      `json:"limitMemory,omitempty" jsonschema:"memory limit in MB"`
}

// DockerContainersRemoveInput targets a container removal.
type DockerContainersRemoveInput struct {
	ID            string `json:"id" jsonschema:"container id to remove"`
	Force         bool   `json:"force,omitempty" jsonschema:"force removal of a running container"`
	RemoveVolumes bool   `json:"removeVolumes,omitempty" jsonschema:"also remove anonymous volumes"`
}

// DockerContainersLogsInput targets container log retrieval.
type DockerContainersLogsInput struct {
	ID   string `json:"id" jsonschema:"container id to read logs from"`
	Tail string `json:"tail,omitempty" jsonschema:"number of trailing log lines to return"`
}

// DockerContainerIDInput targets a single container by id.
type DockerContainerIDInput struct {
	ID string `json:"id" jsonschema:"container id"`
}

// DockerContainersExecInput runs a command inside a container.
type DockerContainersExecInput struct {
	ID      string `json:"id" jsonschema:"container id to exec into"`
	Command string `json:"command" jsonschema:"command to execute inside the container"`
}

// DockerContainersLimitsInput updates a container's resource limits.
type DockerContainersLimitsInput struct {
	ID       string `json:"id" jsonschema:"container id to update"`
	CPU      int    `json:"cpu,omitempty" jsonschema:"CPU limit"`
	MemoryMB int    `json:"memoryMb,omitempty" jsonschema:"memory limit in MB"`
}

// DockerImagesSearchInput searches the registry.
type DockerImagesSearchInput struct {
	Q string `json:"q,omitempty" jsonschema:"search term for registry images"`
}

// DockerImagesPullInput starts an image pull.
type DockerImagesPullInput struct {
	Image string `json:"image" jsonschema:"image reference to pull, e.g. redis:7"`
}

// DockerImagesRemoveInput removes an image.
type DockerImagesRemoveInput struct {
	Image string `json:"image" jsonschema:"image reference or id to remove"`
	Force bool   `json:"force,omitempty" jsonschema:"force removal even if in use"`
}

// DockerVolumesCreateInput creates a volume.
type DockerVolumesCreateInput struct {
	Name   string `json:"name" jsonschema:"volume name"`
	Driver string `json:"driver,omitempty" jsonschema:"volume driver, defaults to local"`
}

// DockerVolumesRemoveInput removes a volume.
type DockerVolumesRemoveInput struct {
	Name  string `json:"name" jsonschema:"volume name to remove"`
	Force bool   `json:"force,omitempty" jsonschema:"force removal even if in use"`
}

// DockerNetworksCreateInput creates a network.
type DockerNetworksCreateInput struct {
	Name       string `json:"name" jsonschema:"network name"`
	Driver     string `json:"driver,omitempty" jsonschema:"network driver, e.g. bridge"`
	Subnet     string `json:"subnet,omitempty" jsonschema:"subnet in CIDR notation"`
	Gateway    string `json:"gateway,omitempty" jsonschema:"gateway address"`
	Attachable bool   `json:"attachable,omitempty" jsonschema:"allow manual container attachment"`
	Internal   bool   `json:"internal,omitempty" jsonschema:"restrict external access"`
}

// DockerNetworksRemoveInput removes a network.
type DockerNetworksRemoveInput struct {
	Name string `json:"name" jsonschema:"network name to remove"`
}

// DockerNetworksConnectInput attaches a container to a network.
type DockerNetworksConnectInput struct {
	Network   string `json:"network" jsonschema:"network to connect to"`
	Container string `json:"container" jsonschema:"container to attach"`
	Alias     string `json:"alias,omitempty" jsonschema:"network alias for the container"`
}

// DockerNetworksDisconnectInput detaches a container from a network.
type DockerNetworksDisconnectInput struct {
	Network   string `json:"network" jsonschema:"network to disconnect from"`
	Container string `json:"container" jsonschema:"container to detach"`
	Force     bool   `json:"force,omitempty" jsonschema:"force disconnection"`
}

func registerDocker(r *registry) {
	addTool(r, "docker", "higo.docker.stacks.list",
		"List Docker compose stacks.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DockerStacks(ctx)
		})

	addTool(r, "docker", "higo.docker.containers.list",
		"List Docker containers.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DockerContainers(ctx)
		})

	addTool(r, "docker", "higo.docker.containers.create",
		"Create and start a new Docker container.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainersCreateInput) (json.RawMessage, error) {
			return c.DockerCreateContainer(ctx, in)
		})

	addTool(r, "docker", "higo.docker.containers.remove",
		"Remove a Docker container.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainersRemoveInput) (json.RawMessage, error) {
			return c.DockerRemoveContainer(ctx, in.ID, map[string]any{
				"force":         in.Force,
				"removeVolumes": in.RemoveVolumes,
			})
		})

	addTool(r, "docker", "higo.docker.containers.logs",
		"Read recent log lines from a Docker container.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainersLogsInput) (json.RawMessage, error) {
			return c.DockerContainerLogs(ctx, in.ID, in.Tail)
		})

	addTool(r, "docker", "higo.docker.containers.start",
		"Start a stopped Docker container.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainerIDInput) (json.RawMessage, error) {
			return c.DockerStartContainer(ctx, in.ID)
		})

	addTool(r, "docker", "higo.docker.containers.stop",
		"Stop a running Docker container.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainerIDInput) (json.RawMessage, error) {
			return c.DockerStopContainer(ctx, in.ID)
		})

	addTool(r, "docker", "higo.docker.containers.restart",
		"Restart a Docker container.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainerIDInput) (json.RawMessage, error) {
			return c.DockerRestartContainer(ctx, in.ID)
		})

	addTool(r, "docker", "higo.docker.containers.complete_restart",
		"Recreate and restart a Docker container with its original configuration.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainerIDInput) (json.RawMessage, error) {
			return c.DockerCompleteRestartContainer(ctx, in.ID)
		})

	addTool(r, "docker", "higo.docker.containers.exec",
		"Execute a command inside a running Docker container.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainersExecInput) (json.RawMessage, error) {
			return c.DockerExecContainer(ctx, in.ID, map[string]any{"command": in.Command})
		})

	addTool(r, "docker", "higo.docker.containers.limits",
		"Update CPU and memory limits for a Docker container.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerContainersLimitsInput) (json.RawMessage, error) {
			return c.DockerUpdateContainerLimits(ctx, in.ID, map[string]any{
				"cpu":      in.CPU,
				"memoryMb": in.MemoryMB,
			})
		})

	addTool(r, "docker", "higo.docker.images.list",
		"List Docker images.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DockerImages(ctx)
		})

	addTool(r, "docker", "higo.docker.images.search",
		"Search the registry for Docker images.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in DockerImagesSearchInput) (json.RawMessage, error) {
			return c.DockerSearchImages(ctx, in.Q)
		})

	addTool(r, "docker", "higo.docker.images.pull",
		"Start pulling a Docker image from a registry.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerImagesPullInput) (json.RawMessage, error) {
			return c.DockerPullImage(ctx, map[string]any{"image": in.Image})
		})

	addTool(r, "docker", "higo.docker.images.pulls",
		"List in-progress Docker image pull tasks.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DockerImagePulls(ctx)
		})

	addTool(r, "docker", "higo.docker.images.remove",
		"Remove a Docker image.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in DockerImagesRemoveInput) (json.RawMessage, error) {
			return c.DockerRemoveImage(ctx, map[string]any{
				"image": in.Image,
				"force": in.Force,
			})
		})

	addTool(r, "docker", "higo.docker.volumes.list",
		"List Docker volumes.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DockerVolumes(ctx)
		})

	addTool(r, "docker", "higo.docker.volumes.create",
		"Create a Docker volume.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerVolumesCreateInput) (json.RawMessage, error) {
			return c.DockerCreateVolume(ctx, in)
		})

	addTool(r, "docker", "higo.docker.volumes.remove",
		"Remove a Docker volume.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in DockerVolumesRemoveInput) (json.RawMessage, error) {
			return c.DockerRemoveVolume(ctx, map[string]any{
				"name":  in.Name,
				"force": in.Force,
			})
		})

	addTool(r, "docker", "higo.docker.networks.list",
		"List Docker networks.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.DockerNetworks(ctx)
		})

	addTool(r, "docker", "higo.docker.networks.create",
		"Create a Docker network.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerNetworksCreateInput) (json.RawMessage, error) {
			return c.DockerCreateNetwork(ctx, in)
		})

	addTool(r, "docker", "higo.docker.networks.remove",
		"Remove a Docker network.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in DockerNetworksRemoveInput) (json.RawMessage, error) {
			return c.DockerRemoveNetwork(ctx, map[string]any{"name": in.Name})
		})

	addTool(r, "docker", "higo.docker.networks.connect",
		"Connect a Docker container to a network.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerNetworksConnectInput) (json.RawMessage, error) {
			return c.DockerConnectNetwork(ctx, in)
		})

	addTool(r, "docker", "higo.docker.networks.disconnect",
		"Disconnect a Docker container from a network.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in DockerNetworksDisconnectInput) (json.RawMessage, error) {
			return c.DockerDisconnectNetwork(ctx, in)
		})
}
