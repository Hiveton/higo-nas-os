package storage

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestScheduleDue(t *testing.T) {
	now := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	if scheduleDue(SnapshotSchedule{Enabled: false, IntervalHours: 1}, now) {
		t.Fatal("disabled schedule must not be due")
	}
	if !scheduleDue(SnapshotSchedule{Enabled: true, IntervalHours: 6}, now) {
		t.Fatal("never-run schedule must be due")
	}
	recent := SnapshotSchedule{Enabled: true, IntervalHours: 6, LastRun: now.Add(-2 * time.Hour)}
	if scheduleDue(recent, now) {
		t.Fatal("schedule run 2h ago with 6h interval must not be due")
	}
	old := SnapshotSchedule{Enabled: true, IntervalHours: 6, LastRun: now.Add(-7 * time.Hour)}
	if !scheduleDue(old, now) {
		t.Fatal("schedule run 7h ago with 6h interval must be due")
	}
}

func TestSnapshotsToPruneKeepsNewest(t *testing.T) {
	snaps := []ZFSSnapshot{{Name: "p@c"}, {Name: "p@b"}, {Name: "p@a"}} // newest first
	prune := snapshotsToPrune(snaps, 2)
	if len(prune) != 1 || prune[0].Name != "p@a" {
		t.Fatalf("expected to prune the oldest (p@a), got %#v", prune)
	}
	if snapshotsToPrune(snaps, 5) != nil {
		t.Fatal("nothing to prune when keep >= count")
	}
}

func TestScheduledSnapshotCreatesAndPrunes(t *testing.T) {
	service := NewService(NewDevAdapter())
	fixed := time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixed }
	pool := zfsPoolName("媒体库")

	var mu sync.Mutex
	var cmds [][]string
	service.zfsRunner = func(_ context.Context, name string, args ...string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		cmds = append(cmds, append([]string{name}, args...))
		if name == zfsBin && len(args) > 1 && args[0] == "list" {
			return []byte(
				pool + "@higoos-20260101-120000\t1048576\t1735732800\n" +
					pool + "@higoos-20260102-120000\t1048576\t1735819200\n" +
					pool + "@higoos-20260103-120000\t1048576\t1735905600\n"), nil
		}
		return nil, nil
	}
	service.mu.Lock()
	service.spaces = []StorageSpace{{ID: "pool-1", Name: "媒体库", FileSystem: FileSystemZFS}}
	service.mu.Unlock()

	if _, err := service.SetSnapshotSchedule(context.Background(), SnapshotSchedule{
		PoolID: "pool-1", Enabled: true, IntervalHours: 6, Keep: 2,
	}); err != nil {
		t.Fatalf("set schedule: %v", err)
	}

	service.runScheduledSnapshotsNow(context.Background())

	mu.Lock()
	defer mu.Unlock()
	var created, destroyed bool
	for _, c := range cmds {
		if len(c) >= 3 && c[0] == zfsBin && c[1] == "snapshot" && strings.HasPrefix(c[2], pool+"@higoos-") {
			created = true
		}
		// keep=2 with 3 existing → the oldest (0101) is destroyed
		if len(c) >= 3 && c[0] == zfsBin && c[1] == "destroy" && c[2] == pool+"@higoos-20260101-120000" {
			destroyed = true
		}
	}
	if !created {
		t.Fatalf("scheduled run should create a snapshot, got commands: %v", cmds)
	}
	if !destroyed {
		t.Fatalf("scheduled run should prune the oldest snapshot, got commands: %v", cmds)
	}

	// LastRun should advance so it is not immediately due again.
	if scheduleDue(mustSchedule(t, service, "pool-1"), fixed) {
		t.Fatal("schedule should not be due again right after running")
	}
}

func mustSchedule(t *testing.T, s *Service, poolID string) SnapshotSchedule {
	t.Helper()
	for _, sc := range s.SnapshotSchedules(context.Background()) {
		if sc.PoolID == poolID {
			return sc
		}
	}
	t.Fatalf("schedule %s not found", poolID)
	return SnapshotSchedule{}
}

func TestSetScheduleRejectsNonZFSSpace(t *testing.T) {
	service := NewService(NewDevAdapter())
	service.mu.Lock()
	service.spaces = []StorageSpace{{ID: "pool-1", Name: "媒体库", FileSystem: FileSystemEXT4}}
	service.mu.Unlock()
	if _, err := service.SetSnapshotSchedule(context.Background(), SnapshotSchedule{PoolID: "pool-1", Enabled: true, IntervalHours: 6}); err == nil {
		t.Fatal("expected error setting schedule on non-ZFS space")
	}
}
