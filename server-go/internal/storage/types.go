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
	TaskKindSMARTScan   TaskKind = "smart-scan"
	TaskKindRepair      TaskKind = "repair"
	TaskKindSnapshot    TaskKind = "snapshot"
	TaskKindCreateSpace TaskKind = "create-space"
	TaskKindDeleteSpace TaskKind = "delete-space"
	TaskKindRemoveDisk  TaskKind = "remove-disk"
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
	MountPath   string `json:"mountPath,omitempty"`
}

type SpaceMode string

const (
	SpaceModeBasic  SpaceMode = "basic"
	SpaceModeLinear SpaceMode = "linear"
	SpaceModeRAID0  SpaceMode = "raid0"
	SpaceModeRAID1  SpaceMode = "raid1"
	SpaceModeRAID5  SpaceMode = "raid5"
	SpaceModeRAID6  SpaceMode = "raid6"
	SpaceModeRAID10 SpaceMode = "raid10"
)

type FileSystem string

const (
	FileSystemEXT4  FileSystem = "ext4"
	FileSystemBTRFS FileSystem = "btrfs"
	FileSystemZFS   FileSystem = "zfs"
)

type StorageSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// PoolID is the storage pool this space is created on (the first-level
	// physical grouping). A space always belongs to exactly one pool.
	PoolID      string     `json:"poolId"`
	Mode        SpaceMode  `json:"mode"`
	FileSystem  FileSystem `json:"fileSystem"`
	DiskSlots   []string   `json:"diskSlots"`
	MountPath   string     `json:"mountPath"`
	UsedPercent int        `json:"usedPercent"`
	Total       string     `json:"total"`
	Health      Health     `json:"health"`
	CreatedAt   time.Time  `json:"createdAt"`
	CreatedBy   string     `json:"createdBy,omitempty"`
}

type Disk struct {
	Slot           string          `json:"slot"`
	Size           string          `json:"size"`
	State          DiskState       `json:"state"`
	Temperature    string          `json:"temperature"`
	Serial         string          `json:"serial"`
	Health         Health          `json:"health"`
	Role           string          `json:"role"`
	PoolID         string          `json:"poolId"`
	Model          string          `json:"model"`
	Interface      string          `json:"interface"`
	DevicePath     string          `json:"devicePath,omitempty"`
	DeviceType     string          `json:"deviceType,omitempty"`
	MediaType      string          `json:"mediaType,omitempty"`
	Rotational     *bool           `json:"rotational,omitempty"`
	SystemDisk     bool            `json:"systemDisk,omitempty"`
	FileSystem     string          `json:"fileSystem,omitempty"`
	MountPath      string          `json:"mountPath,omitempty"`
	StandbyMinutes int             `json:"standbyMinutes,omitempty"`
	SSDCache       bool            `json:"ssdCache,omitempty"`
	CacheMode      string          `json:"cacheMode,omitempty"`
	Partitions     []DiskPartition `json:"partitions,omitempty"`
}

type DiskPartition struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Size       string `json:"size"`
	Used       string `json:"used,omitempty"`
	Total      string `json:"total,omitempty"`
	FileSystem string `json:"fileSystem,omitempty"`
	MountPath  string `json:"mountPath,omitempty"`
	System     bool   `json:"system,omitempty"`
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

type AddDiskRequest struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	Role      string `json:"role"`
	Actor     string `json:"actor"`
}

type CreateSpaceRequest struct {
	Name string `json:"name"`
	// PoolID selects the storage pool to create this space on (first-level
	// choice). When set, the disks are derived from the pool; DiskSlots stays
	// optional for advanced/explicit disk selection.
	PoolID     string     `json:"poolId"`
	Mode       SpaceMode  `json:"mode"`
	FileSystem FileSystem `json:"fileSystem"`
	DiskSlots  []string   `json:"diskSlots"`
	MountPath  string     `json:"mountPath"`
	Actor      string     `json:"actor"`
	Confirm    bool       `json:"confirm"`
	FormatDisk *bool      `json:"formatDisk,omitempty"`
}

type DeleteSpaceRequest struct {
	Actor   string `json:"actor"`
	Confirm bool   `json:"confirm"`
}

type RemoveDiskRequest struct {
	Actor   string `json:"actor"`
	Confirm bool   `json:"confirm"`
}

type DiskSettingsRequest struct {
	StandbyMinutes int    `json:"standbyMinutes"`
	SSDCache       bool   `json:"ssdCache"`
	CacheMode      string `json:"cacheMode"`
	Actor          string `json:"actor"`
}
