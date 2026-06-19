package backups

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// scheduledJob is the scheduling view of a backup job.
type scheduledJob struct {
	ID            string
	IntervalHours int
}

// backupScheduler runs enabled jobs automatically on their interval. The actual
// run is injected so the scheduler is testable without real file sync.
type backupScheduler struct {
	mu        sync.Mutex
	lastRun   map[string]time.Time
	statePath string
	now       func() time.Time
	logger    *slog.Logger

	enabledJobs func() []scheduledJob
	runJob      func(ctx context.Context, id string) error
}

func newBackupScheduler(statePath string, now func() time.Time, logger *slog.Logger) *backupScheduler {
	sc := &backupScheduler{lastRun: map[string]time.Time{}, statePath: statePath, now: now, logger: logger}
	if statePath != "" {
		var persisted map[string]time.Time
		if err := state.LoadJSON(statePath, &persisted); err == nil && persisted != nil {
			sc.lastRun = persisted
		}
	}
	return sc
}

// due returns the ids of enabled jobs whose interval has elapsed since last run.
func (sc *backupScheduler) due(now time.Time) []string {
	jobs := sc.enabledJobs()
	sc.mu.Lock()
	defer sc.mu.Unlock()
	var due []string
	for _, j := range jobs {
		if j.IntervalHours <= 0 {
			continue
		}
		last := sc.lastRun[j.ID]
		if last.IsZero() || now.Sub(last) >= time.Duration(j.IntervalHours)*time.Hour {
			due = append(due, j.ID)
		}
	}
	return due
}

func (sc *backupScheduler) markRun(id string, now time.Time) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.lastRun[id] = now
	if sc.statePath != "" {
		_ = state.SaveJSON(sc.statePath, sc.lastRun)
	}
}

// runDue triggers each due job's backup run.
func (sc *backupScheduler) runDue(ctx context.Context) {
	now := sc.now()
	for _, id := range sc.due(now) {
		if err := sc.runJob(ctx, id); err != nil {
			sc.logger.Warn("scheduled backup failed", slog.String("job", id), slog.Any("error", err))
			continue
		}
		sc.markRun(id, now)
	}
}

func (sc *backupScheduler) start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sc.runDue(ctx)
			}
		}
	}()
}
