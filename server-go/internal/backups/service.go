package backups

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"higoos/server-go/internal/state"
	"higoos/server-go/internal/tasks"
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
	runner    *tasks.Manager
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

// Run marks a backup job as syncing and, when a task runner is attached,
// schedules a real incremental file sync (see syncTree). Without a runner it
// keeps the legacy state-string behaviour so unit tests stay deterministic.
func (s *Service) Run(ctx context.Context, id string) (Job, error) {
	job, err := s.update(ctx, id, func(job *Job) {
		job.State = "同步中"
		job.Progress = minInt(99, maxInt(job.Progress+11, 82))
		job.Speed = "正在同步增量数据"
		job.ETA = "进行中"
		job.LastRun = "刚刚"
		job.Health = "正常"
	})
	if err != nil {
		return Job{}, err
	}
	if s.runner != nil {
		if _, err := s.runner.Enqueue(backupRunTaskKind, backupRunPayload{JobID: id}); err != nil {
			return job, err
		}
	}
	return job, nil
}

const backupRunTaskKind = "backups.run"

type backupRunPayload struct {
	JobID string `json:"jobId"`
}

// AttachTaskRunner wires the shared task runtime so backup runs perform a real
// incremental file sync. Registers the handler; call once before Start.
func (s *Service) AttachTaskRunner(m *tasks.Manager) {
	if m == nil {
		return
	}
	s.runner = m
	m.Register(backupRunTaskKind, s.runBackupJob)
	m.Register(backupVerifyTaskKind, s.verifyBackupJob)
}

const backupVerifyTaskKind = "backups.verify"

// verifyBackupJob is the task handler that checksum-verifies a job's target
// against its source and records the outcome on the job.
func (s *Service) verifyBackupJob(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	var payload backupRunPayload
	if err := h.Unmarshal(&payload); err != nil {
		return nil, err
	}
	source, target, ok := s.jobPaths(payload.JobID)
	if !ok {
		return nil, fmt.Errorf("backup job not found: %s", payload.JobID)
	}
	h.Progress(20, "verifying")

	res, err := verifyTree(source, target)
	if err != nil {
		_, _ = s.update(context.Background(), payload.JobID, func(job *Job) {
			job.State = "源路径不可用"
			job.Health = "需要配置有效的源/目标路径"
			job.ETA = "已暂停"
		})
		return nil, err
	}
	_, _ = s.update(context.Background(), payload.JobID, func(job *Job) {
		job.Progress = 100
		job.LastRun = time.Now().Format("15:04")
		if res.Mismatch == 0 && res.Missing == 0 {
			job.State = "已完成"
			job.Health = "校验通过"
			job.Speed = fmt.Sprintf("校验 %d 个文件，全部一致", res.Checked)
			job.ETA = "等待下次计划"
		} else {
			job.State = "校验未通过"
			job.Health = fmt.Sprintf("发现 %d 处不一致 / %d 个缺失", res.Mismatch, res.Missing)
			job.ETA = "建议重新运行备份"
		}
	})
	return json.Marshal(map[string]any{
		"jobId":    payload.JobID,
		"checked":  res.Checked,
		"mismatch": res.Mismatch,
		"missing":  res.Missing,
	})
}

// runBackupJob is the task handler that executes a backup job's incremental
// sync and writes the outcome back to the job record.
func (s *Service) runBackupJob(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	var payload backupRunPayload
	if err := h.Unmarshal(&payload); err != nil {
		return nil, err
	}
	source, target, ok := s.jobPaths(payload.JobID)
	if !ok {
		return nil, fmt.Errorf("backup job not found: %s", payload.JobID)
	}
	h.Progress(20, "syncing")

	res, syncErr := syncTree(source, target)
	if syncErr != nil {
		// Honest failure: demo jobs point at logical space names, not real
		// paths, so they surface a clear blocked state instead of fake success.
		_, _ = s.update(context.Background(), payload.JobID, func(job *Job) {
			job.State = "源路径不可用"
			job.Health = "需要配置有效的源/目标路径"
			job.Speed = "0 MB/s"
			job.ETA = "已暂停"
		})
		return nil, syncErr
	}

	now := time.Now()
	_, _ = s.update(context.Background(), payload.JobID, func(job *Job) {
		job.State = "已完成"
		job.Progress = 100
		job.Health = "正常"
		job.Speed = fmt.Sprintf("复制 %d 个文件 / 跳过 %d 个", res.Copied, res.Skipped)
		job.ETA = "等待下次计划"
		job.LastRun = now.Format("15:04")
	})
	return json.Marshal(map[string]any{
		"jobId":   payload.JobID,
		"copied":  res.Copied,
		"skipped": res.Skipped,
		"bytes":   res.Bytes,
	})
}

func (s *Service) jobPaths(id string) (source, target string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, job := range s.jobs {
		if job.ID == id {
			return job.Source, job.Target, true
		}
	}
	return "", "", false
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

// Verify marks a job as verifying and, when a runner is attached, schedules a
// real checksum verification of the target against the source.
func (s *Service) Verify(ctx context.Context, id string) (Job, error) {
	job, err := s.update(ctx, id, func(job *Job) {
		job.State = "校验中"
		job.Speed = "正在校验快照"
		job.ETA = "进行中"
		job.LastRun = time.Now().Format("15:04")
	})
	if err != nil {
		return Job{}, err
	}
	if s.runner != nil {
		if _, err := s.runner.Enqueue(backupVerifyTaskKind, backupRunPayload{JobID: id}); err != nil {
			return job, err
		}
	}
	return job, nil
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
