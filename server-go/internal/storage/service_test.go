package storage

import (
	"context"
	"strings"
	"testing"
)

func TestDevServiceReturnsFrontendAlignedPoolsAndDisks(t *testing.T) {
	service := NewService(NewDevAdapter())
	ctx := context.Background()

	pools, err := service.Pools(ctx)
	if err != nil {
		t.Fatalf("pools: %v", err)
	}
	if len(pools) != 3 {
		t.Fatalf("expected 3 storage pools, got %d", len(pools))
	}
	wantPools := []struct {
		name   string
		kind   string
		used   int
		total  string
		health Health
		temp   string
	}{
		{name: "开发主机根卷", kind: "主机卷", used: 10, total: "245 GB", health: HealthHealthy, temp: "N/A"},
		{name: "开发主机数据卷", kind: "主机卷", used: 41, total: "245 GB", health: HealthHealthy, temp: "N/A"},
		{name: "开发主机外接卷", kind: "主机卷", used: 93, total: "215 MB", health: HealthWarning, temp: "N/A"},
	}
	for index, want := range wantPools {
		got := pools[index]
		if got.Name != want.name || got.Type != want.kind || got.UsedPercent != want.used || got.Total != want.total || got.Health != want.health || got.Temperature != want.temp {
			t.Fatalf("pool %d = %#v, want %#v", index, got, want)
		}
	}

	disks, err := service.Disks(ctx)
	if err != nil {
		t.Fatalf("disks: %v", err)
	}
	if len(disks) != 3 {
		t.Fatalf("expected 3 disks, got %d", len(disks))
	}
	if disks[0].Slot != "1" || disks[0].Size != "245 GB" || disks[0].State != DiskStateHealthy || disks[0].Temperature != "N/A" || disks[0].PoolID != "host-dev-root" {
		t.Fatalf("unexpected first disk: %#v", disks[0])
	}
	if disks[2].Slot != "3" || disks[2].Health != HealthWarning || disks[2].Role != "volume" {
		t.Fatalf("unexpected external disk: %#v", disks[2])
	}
}

func TestSmartReportsCoverEveryDiskWithHealthAndAttributes(t *testing.T) {
	service := NewService(NewDevAdapter())

	reports, err := service.SmartReports(context.Background())
	if err != nil {
		t.Fatalf("smart reports: %v", err)
	}
	if len(reports) != 3 {
		t.Fatalf("expected 3 SMART reports, got %d", len(reports))
	}
	for _, report := range reports {
		if report.DiskSlot == "" || report.Serial == "" {
			t.Fatalf("report missing disk identity: %#v", report)
		}
		if report.Health != HealthHealthy && report.Health != HealthWarning {
			t.Fatalf("expected known SMART health, got %#v", report)
		}
		if len(report.Attributes) == 0 {
			t.Fatalf("expected SMART attributes for slot %s", report.DiskSlot)
		}
	}
}

func TestStorageTasksAreCreatedAndCanBeFetched(t *testing.T) {
	service := NewService(NewDevAdapter())
	ctx := context.Background()

	smartTask, err := service.StartSMARTScan(ctx, TaskTarget{TargetSlot: "1"})
	if err != nil {
		t.Fatalf("start smart scan: %v", err)
	}
	repairTask, err := service.StartRepair(ctx, TaskTarget{TargetPool: "pool-raid5"})
	if err != nil {
		t.Fatalf("start repair: %v", err)
	}
	snapshotTask, err := service.CreateSnapshot(ctx, TaskTarget{TargetPool: "pool-backup"})
	if err != nil {
		t.Fatalf("create snapshot: %v", err)
	}

	tasks := []struct {
		got        StorageTask
		wantKind   TaskKind
		wantSlot   string
		wantPool   string
		wantPrefix string
	}{
		{got: smartTask, wantKind: TaskKindSMARTScan, wantSlot: "1", wantPrefix: "smart-"},
		{got: repairTask, wantKind: TaskKindRepair, wantPool: "pool-raid5", wantPrefix: "repair-"},
		{got: snapshotTask, wantKind: TaskKindSnapshot, wantPool: "pool-backup", wantPrefix: "snapshot-"},
	}
	for _, tt := range tasks {
		if tt.got.ID == "" || tt.got.Kind != tt.wantKind || tt.got.State != TaskStateQueued || tt.got.Progress != 0 || tt.got.Message == "" || tt.got.CreatedAt.IsZero() {
			t.Fatalf("task missing generated fields: %#v", tt.got)
		}
		if !strings.HasPrefix(tt.got.ID, tt.wantPrefix) {
			t.Fatalf("task id %q does not have prefix %q", tt.got.ID, tt.wantPrefix)
		}
		if tt.got.TargetSlot != tt.wantSlot || tt.got.TargetPool != tt.wantPool {
			t.Fatalf("task target = slot %q pool %q, want slot %q pool %q", tt.got.TargetSlot, tt.got.TargetPool, tt.wantSlot, tt.wantPool)
		}

		fetched, err := service.GetTask(ctx, tt.got.ID)
		if err != nil {
			t.Fatalf("get task %s: %v", tt.got.ID, err)
		}
		if fetched.ID != tt.got.ID || fetched.Kind != tt.wantKind {
			t.Fatalf("fetched task mismatch: got %#v want %#v", fetched, tt.got)
		}
	}
}

func TestGetTaskRejectsUnknownID(t *testing.T) {
	service := NewService(NewDevAdapter())

	if _, err := service.GetTask(context.Background(), "missing-task"); err == nil {
		t.Fatal("expected unknown task error")
	}
}
