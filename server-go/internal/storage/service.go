package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

type Service struct {
	adapter Adapter
	now     func() time.Time

	mu        sync.Mutex
	taskSeq   int
	tasks     map[string]StorageTask
	statePath string
}

type snapshot struct {
	TaskSeq int                    `json:"taskSeq"`
	Tasks   map[string]StorageTask `json:"tasks"`
}

func NewService(adapter Adapter) *Service {
	if adapter == nil {
		adapter = NewHostAdapter()
	}
	return &Service{
		adapter: adapter,
		now:     time.Now,
		tasks:   map[string]StorageTask{},
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
	return service, nil
}

func (s *Service) Pools(ctx context.Context) ([]StoragePool, error) {
	return s.adapter.Pools(ctx)
}

func (s *Service) Disks(ctx context.Context) ([]Disk, error) {
	return s.adapter.Disks(ctx)
}

func (s *Service) SmartReports(ctx context.Context) ([]SmartReport, error) {
	return s.adapter.SmartReports(ctx)
}

func (s *Service) StartSMARTScan(ctx context.Context, target TaskTarget) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	return s.createTask(TaskKindSMARTScan, target, "SMART 扫描已加入任务队列")
}

func (s *Service) StartRepair(ctx context.Context, target TaskTarget) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	return s.createTask(TaskKindRepair, target, "阵列修复已加入任务队列")
}

func (s *Service) CreateSnapshot(ctx context.Context, target TaskTarget) (StorageTask, error) {
	if err := ctx.Err(); err != nil {
		return StorageTask{}, err
	}
	return s.createTask(TaskKindSnapshot, target, "快照创建已加入任务队列")
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
		TaskSeq: s.taskSeq,
		Tasks:   cloneTasks(s.tasks),
	})
}

func taskPrefix(kind TaskKind) string {
	switch kind {
	case TaskKindSMARTScan:
		return "smart"
	case TaskKindRepair:
		return "repair"
	case TaskKindSnapshot:
		return "snapshot"
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
