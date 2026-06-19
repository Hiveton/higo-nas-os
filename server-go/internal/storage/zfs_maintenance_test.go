package storage

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"higoos/server-go/internal/tasks"
)

func newZFSTestService(t *testing.T, fs FileSystem) (*Service, *[][]string, func()) {
	t.Helper()
	service := NewService(NewDevAdapter())
	var mu sync.Mutex
	cmds := &[][]string{}
	service.zfsRunner = func(_ context.Context, name string, args ...string) ([]byte, error) {
		mu.Lock()
		defer mu.Unlock()
		*cmds = append(*cmds, append([]string{name}, args...))
		return nil, nil
	}
	service.mu.Lock()
	service.spaces = []StorageSpace{{ID: "pool-1", Name: "媒体库", FileSystem: fs, Mode: SpaceModeRAID5}}
	service.mu.Unlock()

	mgr, err := tasks.NewManager("", tasks.WithWorkers(1), tasks.WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("task manager: %v", err)
	}
	service.AttachTaskRunner(mgr)
	ctx, cancel := context.WithCancel(context.Background())
	mgr.Start(ctx)
	return service, cmds, func() { cancel(); mgr.Stop() }
}

func waitTaskState(t *testing.T, s *Service, id string, want TaskState) StorageTask {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		task, err := s.GetTask(context.Background(), id)
		if err == nil && (task.State == TaskStateCompleted || task.State == TaskStateFailed) {
			if task.State != want {
				t.Fatalf("task %s reached %q, want %q (msg=%q)", id, task.State, want, task.Message)
			}
			return task
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("task %s did not finish", id)
	return StorageTask{}
}

func TestZFSSnapshotAndScrubRunRealCommands(t *testing.T) {
	service, cmds, stop := newZFSTestService(t, FileSystemZFS)
	defer stop()
	ctx := context.Background()

	snapTask, err := service.CreateSnapshot(ctx, TaskTarget{TargetPool: "pool-1"})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	repairTask, err := service.StartRepair(ctx, TaskTarget{TargetPool: "pool-1"})
	if err != nil {
		t.Fatalf("start repair: %v", err)
	}
	snap := waitTaskState(t, service, snapTask.ID, TaskStateCompleted)
	waitTaskState(t, service, repairTask.ID, TaskStateCompleted)

	pool := zfsPoolName("媒体库")
	var sawSnap, sawScrub bool
	for _, c := range *cmds {
		if len(c) >= 3 && c[0] == zfsBin && c[1] == "snapshot" && strings.HasPrefix(c[2], pool+"@higoos-") {
			sawSnap = true
		}
		if len(c) >= 3 && c[0] == zpoolBin && c[1] == "scrub" && c[2] == pool {
			sawScrub = true
		}
	}
	if !sawSnap {
		t.Fatalf("expected a real `zfs snapshot %s@...`, got commands: %v", pool, *cmds)
	}
	if !sawScrub {
		t.Fatalf("expected a real `zpool scrub %s`, got commands: %v", pool, *cmds)
	}
	if !strings.Contains(snap.Message, "ZFS 快照已创建") {
		t.Fatalf("snapshot summary should mention the real ZFS snapshot, got %q", snap.Message)
	}
}

func TestPoolsReflectLiveZFSCapacityAndHealth(t *testing.T) {
	service := NewService(NewDevAdapter())
	// 10 TiB pool, 50% allocated, DEGRADED.
	service.zfsRunner = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == zpoolBin && len(args) > 0 && args[0] == "list" {
			return []byte("10995116277760\t5497558138880\tDEGRADED\n"), nil
		}
		return nil, nil
	}
	service.mu.Lock()
	service.spaces = []StorageSpace{{
		ID: "pool-1", Name: "媒体库", FileSystem: FileSystemZFS, Mode: SpaceModeRAID5,
		UsedPercent: 12, Total: "估算值", Health: HealthHealthy, // stale create-time estimates
	}}
	service.mu.Unlock()

	pools, err := service.Pools(context.Background())
	if err != nil {
		t.Fatalf("pools: %v", err)
	}
	var p *StoragePool
	for i := range pools {
		if pools[i].ID == "pool-1" {
			p = &pools[i]
		}
	}
	if p == nil {
		t.Fatalf("zfs space pool not found in %#v", pools)
	}
	if p.UsedPercent != 50 {
		t.Fatalf("expected live 50%% usage, got %d", p.UsedPercent)
	}
	if p.Health != HealthWarning {
		t.Fatalf("expected DEGRADED→警告, got %q", p.Health)
	}
	if p.Total == "估算值" {
		t.Fatalf("expected live total to replace the estimate, got %q", p.Total)
	}
}

func TestPoolsFallBackToStoredWhenZpoolFails(t *testing.T) {
	service := NewService(NewDevAdapter())
	// zpool unavailable (dev host) → keep stored estimates.
	service.zfsRunner = func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, context.DeadlineExceeded
	}
	service.mu.Lock()
	service.spaces = []StorageSpace{{
		ID: "pool-1", Name: "媒体库", FileSystem: FileSystemZFS,
		UsedPercent: 33, Total: "8 TB", Health: HealthHealthy,
	}}
	service.mu.Unlock()

	pools, _ := service.Pools(context.Background())
	for _, p := range pools {
		if p.ID == "pool-1" {
			if p.UsedPercent != 33 || p.Total != "8 TB" {
				t.Fatalf("expected stored estimates on zpool failure, got %d / %q", p.UsedPercent, p.Total)
			}
			return
		}
	}
	t.Fatal("pool not found")
}

func TestListSnapshotsParsesAndSortsNewestFirst(t *testing.T) {
	service := NewService(NewDevAdapter())
	pool := zfsPoolName("媒体库")
	service.zfsRunner = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == zfsBin && len(args) > 1 && args[0] == "list" && args[1] == "-t" {
			return []byte(
				pool + "@higoos-20260101-120000\t1048576\t1735732800\n" +
					pool + "@higoos-20260102-120000\t2097152\t1735819200\n"), nil
		}
		return nil, nil
	}
	service.mu.Lock()
	service.spaces = []StorageSpace{{ID: "pool-1", Name: "媒体库", FileSystem: FileSystemZFS}}
	service.mu.Unlock()

	snaps, err := service.ListSnapshots(context.Background(), "pool-1")
	if err != nil {
		t.Fatalf("list snapshots: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}
	if !strings.HasSuffix(snaps[0].Name, "20260102-120000") {
		t.Fatalf("expected newest snapshot first, got %s", snaps[0].Name)
	}
	if snaps[0].UsedBytes != 2097152 || snaps[0].Used == "" || snaps[0].CreatedAt.IsZero() {
		t.Fatalf("snapshot fields not parsed: %#v", snaps[0])
	}
}

func TestListSnapshotsRejectsNonZFSSpace(t *testing.T) {
	service := NewService(NewDevAdapter())
	service.mu.Lock()
	service.spaces = []StorageSpace{{ID: "pool-1", Name: "媒体库", FileSystem: FileSystemBTRFS}}
	service.mu.Unlock()
	if _, err := service.ListSnapshots(context.Background(), "pool-1"); err == nil {
		t.Fatal("expected error listing snapshots of a non-ZFS space")
	}
}

func TestRollbackSnapshotRunsCommandOnlyForManagedPool(t *testing.T) {
	service := NewService(NewDevAdapter())
	pool := zfsPoolName("媒体库")
	var cmds [][]string
	service.zfsRunner = func(_ context.Context, name string, args ...string) ([]byte, error) {
		cmds = append(cmds, append([]string{name}, args...))
		return nil, nil
	}
	service.mu.Lock()
	service.spaces = []StorageSpace{{ID: "pool-1", Name: "媒体库", FileSystem: FileSystemZFS}}
	service.mu.Unlock()
	ctx := context.Background()

	// Managed pool → runs zfs rollback.
	snap := pool + "@higoos-20260102-120000"
	if _, err := service.RollbackSnapshot(ctx, snap); err != nil {
		t.Fatalf("rollback managed: %v", err)
	}
	if len(cmds) != 1 || cmds[0][0] != zfsBin || cmds[0][1] != "rollback" || cmds[0][2] != snap {
		t.Fatalf("expected `zfs rollback %s`, got %v", snap, cmds)
	}

	// Unmanaged pool → refused, no command.
	cmds = nil
	if _, err := service.RollbackSnapshot(ctx, "tank@evil"); err == nil {
		t.Fatal("expected rollback of unmanaged pool to be refused")
	}
	if len(cmds) != 0 {
		t.Fatalf("unmanaged rollback must not run a command, got %v", cmds)
	}

	// Malformed name → error.
	if _, err := service.RollbackSnapshot(ctx, "no-at-sign"); err == nil {
		t.Fatal("expected error for malformed snapshot name")
	}
}

func TestPoolDetailReadsCompressionFragmentationDedup(t *testing.T) {
	service := NewService(NewDevAdapter())
	service.zfsRunner = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == zpoolBin && len(args) > 0 && args[0] == "list" {
			// size, alloc, free, cap, frag, dedup, health
			return []byte("10995116277760\t5497558138880\t5497558138880\t50\t12\t1.00\tONLINE\n"), nil
		}
		if name == zfsBin && len(args) > 0 && args[0] == "get" {
			return []byte("1.45x\n"), nil
		}
		return nil, nil
	}
	service.mu.Lock()
	service.spaces = []StorageSpace{{ID: "pool-1", Name: "媒体库", FileSystem: FileSystemZFS}}
	service.mu.Unlock()

	detail, err := service.PoolDetail(context.Background(), "pool-1")
	if err != nil {
		t.Fatalf("pool detail: %v", err)
	}
	if detail.CapacityPct != 50 || detail.Fragmentation != 12 {
		t.Fatalf("unexpected capacity/frag: %#v", detail)
	}
	if detail.CompressRatio != "1.45x" || detail.DedupRatio != "1.00x" {
		t.Fatalf("unexpected ratios: compress=%q dedup=%q", detail.CompressRatio, detail.DedupRatio)
	}
	if detail.Health != HealthHealthy || detail.SizeBytes != 10995116277760 {
		t.Fatalf("unexpected health/size: %#v", detail)
	}
}

func TestPoolDetailRejectsNonZFSSpace(t *testing.T) {
	service := NewService(NewDevAdapter())
	service.mu.Lock()
	service.spaces = []StorageSpace{{ID: "pool-1", Name: "媒体库", FileSystem: FileSystemEXT4}}
	service.mu.Unlock()
	if _, err := service.PoolDetail(context.Background(), "pool-1"); err == nil {
		t.Fatal("expected error for non-ZFS space detail")
	}
}

func TestNonZFSSpaceUsesSimulationNotZpool(t *testing.T) {
	service, cmds, stop := newZFSTestService(t, FileSystemBTRFS)
	defer stop()
	ctx := context.Background()

	snapTask, err := service.CreateSnapshot(ctx, TaskTarget{TargetPool: "pool-1"})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}
	waitTaskState(t, service, snapTask.ID, TaskStateCompleted)

	if len(*cmds) != 0 {
		t.Fatalf("non-ZFS space must not invoke zfs/zpool, got: %v", *cmds)
	}
}
