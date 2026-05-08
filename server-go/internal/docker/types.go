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
