package backups

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"higoos/server-go/internal/tasks"
)

func TestBackupSchedulerAutoRunsAndRespectsInterval(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "doc.txt"), "scheduled backup")

	service := NewService()
	service.mu.Lock()
	service.jobs = service.jobs[:1]
	service.jobs[0].Source = src
	service.jobs[0].Target = dst
	service.jobs[0].Enabled = true
	service.jobs[0].IntervalHours = 6
	jobID := service.jobs[0].ID
	service.mu.Unlock()

	mgr, err := tasks.NewManager("", tasks.WithWorkers(1), tasks.WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("task manager: %v", err)
	}
	service.AttachTaskRunner(mgr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	sc := service.ensureScheduler()
	fixed := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	sc.now = func() time.Time { return fixed }

	// First pass: the enabled job is due (never run) → real sync runs.
	if due := sc.due(fixed); len(due) != 1 || due[0] != jobID {
		t.Fatalf("job should be due on first pass, got %#v", due)
	}
	service.runScheduledBackupsNow(ctx)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		jobs, _ := service.Jobs(ctx)
		if jobs[0].State == "已完成" {
			if _, err := os.Stat(filepath.Join(dst, "doc.txt")); err != nil {
				t.Fatalf("scheduled backup should copy the file: %v", err)
			}
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	// Right after running, the 6h interval has not elapsed → not due again.
	if due := sc.due(fixed); len(due) != 0 {
		t.Fatalf("job should not be due right after running, got %#v", due)
	}
	// 7h later it is due again.
	if due := sc.due(fixed.Add(7 * time.Hour)); len(due) != 1 {
		t.Fatalf("job should be due again after the interval, got %#v", due)
	}
}

func TestBackupSchedulerSkipsDisabledAndZeroInterval(t *testing.T) {
	service := NewService()
	service.mu.Lock()
	service.jobs = []Job{
		{ID: "disabled", Enabled: false, IntervalHours: 6},
		{ID: "manual", Enabled: true, IntervalHours: 0},
		{ID: "auto", Enabled: true, IntervalHours: 6},
	}
	service.mu.Unlock()
	sc := service.ensureScheduler()
	due := sc.due(time.Now())
	if len(due) != 1 || due[0] != "auto" {
		t.Fatalf("only the enabled+interval job should be due, got %#v", due)
	}
}

func TestSetJobScheduleComputesNextRun(t *testing.T) {
	service := NewService()
	id := service.jobs[0].ID

	job, err := service.SetJobSchedule(context.Background(), id, true, 6)
	if err != nil {
		t.Fatalf("set schedule: %v", err)
	}
	if job.IntervalHours != 6 || job.Schedule != "每 6 小时" {
		t.Fatalf("schedule fields not set: %#v", job)
	}
	// NextRun should be a computed MM-DD HH:MM ~6h ahead, not a demo string.
	want := time.Now().Add(6 * time.Hour).Format("01-02 15:04")
	if job.NextRun != want {
		t.Fatalf("NextRun=%q, want ~%q", job.NextRun, want)
	}

	disabled, err := service.SetJobSchedule(context.Background(), id, false, 0)
	if err != nil {
		t.Fatalf("disable: %v", err)
	}
	if disabled.Enabled || disabled.Schedule != "手动" || disabled.NextRun != "手动触发" {
		t.Fatalf("disabled schedule not reflected: %#v", disabled)
	}
}
