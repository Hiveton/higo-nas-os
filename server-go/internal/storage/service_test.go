package storage

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"higoos/server-go/internal/tasks"
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
	plans     []SpaceProvisionPlan
	teardowns []SpaceProvisionPlan
	err       error
}

func (p *recordingProvisioner) Provision(_ context.Context, plan SpaceProvisionPlan) error {
	p.plans = append(p.plans, plan)
	return p.err
}

func (p *recordingProvisioner) Deprovision(_ context.Context, plan SpaceProvisionPlan) error {
	p.teardowns = append(p.teardowns, plan)
	return nil
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

func TestScanTasksAreExecutedToCompletionByRunner(t *testing.T) {
	service := NewService(NewDevAdapter())
	mgr, err := tasks.NewManager("", tasks.WithWorkers(2), tasks.WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("new task manager: %v", err)
	}
	service.AttachTaskRunner(mgr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	cases := []struct {
		name  string
		start func() (StorageTask, error)
	}{
		{"smart", func() (StorageTask, error) { return service.StartSMARTScan(ctx, TaskTarget{TargetSlot: "1"}) }},
		{"repair", func() (StorageTask, error) { return service.StartRepair(ctx, TaskTarget{TargetPool: "pool-raid5"}) }},
		{"snapshot", func() (StorageTask, error) { return service.CreateSnapshot(ctx, TaskTarget{TargetPool: "pool-backup"}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			task, err := tc.start()
			if err != nil {
				t.Fatalf("start: %v", err)
			}
			deadline := time.Now().Add(2 * time.Second)
			var final StorageTask
			for time.Now().Before(deadline) {
				final, err = service.GetTask(ctx, task.ID)
				if err != nil {
					t.Fatalf("get task: %v", err)
				}
				if final.State == TaskStateCompleted || final.State == TaskStateFailed {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			if final.State != TaskStateCompleted {
				t.Fatalf("expected task completed, got state=%q message=%q", final.State, final.Message)
			}
			if final.Progress != 100 {
				t.Fatalf("expected progress 100, got %d", final.Progress)
			}
		})
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

func TestValidateSpaceModeMatrix(t *testing.T) {
	cases := []struct {
		mode    SpaceMode
		count   int
		wantErr bool
	}{
		{SpaceModeBasic, 1, false},
		{SpaceModeBasic, 2, true},   // basic is single-disk only
		{SpaceModeLinear, 1, true},  // linear needs 2+ disks to concatenate
		{SpaceModeLinear, 2, false},
		{SpaceModeRAID0, 2, false},
		{SpaceModeRAID1, 1, true},
		{SpaceModeRAID5, 2, true},
		{SpaceModeRAID5, 3, false},
		{SpaceModeRAID6, 3, true},
		{SpaceModeRAID6, 4, false},
		{SpaceModeRAID10, 3, true}, // odd count rejected
		{SpaceModeRAID10, 4, false},
	}
	for _, c := range cases {
		err := validateSpaceMode(c.mode, c.count)
		if (err != nil) != c.wantErr {
			t.Fatalf("validateSpaceMode(%s,%d): err=%v wantErr=%v", c.mode, c.count, err, c.wantErr)
		}
	}
}

func TestDeleteSpaceTwoPhaseGovernance(t *testing.T) {
	provisioner := &recordingProvisioner{}
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	service := NewService(staticAdapter{disks: []Disk{{
		Slot: "data", Size: "245 GB", State: DiskStateHealthy, Health: HealthHealthy,
		Role: "disk", DevicePath: "/dev/test-data", DeviceType: "disk", MediaType: "SSD",
	}}})
	service.provisioner = provisioner
	ctx := context.Background()

	space, err := service.CreateSpace(ctx, CreateSpaceRequest{
		Name: "待删空间", Mode: SpaceModeBasic, FileSystem: FileSystemEXT4,
		DiskSlots: []string{"data"}, MountPath: root + "/del", Actor: "admin",
		Confirm: true, FormatDisk: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("create space: %v", err)
	}

	// Phase 1: preview registers a single-use token and does NOT remove the space.
	preview, err := service.PreviewDeleteSpace(ctx, space.ID, "admin")
	if err != nil {
		t.Fatalf("preview delete: %v", err)
	}
	if preview.ConfirmationID == "" || preview.Impact == "" || !preview.RequiresConfirmation {
		t.Fatalf("unexpected preview: %#v", preview)
	}
	if preview.Risk != "high" {
		t.Fatalf("expected high risk, got %q", preview.Risk)
	}
	if spaces, _ := service.Spaces(ctx); !containsSpace(spaces, space.ID) {
		t.Fatal("preview must not delete the space")
	}

	// A bad confirmation id is rejected.
	if _, err := service.ConfirmDeleteSpace(ctx, ConfirmDeleteRequest{ConfirmationID: "storage-confirm-999", Actor: "admin"}); err == nil {
		t.Fatal("expected error for unknown confirmation id")
	}

	// Phase 2: confirm executes once.
	task, err := service.ConfirmDeleteSpace(ctx, ConfirmDeleteRequest{ConfirmationID: preview.ConfirmationID, Actor: "admin"})
	if err != nil {
		t.Fatalf("confirm delete: %v", err)
	}
	if task.Kind != TaskKindDeleteSpace || task.TargetPool != space.ID {
		t.Fatalf("unexpected delete task: %#v", task)
	}
	if spaces, _ := service.Spaces(ctx); containsSpace(spaces, space.ID) {
		t.Fatal("space should be gone after confirm")
	}

	// Single-use: the same token cannot be replayed.
	if _, err := service.ConfirmDeleteSpace(ctx, ConfirmDeleteRequest{ConfirmationID: preview.ConfirmationID, Actor: "admin"}); err == nil {
		t.Fatal("expected single-use token to be consumed")
	}

	// One audit entry recorded for the confirmed deletion.
	auditTrail := service.Audit()
	if len(auditTrail) != 1 || auditTrail[0].SpaceID != space.ID || auditTrail[0].Result != "confirmed" {
		t.Fatalf("unexpected audit trail: %#v", auditTrail)
	}

	// Confirm tore down the underlying storage (so the disk isn't orphaned).
	if len(provisioner.teardowns) != 1 || provisioner.teardowns[0].MountPath != space.MountPath {
		t.Fatalf("expected one teardown for %s, got %#v", space.MountPath, provisioner.teardowns)
	}
}

func TestCommandProvisionerDeprovisionRemovesFstabEntry(t *testing.T) {
	root := t.TempDir()
	fstab := root + "/fstab"
	keep := "UUID=other /srv/higoos/nas/keep btrfs defaults,nofail 0 2"
	drop := "UUID=gone /srv/higoos/nas/9 btrfs defaults,nofail 0 2"
	if err := os.WriteFile(fstab, []byte(keep+"\n"+drop+"\n"), 0o644); err != nil {
		t.Fatalf("seed fstab: %v", err)
	}
	var commands []string
	provisioner := &commandSpaceProvisioner{
		fstabPath: fstab,
		runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			commands = append(commands, name)
			return nil, nil // findmnt returns empty -> not mounted -> no umount
		},
	}
	if err := provisioner.Deprovision(context.Background(), SpaceProvisionPlan{
		FileSystem: FileSystemBTRFS,
		MountPath:  "/srv/higoos/nas/9",
	}); err != nil {
		t.Fatalf("deprovision: %v", err)
	}
	out, _ := os.ReadFile(fstab)
	if strings.Contains(string(out), "/srv/higoos/nas/9") {
		t.Fatalf("fstab still contains removed mount: %q", out)
	}
	if !strings.Contains(string(out), "/srv/higoos/nas/keep") {
		t.Fatalf("fstab lost the unrelated entry: %q", out)
	}
}

func TestConfirmDeleteSpaceRejectsExpiredToken(t *testing.T) {
	provisioner := &recordingProvisioner{}
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	service := NewService(staticAdapter{disks: []Disk{{
		Slot: "data", Size: "245 GB", State: DiskStateHealthy, Health: HealthHealthy,
		Role: "disk", DevicePath: "/dev/test-data", DeviceType: "disk",
	}}})
	service.provisioner = provisioner
	clock := time.Unix(1_700_000_000, 0).UTC()
	service.now = func() time.Time { return clock }
	ctx := context.Background()

	space, err := service.CreateSpace(ctx, CreateSpaceRequest{
		Name: "过期空间", Mode: SpaceModeBasic, FileSystem: FileSystemEXT4,
		DiskSlots: []string{"data"}, MountPath: root + "/exp", Actor: "admin",
		Confirm: true, FormatDisk: boolPtr(true),
	})
	if err != nil {
		t.Fatalf("create space: %v", err)
	}
	preview, err := service.PreviewDeleteSpace(ctx, space.ID, "admin")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	// Advance the clock past the TTL.
	clock = clock.Add(confirmationTTL + time.Minute)
	if _, err := service.ConfirmDeleteSpace(ctx, ConfirmDeleteRequest{ConfirmationID: preview.ConfirmationID}); err == nil {
		t.Fatal("expected expired confirmation to be rejected")
	}
	if spaces, _ := service.Spaces(ctx); !containsSpace(spaces, space.ID) {
		t.Fatal("expired confirm must not delete the space")
	}
}

func containsSpace(spaces []StorageSpace, id string) bool {
	for _, s := range spaces {
		if s.ID == id {
			return true
		}
	}
	return false
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
