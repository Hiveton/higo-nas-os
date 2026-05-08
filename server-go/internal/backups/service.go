package backups

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

type Job struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Source    string `json:"source"`
	Target    string `json:"target"`
	State     string `json:"state"`
	Schedule  string `json:"schedule"`
	Progress  int    `json:"progress"`
	Speed     string `json:"speed"`
	ETA       string `json:"eta"`
	LastRun   string `json:"lastRun"`
	NextRun   string `json:"nextRun"`
	Retention string `json:"retention"`
	Policy    string `json:"policy"`
	Health    string `json:"health"`
	Enabled   bool   `json:"enabled"`
}

type Service struct {
	mu        sync.RWMutex
	jobs      []Job
	statePath string
}

func NewService() *Service {
	return &Service{jobs: []Job{
		{
			ID:        "family-photo",
			Name:      "家庭相册增量备份",
			Source:    "照片与视频",
			Target:    "异地备份卷",
			State:     "同步中",
			Schedule:  "每 6 小时",
			Progress:  72,
			Speed:     "18 MB/s",
			ETA:       "剩余 16 分钟",
			LastRun:   "今天 08:30",
			NextRun:   "今天 14:30",
			Retention: "保留 180 天",
			Policy:    "去重 + 加密 + 远端校验",
			Health:    "正常",
			Enabled:   true,
		},
		{
			ID:        "team-snapshot",
			Name:      "团队空间快照",
			Source:    "项目资料",
			Target:    "每日快照",
			State:     "校验中",
			Schedule:  "每天 02:00",
			Progress:  94,
			Speed:     "已校验 1.8 TB",
			ETA:       "等待归档索引",
			LastRun:   "今天 02:00",
			NextRun:   "明天 02:00",
			Retention: "保留 365 天",
			Policy:    "只读快照 + 变更审计",
			Health:    "正常",
			Enabled:   true,
		},
		{
			ID:        "system-config",
			Name:      "系统配置备份",
			Source:    "配置、权限、模型策略",
			Target:    "HiGoNAS 内部快照",
			State:     "已完成",
			Schedule:  "每日",
			Progress:  100,
			Speed:     "上次 42 秒",
			ETA:       "等待下次计划",
			LastRun:   "今天 09:20",
			NextRun:   "明天 09:20",
			Retention: "保留 90 天",
			Policy:    "配置签名 + 本地加密",
			Health:    "正常",
			Enabled:   true,
		},
	}}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "backups.json")
	var jobs []Job
	if err := state.LoadJSON(service.statePath, &jobs); err != nil {
		return nil, err
	}
	if len(jobs) > 0 {
		service.jobs = cloneJobs(jobs)
	}
	return service, nil
}

func (s *Service) Jobs(ctx context.Context) ([]Job, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneJobs(s.jobs), nil
}

func (s *Service) Run(ctx context.Context, id string) (Job, error) {
	return s.update(ctx, id, func(job *Job) {
		job.State = "同步中"
		job.Progress = minInt(99, maxInt(job.Progress+11, 82))
		job.Speed = "24 MB/s"
		job.ETA = "正在同步增量数据"
		job.LastRun = "刚刚"
		job.Health = "正常"
	})
}

func (s *Service) Pause(ctx context.Context, id string) (Job, error) {
	return s.update(ctx, id, func(job *Job) {
		job.State = "已暂停"
		job.Speed = "0 MB/s"
		job.ETA = "等待恢复"
	})
}

func (s *Service) Resume(ctx context.Context, id string) (Job, error) {
	return s.Run(ctx, id)
}

func (s *Service) Verify(ctx context.Context, id string) (Job, error) {
	return s.update(ctx, id, func(job *Job) {
		job.State = "校验中"
		job.Progress = minInt(100, maxInt(job.Progress, 96))
		job.Speed = "正在校验快照"
		job.ETA = "预计 3 分钟"
		job.LastRun = time.Now().Format("15:04")
		job.Health = "校验通过"
	})
}

func (s *Service) update(ctx context.Context, id string, mutate func(*Job)) (Job, error) {
	if err := ctx.Err(); err != nil {
		return Job{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.jobs {
		if s.jobs[index].ID == id {
			mutate(&s.jobs[index])
			if err := s.saveLocked(); err != nil {
				return Job{}, err
			}
			return s.jobs[index], nil
		}
	}
	return Job{}, fmt.Errorf("backup job not found: %s", id)
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, cloneJobs(s.jobs))
}

func cloneJobs(jobs []Job) []Job {
	out := make([]Job, len(jobs))
	copy(out, jobs)
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
