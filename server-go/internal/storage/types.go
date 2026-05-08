package storage

import "time"

type Health string

const (
	HealthHealthy  Health = "健康"
	HealthSyncing  Health = "同步中"
	HealthWarning  Health = "警告"
	HealthCritical Health = "故障"
)

type DiskState string

const (
	DiskStateHealthy  DiskState = "健康"
	DiskStateHotSpare DiskState = "热备"
	DiskStateRebuild  DiskState = "重建中"
	DiskStateOffline  DiskState = "离线"
)

type TaskKind string

const (
	TaskKindSMARTScan TaskKind = "smart-scan"
	TaskKindRepair    TaskKind = "repair"
	TaskKindSnapshot  TaskKind = "snapshot"
)

type TaskState string

const (
	TaskStateQueued    TaskState = "queued"
	TaskStateRunning   TaskState = "running"
	TaskStateCompleted TaskState = "completed"
	TaskStateFailed    TaskState = "failed"
)

type StoragePool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	UsedPercent int    `json:"usedPercent"`
	Total       string `json:"total"`
	Health      Health `json:"health"`
	Temperature string `json:"temperature"`
}

type Disk struct {
	Slot        string    `json:"slot"`
	Size        string    `json:"size"`
	State       DiskState `json:"state"`
	Temperature string    `json:"temperature"`
	Serial      string    `json:"serial"`
	Health      Health    `json:"health"`
	Role        string    `json:"role"`
	PoolID      string    `json:"poolId"`
	Model       string    `json:"model"`
	Interface   string    `json:"interface"`
}

type SmartAttribute struct {
	Name      string `json:"name"`
	Value     int    `json:"value"`
	Threshold int    `json:"threshold"`
	Status    string `json:"status"`
}

type SmartReport struct {
	DiskSlot    string           `json:"diskSlot"`
	Serial      string           `json:"serial"`
	Health      Health           `json:"health"`
	Temperature string           `json:"temperature"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	Attributes  []SmartAttribute `json:"attributes"`
}

type StorageTask struct {
	ID         string    `json:"id"`
	Kind       TaskKind  `json:"kind"`
	State      TaskState `json:"state"`
	Progress   int       `json:"progress"`
	Message    string    `json:"message"`
	TargetSlot string    `json:"targetSlot,omitempty"`
	TargetPool string    `json:"targetPool,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type TaskTarget struct {
	TargetSlot string `json:"targetSlot,omitempty"`
	TargetPool string `json:"targetPool,omitempty"`
}
