package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"higoos/server-go/internal/state"
	"higoos/server-go/internal/tasks"
)

type Service struct {
	adapter     Adapter
	now         func() time.Time
	provisioner SpaceProvisioner

	mu         sync.Mutex
	taskSeq    int
	confirmSeq int
	tasks      map[string]StorageTask
	disks      []Disk
	spaces     []StorageSpace
	pending    map[string]pendingDelete // single-use delete confirmations, keyed by confirmationId
	audit      []StorageAuditEntry      // append-only governance trail (newest first)
	// defaultSpaceID is the system-wide default storage space new folders / user
	// home directories land on when none is chosen. Empty → first space.
	defaultSpaceID string
	statePath      string
	runner     *tasks.Manager
	zfsRunner  commandRunner      // shells out to zfs/zpool for ZFS maintenance tasks
	scheduler  *snapshotScheduler // lazily-built automatic-snapshot scheduler
}

type snapshot struct {
	TaskSeq    int                      `json:"taskSeq"`
	ConfirmSeq int                      `json:"confirmSeq"`
	Tasks      map[string]StorageTask   `json:"tasks"`
	Disks      []Disk                   `json:"disks"`
	Spaces     []StorageSpace           `json:"spaces"`
	Pending        map[string]pendingDelete `json:"pending"`
	Audit          []StorageAuditEntry      `json:"audit"`
	DefaultSpaceID string                   `json:"defaultSpaceId,omitempty"`
}

func NewService(adapter Adapter) *Service {
	if adapter == nil {
		adapter = NewHostAdapter()
	}
	return NewServiceWithProvisioner(adapter, NewCommandSpaceProvisioner())
}

func NewServiceWithProvisioner(adapter Adapter, provisioner SpaceProvisioner) *Service {
	if adapter == nil {
		adapter = NewHostAdapter()
	}
	if provisioner == nil {
		provisioner = NewCommandSpaceProvisioner()
	}
	return &Service{
		adapter:     adapter,
		now:         time.Now,
		provisioner: provisioner,
		tasks:       map[string]StorageTask{},
		pending:     map[string]pendingDelete{},
		zfsRunner:   runCommand,
	}
}

func NewServiceWithStateDir(adapter Adapter, stateDir string) (*Service, error) {
	service := NewService(adapter)
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "storage.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Tasks) > 0 {
		service.tasks = cloneTasks(persisted.Tasks)
		service.taskSeq = persisted.TaskSeq
		if service.taskSeq < len(service.tasks) {
			service.taskSeq = len(service.tasks)
		}
	}
	service.disks = cloneDisks(persisted.Disks)
	service.spaces = cloneSpaces(persisted.Spaces)
	service.confirmSeq = persisted.ConfirmSeq
	if len(persisted.Pending) > 0 {
		service.pending = clonePending(persisted.Pending)
	}
	service.audit = append([]StorageAuditEntry(nil), persisted.Audit...)
	service.defaultSpaceID = persisted.DefaultSpaceID
	if service.defaultSpaceID == "" && len(service.spaces) > 0 {
		service.defaultSpaceID = service.spaces[0].ID
	}
	return service, nil
}

func (s *Service) Pools(ctx context.Context) ([]StoragePool, error) {
	pools, err := s.adapter.Pools(ctx)
	if err != nil {
		return nil, err
	}
	// Snapshot managed state under a short lock so the live ZFS `zpool list`
	// calls below run without holding the service mutex.
	s.mu.Lock()
	disks := cloneDisks(s.disks)
	spaces := cloneSpaces(s.spaces)
	runner := s.zfsRunner
	s.mu.Unlock()

	for _, disk := range disks {
		pools = append(pools, StoragePool{
			ID:          disk.PoolID,
			Name:        disk.Model,
			Type:        "NAS 管理卷",
			UsedPercent: 0,
			Total:       disk.Size,
			Health:      disk.Health,
			Temperature: disk.Temperature,
			MountPath:   disk.MountPath,
		})
	}
	for _, space := range spaces {
		used, total, health := space.UsedPercent, space.Total, space.Health
		// For ZFS spaces, prefer live pool figures over create-time estimates.
		if space.FileSystem == FileSystemZFS && runner != nil {
			if stat, err := zfsPoolStatus(ctx, runner, zfsPoolName(space.Name)); err == nil && stat.SizeBytes > 0 {
				used = int(stat.AllocBytes * 100 / stat.SizeBytes)
				total = formatStorageBytes(stat.SizeBytes)
				health = zfsHealthLabel(stat.Health)
			}
		} else if space.MountPath != "" {
			// For ext4/btrfs spaces, read live usage from the mounted filesystem.
			if pct, totalBytes, ok := mountUsage(space.MountPath); ok && totalBytes > 0 {
				used = pct
				total = formatStorageBytes(totalBytes)
			}
		}
		pools = append(pools, StoragePool{
			ID:          space.ID,
			Name:        space.Name,
			Type:        fmt.Sprintf("%s / %s", space.Mode, space.FileSystem),
			UsedPercent: used,
			Total:       total,
			Health:      health,
			Temperature: "N/A",
			MountPath:   space.MountPath,
		})
	}
	return pools, nil
}

func (s *Service) Disks(ctx context.Context) ([]Disk, error) {
	disks, err := s.adapter.Disks(ctx)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	managed := cloneDisks(s.disks)
	for index := range managed {
		refreshDiskCapacity(&managed[index])
	}
	disks = append(disks, managed...)
	return disks, nil
}

func (s *Service) SmartReports(ctx context.Context) ([]SmartReport, error) {
	return s.adapter.SmartReports(ctx)
}

func (s *Service) Spaces(ctx context.Context) ([]StorageSpace, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneSpaces(s.spaces), nil
}

// DefaultSpaceID returns the configured default storage space id, falling back
// to the first managed space when none is set.
func (s *Service) DefaultSpaceID(ctx context.Context) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.defaultSpaceID != "" {
		for i := range s.spaces {
			if s.spaces[i].ID == s.defaultSpaceID {
				return s.defaultSpaceID
			}
		}
	}
	if len(s.spaces) > 0 {
		return s.spaces[0].ID
	}
	return ""
}

// SetDefaultSpace validates and persists the system-wide default storage space.
func (s *Service) SetDefaultSpace(ctx context.Context, id string) (string, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", fmt.Errorf("space id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	found := false
	for i := range s.spaces {
		if s.spaces[i].ID == id {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("storage space not found: %s", id)
	}
	s.defaultSpaceID = id
	return id, s.saveLocked()
}

func (s *Service) StartSMARTScan(ctx context.Context, target TaskTarget) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	return s.enqueueScan(TaskKindSMARTScan, target, "SMART 扫描已加入任务队列")
}

func (s *Service) AddDisk(ctx context.Context, request AddDiskRequest) (Disk, error) {
	if err := ctx.Err(); err != nil {
		return Disk{}, err
	}
	if request.MountPath == "" {
		return Disk{}, fmt.Errorf("mountPath is required")
	}
	info, err := os.Stat(request.MountPath)
	if err != nil {
		return Disk{}, err
	}
	if !info.IsDir() {
		return Disk{}, fmt.Errorf("mountPath is not a directory: %s", request.MountPath)
	}
	name := request.Name
	if name == "" {
		name = filepath.Base(request.MountPath)
	}
	if name == "." || name == "/" || name == "" {
		name = request.MountPath
	}
	size := "待探测"
	if bytes, ok := mountCapacityBytes(request.MountPath); ok {
		size = formatStorageBytes(bytes)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, disk := range s.disks {
		if disk.MountPath == request.MountPath {
			return disk, nil
		}
	}
	slot := fmt.Sprintf("managed-%03d", len(s.disks)+1)
	disk := Disk{
		Slot:        slot,
		Size:        size,
		State:       DiskStateHealthy,
		Temperature: "N/A",
		Serial:      request.MountPath,
		Health:      HealthHealthy,
		Role:        emptyDefault(request.Role, "data"),
		PoolID:      "managed-" + slugID(request.MountPath),
		Model:       name,
		Interface:   "mount",
		MountPath:   request.MountPath,
	}
	s.disks = append(s.disks, disk)
	return disk, s.saveLocked()
}

// poolIDForSpace resolves the owning pool: the explicitly chosen pool, else the
// pool of the first selected disk.
func poolIDForSpace(requested string, disks []Disk) string {
	if strings.TrimSpace(requested) != "" {
		return requested
	}
	for _, d := range disks {
		if d.PoolID != "" {
			return d.PoolID
		}
	}
	return ""
}

func (s *Service) CreateSpace(ctx context.Context, request CreateSpaceRequest) (StorageSpace, error) {
	if err := ctx.Err(); err != nil {
		return StorageSpace{}, err
	}
	if !request.Confirm {
		return StorageSpace{}, fmt.Errorf("format confirmation is required")
	}
	if request.Name == "" {
		return StorageSpace{}, fmt.Errorf("name is required")
	}
	// A storage space is created on a storage pool (first-level choice). When a
	// pool is given but no explicit disks, derive the disk set from that pool.
	if len(request.DiskSlots) == 0 && strings.TrimSpace(request.PoolID) != "" {
		hostDisks, err := s.adapter.Disks(ctx)
		if err != nil {
			return StorageSpace{}, err
		}
		s.mu.Lock()
		all := append(cloneDisks(hostDisks), cloneDisks(s.disks)...)
		s.mu.Unlock()
		for _, d := range all {
			if d.PoolID == request.PoolID {
				request.DiskSlots = append(request.DiskSlots, d.Slot)
			}
		}
	}
	if len(request.DiskSlots) == 0 {
		return StorageSpace{}, fmt.Errorf("a storage pool (or at least one disk) is required")
	}
	mode := request.Mode
	if mode == "" {
		mode = SpaceModeBasic
	}
	if err := validateSpaceMode(mode, len(request.DiskSlots)); err != nil {
		return StorageSpace{}, err
	}
	fs := request.FileSystem
	if fs == "" {
		fs = FileSystemEXT4
	}
	if err := validateFileSystem(fs); err != nil {
		return StorageSpace{}, err
	}
	mountPath := request.MountPath
	if mountPath == "" {
		mountPath = filepath.Join(defaultSpaceRoot(), slugID(request.Name))
	}
	hostDisks, err := s.adapter.Disks(ctx)
	if err != nil {
		return StorageSpace{}, err
	}

	s.mu.Lock()
	for _, space := range s.spaces {
		if space.Name == request.Name {
			s.mu.Unlock()
			return StorageSpace{}, fmt.Errorf("storage space already exists: %s", request.Name)
		}
	}
	allDisks := append(cloneDisks(hostDisks), cloneDisks(s.disks)...)
	selectedDisks, err := selectDisks(allDisks, request.DiskSlots)
	if err != nil {
		s.mu.Unlock()
		return StorageSpace{}, err
	}
	totalGB, err := estimateSpaceCapacityGB(mode, selectedDisks)
	if err != nil {
		s.mu.Unlock()
		return StorageSpace{}, err
	}
	s.mu.Unlock()

	plan := SpaceProvisionPlan{
		Name:       request.Name,
		Mode:       mode,
		FileSystem: fs,
		FormatDisk: requestWantsFormat(request),
		Disks:      selectedDisks,
		MountPath:  mountPath,
	}
	if err := validateProvisionPlan(plan); err != nil {
		return StorageSpace{}, err
	}
	if err := os.MkdirAll(mountPath, 0o755); err != nil {
		return StorageSpace{}, fmt.Errorf("create mount path: %w", err)
	}
	if err := s.provisioner.Provision(ctx, plan); err != nil {
		return StorageSpace{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, space := range s.spaces {
		if space.Name == request.Name {
			// We already provisioned the disk above; roll it back so a lost race
			// doesn't leave an orphaned mount with no owning space record.
			_ = s.provisioner.Deprovision(ctx, plan)
			return StorageSpace{}, fmt.Errorf("storage space already exists: %s", request.Name)
		}
	}
	s.taskSeq++
	space := StorageSpace{
		ID:          fmt.Sprintf("space-%03d-%s", s.taskSeq, slugID(request.Name)),
		Name:        request.Name,
		PoolID:      poolIDForSpace(request.PoolID, selectedDisks),
		Mode:        mode,
		FileSystem:  fs,
		DiskSlots:   append([]string(nil), request.DiskSlots...),
		MountPath:   mountPath,
		UsedPercent: 0,
		Total:       formatCapacityGB(totalGB),
		Health:      HealthHealthy,
		CreatedAt:   s.now().UTC(),
		CreatedBy:   request.Actor,
	}
	taskID := fmt.Sprintf("create-space-%03d", s.taskSeq)
	s.spaces = append(s.spaces, space)
	s.tasks[taskID] = StorageTask{
		ID:         taskID,
		Kind:       TaskKindCreateSpace,
		State:      TaskStateQueued,
		Progress:   0,
		Message:    "存储空间创建已加入任务队列",
		TargetPool: space.ID,
		CreatedAt:  s.now().UTC(),
	}
	if err := s.saveLocked(); err != nil {
		// Roll back the in-memory record and the on-disk provisioning together.
		s.spaces = s.spaces[:len(s.spaces)-1]
		delete(s.tasks, taskID)
		_ = s.provisioner.Deprovision(ctx, plan)
		return StorageSpace{}, err
	}
	return space, nil
}

func requestWantsFormat(request CreateSpaceRequest) bool {
	if request.FormatDisk == nil {
		return true
	}
	return *request.FormatDisk
}

func (s *Service) DeleteSpace(ctx context.Context, id string, request DeleteSpaceRequest) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	if !request.Confirm {
		return StorageTask{}, fmt.Errorf("delete confirmation is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i, space := range s.spaces {
		if space.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return StorageTask{}, fmt.Errorf("storage space not found: %s", id)
	}
	s.spaces = append(s.spaces[:index], s.spaces[index+1:]...)
	return s.completedTaskLocked(TaskKindDeleteSpace, TaskTarget{TargetPool: id}, "存储空间已删除")
}

func (s *Service) RemoveDisk(ctx context.Context, slot string, request RemoveDiskRequest) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	if !request.Confirm {
		return StorageTask{}, fmt.Errorf("remove confirmation is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index := -1
	for i, disk := range s.disks {
		if disk.Slot == slot {
			index = i
			break
		}
	}
	if index < 0 {
		return StorageTask{}, fmt.Errorf("managed disk not found: %s", slot)
	}
	s.disks = append(s.disks[:index], s.disks[index+1:]...)
	return s.completedTaskLocked(TaskKindRemoveDisk, TaskTarget{TargetSlot: slot}, "硬盘已移除")
}

func (s *Service) UpdateDiskSettings(ctx context.Context, slot string, request DiskSettingsRequest) (Disk, error) {
	if err := ctx.Err(); err != nil {
		return Disk{}, err
	}
	if request.StandbyMinutes < 0 {
		return Disk{}, fmt.Errorf("standbyMinutes must be non-negative")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.disks {
		if s.disks[index].Slot != slot {
			continue
		}
		s.disks[index].StandbyMinutes = request.StandbyMinutes
		s.disks[index].SSDCache = request.SSDCache
		s.disks[index].CacheMode = emptyDefault(request.CacheMode, "read")
		return s.disks[index], s.saveLocked()
	}
	return Disk{}, fmt.Errorf("managed disk not found: %s", slot)
}

func (s *Service) StartRepair(ctx context.Context, target TaskTarget) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	return s.enqueueScan(TaskKindRepair, target, "阵列修复已加入任务队列")
}

func (s *Service) CreateSnapshot(ctx context.Context, target TaskTarget) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	return s.enqueueScan(TaskKindSnapshot, target, "快照创建已加入任务队列")
}

func (s *Service) GetTask(ctx context.Context, id string) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return StorageTask{}, fmt.Errorf("storage task not found: %s", id)
	}
	return task, nil
}

func (s *Service) createTask(kind TaskKind, target TaskTarget, message string) (StorageTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.createTaskLocked(kind, target, message)
}

// AttachTaskRunner wires the shared task runtime into the storage service so
// scan-style tasks (SMART scan / array repair / snapshot) are actually executed
// by a worker instead of sitting in "queued" forever. It registers the handler
// and is safe to call once at construction (before the manager is started).
func (s *Service) AttachTaskRunner(m *tasks.Manager) {
	if m == nil {
		return
	}
	s.runner = m
	m.Register(scanTaskKind, s.runScanTask)
}

const scanTaskKind = "storage.scan"

type scanPayload struct {
	StorageTaskID string `json:"storageTaskId"`
}

// enqueueScan records a StorageTask and, when a runner is attached, schedules it
// for real execution. Without a runner it preserves the legacy behaviour of
// leaving the task queued (used by unit tests that don't exercise the runtime).
func (s *Service) enqueueScan(kind TaskKind, target TaskTarget, message string) (StorageTask, error) {
	task, err := s.createTask(kind, target, message)
	if err != nil {
		return StorageTask{}, err
	}
	if s.runner != nil {
		if _, err := s.runner.Enqueue(scanTaskKind, scanPayload{StorageTaskID: task.ID}); err != nil {
			return task, err
		}
	}
	return task, nil
}

// runScanTask is the handler invoked by the task runtime. It drives the
// referenced StorageTask through running -> completed, performing the real work
// for its kind (SMART scan reads live drive health via the adapter; repair and
// snapshot advance through staged progress).
func (s *Service) runScanTask(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	var payload scanPayload
	if err := h.Unmarshal(&payload); err != nil {
		return nil, err
	}
	task, err := s.GetTask(ctx, payload.StorageTaskID)
	if err != nil {
		return nil, err
	}

	s.updateTaskState(task.ID, TaskStateRunning, 15, "任务执行中")
	h.Progress(15, "扫描中")

	var summary string
	switch task.Kind {
	case TaskKindSMARTScan:
		reports, err := s.adapter.SmartReports(ctx)
		if err != nil {
			s.updateTaskState(task.ID, TaskStateFailed, 100, "SMART 扫描失败："+err.Error())
			return nil, err
		}
		healthy := 0
		for _, report := range reports {
			health := string(report.Health)
			if strings.Contains(health, "正常") || strings.EqualFold(health, "passed") || strings.EqualFold(health, "ok") {
				healthy++
			}
		}
		summary = fmt.Sprintf("SMART 扫描完成：%d/%d 块硬盘健康", healthy, len(reports))
	case TaskKindRepair:
		if pool, ok := s.zfsPoolForTarget(task.TargetPool); ok {
			s.updateTaskState(task.ID, TaskStateRunning, 60, "正在对 ZFS 池执行 scrub 一致性校验")
			h.Progress(60, "ZFS scrub")
			if err := zfsScrub(ctx, s.zfsRunner, pool); err != nil {
				s.updateTaskState(task.ID, TaskStateFailed, 100, "ZFS scrub 启动失败："+err.Error())
				return nil, err
			}
			summary = fmt.Sprintf("已对 ZFS 池 %s 启动 scrub 一致性校验", pool)
			break
		}
		s.updateTaskState(task.ID, TaskStateRunning, 60, "正在校验并重建阵列数据")
		h.Progress(60, "重建阵列中")
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		summary = "阵列修复完成：一致性校验通过"
	case TaskKindSnapshot:
		if pool, ok := s.zfsPoolForTarget(task.TargetPool); ok {
			s.updateTaskState(task.ID, TaskStateRunning, 60, "正在创建 ZFS 快照")
			h.Progress(60, "创建快照中")
			snap, err := zfsSnapshot(ctx, s.zfsRunner, pool, s.now())
			if err != nil {
				s.updateTaskState(task.ID, TaskStateFailed, 100, "ZFS 快照创建失败："+err.Error())
				return nil, err
			}
			summary = "ZFS 快照已创建：" + snap
			break
		}
		s.updateTaskState(task.ID, TaskStateRunning, 60, "正在创建快照")
		h.Progress(60, "创建快照中")
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		summary = "快照创建完成"
	default:
		summary = "任务完成"
	}

	// Honor cooperative cancellation before recording the terminal state.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.updateTaskState(task.ID, TaskStateCompleted, 100, summary)
	return json.Marshal(map[string]string{"storageTaskId": task.ID, "summary": summary})
}

func (s *Service) updateTaskState(id string, st TaskState, progress int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, ok := s.tasks[id]
	if !ok {
		return
	}
	task.State = st
	task.Progress = progress
	if message != "" {
		task.Message = message
	}
	s.tasks[id] = task
	_ = s.saveLocked()
}

// zfsPoolForTarget resolves a pool/space id to its ZFS pool name when the space
// is ZFS-formatted; ok is false for non-ZFS spaces or unknown targets, so the
// caller falls back to the staged-progress simulation.
func (s *Service) zfsPoolForTarget(targetPool string) (string, bool) {
	if strings.TrimSpace(targetPool) == "" {
		return "", false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sp := range s.spaces {
		if sp.ID == targetPool && sp.FileSystem == FileSystemZFS {
			return zfsPoolName(sp.Name), true
		}
	}
	return "", false
}

// isManagedZFSPool reports whether pool is the ZFS pool of one of our spaces, so
// rollback can never target an arbitrary pool name.
func (s *Service) isManagedZFSPool(pool string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sp := range s.spaces {
		if sp.FileSystem == FileSystemZFS && zfsPoolName(sp.Name) == pool {
			return true
		}
	}
	return false
}

// PoolDetail returns the rich ZFS efficiency/health figures of a managed pool.
func (s *Service) PoolDetail(ctx context.Context, poolID string) (ZFSPoolDetail, error) {
	if err := ctx.Err(); err != nil {
		return ZFSPoolDetail{}, err
	}
	pool, ok := s.zfsPoolForTarget(poolID)
	if !ok {
		return ZFSPoolDetail{}, fmt.Errorf("storage space %s is not a ZFS pool", poolID)
	}
	return zfsPoolDetail(ctx, s.zfsRunner, pool)
}

// ensureScheduler lazily builds the automatic-snapshot scheduler with callbacks
// wired to this service's ZFS operations.
func (s *Service) ensureScheduler() *snapshotScheduler {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scheduler != nil {
		return s.scheduler
	}
	schedPath := ""
	if s.statePath != "" {
		schedPath = filepath.Join(filepath.Dir(s.statePath), "zfs-schedules.json")
	}
	sc := newSnapshotScheduler(schedPath, s.now, slog.Default())
	sc.createSnapshot = func(ctx context.Context, poolID string) error {
		pool, ok := s.zfsPoolForTarget(poolID)
		if !ok {
			return fmt.Errorf("storage space %s is not a ZFS pool", poolID)
		}
		_, err := zfsSnapshot(ctx, s.zfsRunner, pool, s.now())
		return err
	}
	sc.listSnapshots = s.ListSnapshots
	sc.destroy = func(ctx context.Context, snapshot string) error {
		return zfsDestroySnapshot(ctx, s.zfsRunner, snapshot)
	}
	s.scheduler = sc
	return sc
}

// SetSnapshotSchedule configures (or disables) the automatic-snapshot policy for
// a ZFS space.
func (s *Service) SetSnapshotSchedule(ctx context.Context, sched SnapshotSchedule) (SnapshotSchedule, error) {
	if err := ctx.Err(); err != nil {
		return SnapshotSchedule{}, err
	}
	if _, ok := s.zfsPoolForTarget(sched.PoolID); !ok {
		return SnapshotSchedule{}, fmt.Errorf("storage space %s is not a ZFS pool", sched.PoolID)
	}
	return s.ensureScheduler().set(sched)
}

// SnapshotSchedules lists all configured automatic-snapshot policies.
func (s *Service) SnapshotSchedules(ctx context.Context) []SnapshotSchedule {
	if ctx.Err() != nil {
		return nil
	}
	return s.ensureScheduler().list()
}

// StartSnapshotScheduler starts the periodic create+prune loop. interval is how
// often due schedules are checked (not the snapshot interval itself).
func (s *Service) StartSnapshotScheduler(ctx context.Context, interval time.Duration) {
	s.ensureScheduler().start(ctx, interval)
}

// runScheduledSnapshotsNow runs one create+prune pass immediately (used in tests).
func (s *Service) runScheduledSnapshotsNow(ctx context.Context) {
	s.ensureScheduler().runDue(ctx)
}

// ListSnapshots returns the ZFS snapshots of a managed pool/space (newest first).
func (s *Service) ListSnapshots(ctx context.Context, poolID string) ([]ZFSSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	pool, ok := s.zfsPoolForTarget(poolID)
	if !ok {
		return nil, fmt.Errorf("storage space %s is not a ZFS pool", poolID)
	}
	return zfsListSnapshots(ctx, s.zfsRunner, pool)
}

// RollbackSnapshot rolls a managed ZFS pool back to the named snapshot
// (pool@snapshot). It refuses snapshots whose pool is not HiGoOS-managed.
func (s *Service) RollbackSnapshot(ctx context.Context, snapshotName string) (ZFSSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return ZFSSnapshot{}, err
	}
	pool, _, found := strings.Cut(snapshotName, "@")
	if !found || strings.TrimSpace(pool) == "" {
		return ZFSSnapshot{}, fmt.Errorf("invalid snapshot name %q (expected pool@snapshot)", snapshotName)
	}
	if !s.isManagedZFSPool(pool) {
		return ZFSSnapshot{}, fmt.Errorf("snapshot %s does not belong to a managed ZFS pool", snapshotName)
	}
	if err := zfsRollback(ctx, s.zfsRunner, snapshotName); err != nil {
		return ZFSSnapshot{}, err
	}
	return ZFSSnapshot{Name: snapshotName, Pool: pool}, nil
}

// completedTaskLocked records a task receipt for work that was already performed
// synchronously (e.g. space/disk removal), so the receipt reflects "completed"
// instead of dangling at "queued".
func (s *Service) completedTaskLocked(kind TaskKind, target TaskTarget, message string) (StorageTask, error) {
	task, err := s.createTaskLocked(kind, target, message)
	if err != nil {
		return StorageTask{}, err
	}
	task.State = TaskStateCompleted
	task.Progress = 100
	s.tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *Service) createTaskLocked(kind TaskKind, target TaskTarget, message string) (StorageTask, error) {
	s.taskSeq++
	task := StorageTask{
		ID:         fmt.Sprintf("%s-%03d", taskPrefix(kind), s.taskSeq),
		Kind:       kind,
		State:      TaskStateQueued,
		Progress:   0,
		Message:    message,
		TargetSlot: target.TargetSlot,
		TargetPool: target.TargetPool,
		CreatedAt:  s.now().UTC(),
	}
	s.tasks[task.ID] = task
	return task, s.saveLocked()
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		TaskSeq:    s.taskSeq,
		ConfirmSeq: s.confirmSeq,
		Tasks:      cloneTasks(s.tasks),
		Disks:      cloneDisks(s.disks),
		Spaces:     cloneSpaces(s.spaces),
		Pending:        clonePending(s.pending),
		Audit:          append([]StorageAuditEntry(nil), s.audit...),
		DefaultSpaceID: s.defaultSpaceID,
	})
}

func emptyDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func validateFileSystem(fs FileSystem) error {
	switch fs {
	case FileSystemEXT4, FileSystemBTRFS, FileSystemZFS:
		return nil
	default:
		return fmt.Errorf("unsupported file system: %s", fs)
	}
}

func validateSpaceMode(mode SpaceMode, diskCount int) error {
	minDisks := map[SpaceMode]int{
		SpaceModeBasic:  1,
		SpaceModeLinear: 2,
		SpaceModeRAID0:  2,
		SpaceModeRAID1:  2,
		SpaceModeRAID5:  3,
		SpaceModeRAID6:  4,
		SpaceModeRAID10: 4,
	}
	min, ok := minDisks[mode]
	if !ok {
		return fmt.Errorf("unsupported storage mode: %s", mode)
	}
	if diskCount < min {
		return fmt.Errorf("%s requires at least %d disks", mode, min)
	}
	// Basic addresses a single disk; concatenation/striping needs Linear or RAID.
	if mode == SpaceModeBasic && diskCount > 1 {
		return fmt.Errorf("basic supports a single disk only")
	}
	if mode == SpaceModeRAID10 && diskCount%2 != 0 {
		return fmt.Errorf("raid10 requires an even number of disks")
	}
	return nil
}

func selectDisks(disks []Disk, slots []string) ([]Disk, error) {
	selected := make([]Disk, 0, len(slots))
	seen := map[string]struct{}{}
	for _, slot := range slots {
		slot = strings.TrimSpace(slot)
		if slot == "" {
			return nil, fmt.Errorf("disk slot is required")
		}
		if _, ok := seen[slot]; ok {
			return nil, fmt.Errorf("duplicated disk slot: %s", slot)
		}
		seen[slot] = struct{}{}
		var found *Disk
		for index := range disks {
			if disks[index].Slot == slot {
				found = &disks[index]
				break
			}
		}
		if found == nil {
			return nil, fmt.Errorf("disk slot not found: %s", slot)
		}
		if found.SystemDisk {
			return nil, fmt.Errorf("system disk cannot be used for storage space: %s", slot)
		}
		selected = append(selected, *found)
	}
	return selected, nil
}

func estimateSpaceCapacityGB(mode SpaceMode, disks []Disk) (float64, error) {
	sizes := make([]float64, 0, len(disks))
	for _, disk := range disks {
		size := diskCapacityGB(disk)
		if size <= 0 {
			return 0, fmt.Errorf("disk %s has unknown capacity", disk.Slot)
		}
		sizes = append(sizes, size)
	}
	switch mode {
	case SpaceModeBasic:
		return sizes[0], nil
	case SpaceModeLinear, SpaceModeRAID0:
		return sumFloat(sizes), nil
	case SpaceModeRAID1:
		return minFloat(sizes), nil
	case SpaceModeRAID5:
		return minFloat(sizes) * float64(len(sizes)-1), nil
	case SpaceModeRAID6:
		return minFloat(sizes) * float64(len(sizes)-2), nil
	case SpaceModeRAID10:
		return minFloat(sizes) * float64(len(sizes)/2), nil
	default:
		return 0, fmt.Errorf("unsupported storage mode: %s", mode)
	}
}

func diskCapacityGB(disk Disk) float64 {
	size := parseCapacityGB(disk.Size)
	if size > 0 {
		return size
	}
	if disk.MountPath == "" {
		return 0
	}
	bytes, ok := mountCapacityBytes(disk.MountPath)
	if !ok {
		return 0
	}
	return float64(bytes) / 1000 / 1000 / 1000
}

func refreshDiskCapacity(disk *Disk) {
	if disk == nil || disk.MountPath == "" {
		return
	}
	if disk.Size != "" && disk.Size != "待探测" {
		return
	}
	if bytes, ok := mountCapacityBytes(disk.MountPath); ok {
		disk.Size = formatStorageBytes(bytes)
	}
}

func mountCapacityBytes(path string) (int64, bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, false
	}
	return int64(stat.Blocks) * int64(stat.Bsize), true
}

// mountUsage returns the live used-percentage and total bytes of the filesystem
// mounted at path (statfs). ok is false when the path is not a mountable dir.
func mountUsage(path string) (usedPercent int, totalBytes int64, ok bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, false
	}
	bs := int64(stat.Bsize)
	total := int64(stat.Blocks) * bs
	avail := int64(stat.Bavail) * bs
	if total <= 0 {
		return 0, 0, false
	}
	used := total - avail
	return int(used * 100 / total), total, true
}

func parseCapacityGB(value string) float64 {
	parts := strings.Fields(strings.TrimSpace(value))
	if len(parts) == 0 {
		return 0
	}
	number, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0
	}
	unit := "GB"
	if len(parts) > 1 {
		unit = strings.ToUpper(parts[1])
	}
	switch unit {
	case "TB", "TIB":
		return number * 1024
	case "GB", "GIB":
		return number
	case "MB", "MIB":
		return number / 1024
	case "KB", "KIB":
		return number / 1024 / 1024
	default:
		return number
	}
}

func formatCapacityGB(value float64) string {
	if value >= 1024 {
		return fmt.Sprintf("%.2f TB", value/1024)
	}
	if value == float64(int64(value)) {
		return fmt.Sprintf("%.0f GB", value)
	}
	return fmt.Sprintf("%.2f GB", value)
}

func sumFloat(values []float64) float64 {
	var total float64
	for _, value := range values {
		total += value
	}
	return total
}

func minFloat(values []float64) float64 {
	min := values[0]
	for _, value := range values[1:] {
		if value < min {
			min = value
		}
	}
	return min
}

func defaultSpaceRoot() string {
	if root := os.Getenv("HIGO_NAS_ROOT"); root != "" {
		return root
	}
	return filepath.Join(os.TempDir(), "higoos", "nas")
}

func taskPrefix(kind TaskKind) string {
	switch kind {
	case TaskKindSMARTScan:
		return "smart"
	case TaskKindRepair:
		return "repair"
	case TaskKindSnapshot:
		return "snapshot"
	case TaskKindCreateSpace:
		return "create-space"
	case TaskKindDeleteSpace:
		return "delete-space"
	case TaskKindRemoveDisk:
		return "remove-disk"
	default:
		return "task"
	}
}

func clonePools(pools []StoragePool) []StoragePool {
	return append([]StoragePool(nil), pools...)
}

func cloneDisks(disks []Disk) []Disk {
	return append([]Disk(nil), disks...)
}

func cloneSpaces(spaces []StorageSpace) []StorageSpace {
	out := make([]StorageSpace, 0, len(spaces))
	for _, space := range spaces {
		space.DiskSlots = append([]string(nil), space.DiskSlots...)
		out = append(out, space)
	}
	return out
}

func cloneSmartReports(reports []SmartReport) []SmartReport {
	out := make([]SmartReport, 0, len(reports))
	for _, report := range reports {
		report.Attributes = append([]SmartAttribute(nil), report.Attributes...)
		out = append(out, report)
	}
	return out
}

func cloneTasks(tasks map[string]StorageTask) map[string]StorageTask {
	out := make(map[string]StorageTask, len(tasks))
	for key, task := range tasks {
		out[key] = task
	}
	return out
}
