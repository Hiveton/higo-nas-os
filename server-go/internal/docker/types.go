package docker

import "time"

type ContainerStatus string

const (
	ContainerStatusRunning    ContainerStatus = "运行中"
	ContainerStatusStopped    ContainerStatus = "已停止"
	ContainerStatusRestarting ContainerStatus = "重启中"
)

type ComposeStack struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Services int    `json:"services"`
	Ports    string `json:"ports"`
	Volume   string `json:"volume"`
	Network  string `json:"network"`
}

type ResourceLimit struct {
	CPU      int `json:"cpu"`
	MemoryMB int `json:"memoryMb"`
}

type Container struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	Image      string          `json:"image"`
	Stack      string          `json:"stack"`
	Status     ContainerStatus `json:"status"`
	CPU        int             `json:"cpu"`
	Memory     int             `json:"memory"`
	MemoryText string          `json:"memoryText"`
	Ports      []string        `json:"ports"`
	Mounts     []string        `json:"mounts"`
	Env        []string        `json:"env"`
	Limit      ResourceLimit   `json:"limit"`
	Restarts   int             `json:"restarts"`
	Isolation  string          `json:"isolation"`
}

type ContainerLog struct {
	ID          string    `json:"id"`
	ContainerID string    `json:"containerId"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
}

type Image struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	Created    string `json:"created"`
	IconURL    string `json:"iconUrl,omitempty"`
}

type ImageSearchResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Stars       int    `json:"stars"`
	Official    bool   `json:"official"`
	Automated   bool   `json:"automated"`
	IconURL     string `json:"iconUrl,omitempty"`
}

type ImagePullStatus struct {
	ID         string    `json:"id"`
	Image      string    `json:"image"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	Progress   int       `json:"progress"`
	Downloaded string    `json:"downloaded"`
	Total      string    `json:"total"`
	Speed      string    `json:"speed"`
	Error      string    `json:"error"`
	StartedAt  time.Time `json:"startedAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Volume struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Scope      string `json:"scope"`
	Mountpoint string `json:"mountpoint"`
}

type Network struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Driver     string   `json:"driver"`
	Scope      string   `json:"scope"`
	Subnet     string   `json:"subnet"`
	Gateway    string   `json:"gateway"`
	Internal   bool     `json:"internal"`
	Attachable bool     `json:"attachable"`
	Containers []string `json:"containers"`
}

type CreateContainerRequest struct {
	Image         string   `json:"image"`
	Name          string   `json:"name"`
	Ports         []string `json:"ports"`
	Mounts        []string `json:"mounts"`
	Env           []string `json:"env"`
	Labels        []string `json:"labels"`
	ExtraHosts    []string `json:"extraHosts"`
	DNS           []string `json:"dns"`
	Command       string   `json:"command"`
	Entrypoint    string   `json:"entrypoint"`
	RestartPolicy string   `json:"restartPolicy"`
	Network       string   `json:"network"`
	Hostname      string   `json:"hostname"`
	User          string   `json:"user"`
	WorkingDir    string   `json:"workingDir"`
	Privileged    bool     `json:"privileged"`
	AutoRemove    bool     `json:"autoRemove"`
	LimitCPU      int      `json:"limitCpu"`
	LimitMemory   int      `json:"limitMemory"`
}

type RemoveContainerRequest struct {
	Force         bool `json:"force"`
	RemoveVolumes bool `json:"removeVolumes"`
}

type ContainerExecRequest struct {
	Command string `json:"command"`
}

type ContainerExecResult struct {
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output"`
}

type PullImageRequest struct {
	Image string `json:"image"`
}

type RemoveImageRequest struct {
	Image string `json:"image"`
	Force bool   `json:"force"`
}

type CreateVolumeRequest struct {
	Name   string `json:"name"`
	Driver string `json:"driver"`
}

type RemoveVolumeRequest struct {
	Name  string `json:"name"`
	Force bool   `json:"force"`
}

type CreateNetworkRequest struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Subnet     string `json:"subnet"`
	Gateway    string `json:"gateway"`
	Attachable bool   `json:"attachable"`
	Internal   bool   `json:"internal"`
}

type RemoveNetworkRequest struct {
	Name string `json:"name"`
}

type NetworkConnectRequest struct {
	Network   string `json:"network"`
	Container string `json:"container"`
	Alias     string `json:"alias"`
}

type NetworkDisconnectRequest struct {
	Network   string `json:"network"`
	Container string `json:"container"`
	Force     bool   `json:"force"`
}
