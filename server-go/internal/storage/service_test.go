package storage

import (
	"context"
	"os"
	"strings"
	"testing"
)

type staticAdapter struct {
	pools []StoragePool
	disks []Disk
}

func (a staticAdapter) Pools(context.Context) ([]StoragePool, error) {
	return clonePools(a.pools), nil
}

func (a staticAdapter) Disks(context.Context) ([]Disk, error) {
	return cloneDisks(a.disks), nil
}

func (a staticAdapter) SmartReports(context.Context) ([]SmartReport, error) {
	return nil, nil
}

type recordingProvisioner struct {
	plans []SpaceProvisionPlan
	err   error
}

func (p *recordingProvisioner) Provision(_ context.Context, plan SpaceProvisionPlan) error {
	p.plans = append(p.plans, plan)
	return p.err
}

func boolPtr(value bool) *bool {
	return &value
}

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

func TestAddDiskRegistersManagedMount(t *testing.T) {
	service := NewService(NewDevAdapter())
	mountPath := t.TempDir()

	disk, err := service.AddDisk(context.Background(), AddDiskRequest{
		Name:      "Data Disk",
		MountPath: mountPath,
		Role:      "data",
	})
	if err != nil {
		t.Fatalf("add disk: %v", err)
	}
	if disk.MountPath != mountPath || disk.Model != "Data Disk" || disk.PoolID == "" || disk.Size == "" || disk.Size == "待探测" {
		t.Fatalf("unexpected disk: %#v", disk)
	}
	disks, err := service.Disks(context.Background())
	if err != nil {
		t.Fatalf("list disks: %v", err)
	}
	if disks[len(disks)-1].MountPath != mountPath {
		t.Fatalf("registered disk missing from list: %#v", disks)
	}
	pools, err := service.Pools(context.Background())
	if err != nil {
		t.Fatalf("list pools: %v", err)
	}
	if pools[len(pools)-1].MountPath != mountPath {
		t.Fatalf("registered mount missing from pools: %#v", pools)
	}
}

func TestCreateAndDeleteStorageSpace(t *testing.T) {
	provisioner := &recordingProvisioner{}
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	service := NewService(staticAdapter{disks: []Disk{{
		Slot:       "data",
		Size:       "245 GB",
		State:      DiskStateHealthy,
		Health:     HealthHealthy,
		Role:       "disk",
		DevicePath: "/dev/test-data",
		DeviceType: "disk",
		MediaType:  "SSD",
	}}})
	service.provisioner = provisioner
	ctx := context.Background()

	mountPath := root + "/family-photos"
	space, err := service.CreateSpace(ctx, CreateSpaceRequest{
		Name:       "家庭照片",
		Mode:       SpaceModeBasic,
		FileSystem: FileSystemEXT4,
		DiskSlots:  []string{"data"},
		MountPath:  mountPath,
		Actor:      "admin",
		Confirm:    true,
		FormatDisk: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("create storage space: %v", err)
	}
	if space.ID == "" || space.Name != "家庭照片" || space.Mode != SpaceModeBasic || space.FileSystem != FileSystemEXT4 || len(space.DiskSlots) != 1 || space.Total != "245 GB" {
		t.Fatalf("unexpected storage space: %#v", space)
	}
	if len(provisioner.plans) != 1 {
		t.Fatalf("expected provisioner to be called once, got %d", len(provisioner.plans))
	}
	if plan := provisioner.plans[0]; plan.FileSystem != FileSystemEXT4 || plan.Mode != SpaceModeBasic || plan.MountPath != mountPath || plan.Disks[0].DevicePath != "/dev/test-data" {
		t.Fatalf("unexpected provision plan: %#v", plan)
	}
	if info, err := os.Stat(mountPath); err != nil || !info.IsDir() {
		t.Fatalf("expected mount path directory to be created, info=%#v err=%v", info, err)
	}

	spaces, err := service.Spaces(ctx)
	if err != nil {
		t.Fatalf("list storage spaces: %v", err)
	}
	if spaces[len(spaces)-1].ID != space.ID {
		t.Fatalf("created space missing from list: %#v", spaces)
	}

	task, err := service.DeleteSpace(ctx, space.ID, DeleteSpaceRequest{Actor: "admin", Confirm: true})
	if err != nil {
		t.Fatalf("delete storage space: %v", err)
	}
	if task.Kind != TaskKindDeleteSpace || task.TargetPool != space.ID {
		t.Fatalf("unexpected delete task: %#v", task)
	}
	spaces, _ = service.Spaces(ctx)
	for _, candidate := range spaces {
		if candidate.ID == space.ID {
			t.Fatalf("deleted space still listed: %#v", candidate)
		}
	}
}

func TestCreateSpaceRequiresExplicitCreateConfirmation(t *testing.T) {
	service := NewService(NewDevAdapter())

	if _, err := service.CreateSpace(context.Background(), CreateSpaceRequest{
		Name:       "未确认空间",
		Mode:       SpaceModeBasic,
		FileSystem: FileSystemEXT4,
		DiskSlots:  []string{"1"},
	}); err == nil {
		t.Fatal("expected create space to require confirmation")
	}
}

func TestCreateSpaceCalculatesRAIDCapacityAndValidatesMode(t *testing.T) {
	provisioner := &recordingProvisioner{}
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	service := NewService(staticAdapter{disks: []Disk{
		{Slot: "1", Size: "245 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/test-1", DeviceType: "disk"},
		{Slot: "2", Size: "245 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/test-2", DeviceType: "disk"},
	}})
	service.provisioner = provisioner

	space, err := service.CreateSpace(context.Background(), CreateSpaceRequest{
		Name:       "资料镜像",
		Mode:       SpaceModeBasic,
		FileSystem: FileSystemBTRFS,
		DiskSlots:  []string{"1"},
		MountPath:  root + "/mirror",
		Confirm:    true,
		FormatDisk: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("create basic space: %v", err)
	}
	if space.Total != "245 GB" || space.Health != HealthHealthy {
		t.Fatalf("unexpected capacity or health: %#v", space)
	}

	if _, err := service.CreateSpace(context.Background(), CreateSpaceRequest{
		Name:       "错误阵列",
		Mode:       SpaceModeRAID5,
		FileSystem: FileSystemBTRFS,
		DiskSlots:  []string{"1", "2"},
		MountPath:  root + "/bad",
		Confirm:    true,
		FormatDisk: boolPtr(true),
	}); err == nil {
		t.Fatal("expected raid5 creation to require three disks")
	}
}

func TestCreateSpaceRejectsUnsafeRealDiskTargets(t *testing.T) {
	tests := []struct {
		name    string
		disk    Disk
		wantErr string
	}{
		{
			name:    "system disk",
			disk:    Disk{Slot: "sda", Size: "64 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sda", DeviceType: "disk", SystemDisk: true},
			wantErr: "system disk",
		},
		{
			name:    "mounted disk",
			disk:    Disk{Slot: "sdb", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sdb", DeviceType: "disk", MountPath: "/mnt/data"},
			wantErr: "mounted",
		},
		{
			name:    "missing device path",
			disk:    Disk{Slot: "managed", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "data"},
			wantErr: "block device",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provisioner := &recordingProvisioner{}
			root := t.TempDir()
			t.Setenv("HIGO_NAS_ROOT", root)
			service := NewService(staticAdapter{disks: []Disk{tt.disk}})
			service.provisioner = provisioner
			_, err := service.CreateSpace(context.Background(), CreateSpaceRequest{
				Name:       "危险空间",
				Mode:       SpaceModeBasic,
				FileSystem: FileSystemEXT4,
				DiskSlots:  []string{tt.disk.Slot},
				MountPath:  root + "/space",
				Confirm:    true,
				FormatDisk: boolPtr(true),
			})
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected %q error, got %v", tt.wantErr, err)
			}
			if len(provisioner.plans) != 0 {
				t.Fatalf("provisioner should not run for unsafe disk: %#v", provisioner.plans)
			}
		})
	}
}

func TestManagedDiskRemovalAndSettings(t *testing.T) {
	service := NewService(NewDevAdapter())
	mountPath := t.TempDir()
	disk, err := service.AddDisk(context.Background(), AddDiskRequest{Name: "Cache Disk", MountPath: mountPath, Role: "cache"})
	if err != nil {
		t.Fatalf("add disk: %v", err)
	}

	updated, err := service.UpdateDiskSettings(context.Background(), disk.Slot, DiskSettingsRequest{
		StandbyMinutes: 20,
		SSDCache:       true,
		CacheMode:      "read-write",
		Actor:          "admin",
	})
	if err != nil {
		t.Fatalf("update disk settings: %v", err)
	}
	if updated.StandbyMinutes != 20 || !updated.SSDCache || updated.CacheMode != "read-write" {
		t.Fatalf("unexpected updated disk settings: %#v", updated)
	}

	task, err := service.RemoveDisk(context.Background(), disk.Slot, RemoveDiskRequest{Actor: "admin", Confirm: true})
	if err != nil {
		t.Fatalf("remove disk: %v", err)
	}
	if task.Kind != TaskKindRemoveDisk || task.TargetSlot != disk.Slot {
		t.Fatalf("unexpected remove disk task: %#v", task)
	}
	disks, _ := service.Disks(context.Background())
	for _, candidate := range disks {
		if candidate.Slot == disk.Slot {
			t.Fatalf("removed managed disk still listed: %#v", candidate)
		}
	}
}
