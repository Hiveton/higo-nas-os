package storage

import (
	"context"
	"strings"
	"testing"
)

func TestDefaultServiceUsesHostAdapter(t *testing.T) {
	service := NewService(nil)
	if _, ok := service.adapter.(*HostAdapter); !ok {
		t.Fatalf("default storage adapter = %T, want *HostAdapter", service.adapter)
	}
}

func TestHostAdapterBuildsPoolsAndDisksFromDF(t *testing.T) {
	output := []byte(`Filesystem     1024-blocks      Used Available Capacity Mounted on
/dev/disk3s3s1   239362496  12255772 123850648    10%    /
devfs                  217       217         0   100%    /dev
/dev/disk3s1     239362496  83595092 123850648    41%    /System/Volumes/Data
map auto_home            0         0         0   100%    /System/Volumes/Data/home
/dev/disk5s1        209920    192532     15960    93%    /Volumes/OpenClaw
`)
	adapter := NewHostAdapterWithRunner(func(context.Context, string, ...string) ([]byte, error) {
		return output, nil
	})
	ctx := context.Background()

	pools, err := adapter.Pools(ctx)
	if err != nil {
		t.Fatalf("pools: %v", err)
	}
	if len(pools) != 3 {
		t.Fatalf("expected 3 real host pools, got %d: %#v", len(pools), pools)
	}
	if pools[0].Name != "系统根卷" || pools[0].Type == "RAID 5" || pools[0].Total == "42 TB" || pools[0].UsedPercent != 10 {
		t.Fatalf("unexpected first pool: %#v", pools[0])
	}
	if pools[2].Health != HealthWarning {
		t.Fatalf("expected high-usage volume to be warning, got %#v", pools[2])
	}

	disks, err := adapter.Disks(ctx)
	if err != nil {
		t.Fatalf("disks: %v", err)
	}
	if len(disks) != len(pools) {
		t.Fatalf("expected one disk-like row per host volume, got %d disks for %d pools", len(disks), len(pools))
	}
	for _, disk := range disks {
		combined := disk.Serial + " " + disk.Model + " " + disk.Size
		if strings.Contains(combined, "HIGO") || strings.Contains(combined, "HiGo Iron") || strings.Contains(combined, "12 TB") {
			t.Fatalf("host disk still contains fixture data: %#v", disk)
		}
		if disk.Serial == "" || disk.PoolID == "" || disk.Interface != "mount" {
			t.Fatalf("host disk missing real source metadata: %#v", disk)
		}
	}
}
