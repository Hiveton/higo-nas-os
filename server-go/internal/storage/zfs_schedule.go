package storage

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// SnapshotSchedule is an automatic ZFS snapshot policy for one pool/space:
// take a snapshot every IntervalHours and keep only the newest Keep snapshots.
type SnapshotSchedule struct {
	PoolID        string    `json:"poolId"`
	Enabled       bool      `json:"enabled"`
	IntervalHours int       `json:"intervalHours"`
	Keep          int       `json:"keep"`
	LastRun       time.Time `json:"lastRun,omitempty"`
}

// scheduleDue reports whether a schedule should fire at now.
func scheduleDue(s SnapshotSchedule, now time.Time) bool {
	if !s.Enabled || s.IntervalHours <= 0 {
		return false
	}
	if s.LastRun.IsZero() {
		return true
	}
	return now.Sub(s.LastRun) >= time.Duration(s.IntervalHours)*time.Hour
}

// snapshotsToPrune returns the snapshots to destroy to honor a retention count.
// snaps must be newest-first (as ListSnapshots returns), so the tail is oldest.
func snapshotsToPrune(snaps []ZFSSnapshot, keep int) []ZFSSnapshot {
	if keep <= 0 || len(snaps) <= keep {
		return nil
	}
	return snaps[keep:]
}

// zfsDestroySnapshot destroys a single snapshot (pool@snap).
func zfsDestroySnapshot(ctx context.Context, runner commandRunner, snapshot string) error {
	if _, err := runner(ctx, zfsBin, "destroy", snapshot); err != nil {
		return err
	}
	return nil
}

// snapshotScheduler owns the per-pool snapshot policies, their persistence, and
// the periodic create+prune loop. The actual ZFS work is injected so the
// scheduler is testable without ZFS (and without importing Service internals).
type snapshotScheduler struct {
	mu        sync.Mutex
	schedules map[string]SnapshotSchedule
	statePath string
	now       func() time.Time
	logger    *slog.Logger

	createSnapshot func(ctx context.Context, poolID string) error
	listSnapshots  func(ctx context.Context, poolID string) ([]ZFSSnapshot, error)
	destroy        func(ctx context.Context, snapshot string) error
}

func newSnapshotScheduler(statePath string, now func() time.Time, logger *slog.Logger) *snapshotScheduler {
	sc := &snapshotScheduler{
		schedules: map[string]SnapshotSchedule{},
		statePath: statePath,
		now:       now,
		logger:    logger,
	}
	if statePath != "" {
		var persisted map[string]SnapshotSchedule
		if err := state.LoadJSON(statePath, &persisted); err == nil && persisted != nil {
			sc.schedules = persisted
		}
	}
	return sc
}

func (sc *snapshotScheduler) set(s SnapshotSchedule) (SnapshotSchedule, error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s.Keep <= 0 {
		s.Keep = 10
	}
	// preserve LastRun across edits
	if existing, ok := sc.schedules[s.PoolID]; ok {
		s.LastRun = existing.LastRun
	}
	sc.schedules[s.PoolID] = s
	return s, sc.saveLocked()
}

func (sc *snapshotScheduler) get(poolID string) (SnapshotSchedule, bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	s, ok := sc.schedules[poolID]
	return s, ok
}

func (sc *snapshotScheduler) list() []SnapshotSchedule {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	out := make([]SnapshotSchedule, 0, len(sc.schedules))
	for _, s := range sc.schedules {
		out = append(out, s)
	}
	return out
}

// dueLocked returns the pool ids whose schedules are due at now.
func (sc *snapshotScheduler) due(now time.Time) []string {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	var due []string
	for id, s := range sc.schedules {
		if scheduleDue(s, now) {
			due = append(due, id)
		}
	}
	return due
}

func (sc *snapshotScheduler) markRun(poolID string, now time.Time) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if s, ok := sc.schedules[poolID]; ok {
		s.LastRun = now
		sc.schedules[poolID] = s
		_ = sc.saveLocked()
	}
}

// runDue creates a snapshot for each due pool and prunes to the retention count.
func (sc *snapshotScheduler) runDue(ctx context.Context) {
	now := sc.now()
	for _, poolID := range sc.due(now) {
		if err := sc.createSnapshot(ctx, poolID); err != nil {
			sc.logger.Warn("scheduled snapshot failed", slog.String("pool", poolID), slog.Any("error", err))
			continue
		}
		sc.markRun(poolID, now)
		sc.prune(ctx, poolID)
	}
}

func (sc *snapshotScheduler) prune(ctx context.Context, poolID string) {
	s, ok := sc.get(poolID)
	if !ok {
		return
	}
	snaps, err := sc.listSnapshots(ctx, poolID)
	if err != nil {
		return
	}
	for _, victim := range snapshotsToPrune(snaps, s.Keep) {
		if err := sc.destroy(ctx, victim.Name); err != nil {
			sc.logger.Warn("snapshot prune failed", slog.String("snapshot", victim.Name), slog.Any("error", err))
		}
	}
}

// start runs the create+prune loop until ctx is canceled.
func (sc *snapshotScheduler) start(ctx context.Context, interval time.Duration) {
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

func (sc *snapshotScheduler) saveLocked() error {
	if sc.statePath == "" {
		return nil
	}
	return state.SaveJSON(sc.statePath, sc.schedules)
}
