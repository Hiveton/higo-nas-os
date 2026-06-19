package storage

import (
	"context"
	"testing"
)

func TestPoolsShowLiveCapacityForNonZFSSpace(t *testing.T) {
	service := NewService(NewDevAdapter())
	mount := t.TempDir() // a real, statfs-able directory
	service.mu.Lock()
	service.spaces = []StorageSpace{{
		ID: "space-1", Name: "家庭照片", Mode: SpaceModeBasic, FileSystem: FileSystemBTRFS,
		MountPath: mount, UsedPercent: 99, Total: "估算值", Health: HealthHealthy,
	}}
	service.mu.Unlock()

	pools, err := service.Pools(context.Background())
	if err != nil {
		t.Fatalf("pools: %v", err)
	}
	var p *StoragePool
	for i := range pools {
		if pools[i].ID == "space-1" {
			p = &pools[i]
		}
	}
	if p == nil {
		t.Fatalf("space not found in pools: %#v", pools)
	}
	if p.UsedPercent < 0 || p.UsedPercent > 100 {
		t.Fatalf("live used%% out of range: %d", p.UsedPercent)
	}
	if p.Total == "估算值" {
		t.Fatalf("expected live total to replace the estimate, got %q", p.Total)
	}
}

func TestPoolsFallBackToEstimateForInvalidMount(t *testing.T) {
	service := NewService(NewDevAdapter())
	service.mu.Lock()
	service.spaces = []StorageSpace{{
		ID: "space-1", Name: "家庭照片", FileSystem: FileSystemEXT4,
		MountPath: "/path/does/not/exist", UsedPercent: 42, Total: "8 TB", Health: HealthHealthy,
	}}
	service.mu.Unlock()

	pools, _ := service.Pools(context.Background())
	for _, p := range pools {
		if p.ID == "space-1" {
			if p.UsedPercent != 42 || p.Total != "8 TB" {
				t.Fatalf("expected stored estimate on invalid mount, got %d / %q", p.UsedPercent, p.Total)
			}
			return
		}
	}
	t.Fatal("space not found")
}
