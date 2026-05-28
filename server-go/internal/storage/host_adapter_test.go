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

func TestHostAdapterBuildsDisksFromLSBLK(t *testing.T) {
	lsblk := []byte(`{
  "blockdevices": [
    {"name":"sda","path":"/dev/sda","type":"disk","size":64424509440,"rota":false,"tran":null,"model":"Virtual disk","serial":null,"mountpoints":[null],"fstype":null,"state":"running","children":[
      {"name":"sda1","path":"/dev/sda1","type":"part","size":1127219200,"fsused":94371840,"fssize":1127219200,"rota":false,"tran":null,"model":null,"serial":null,"mountpoints":["/boot/efi"],"fstype":"vfat","state":null},
      {"name":"sda2","path":"/dev/sda2","type":"part","size":2147483648,"fsused":1073741824,"fssize":2147483648,"rota":false,"tran":null,"model":null,"serial":null,"mountpoints":["/boot"],"fstype":"ext4","state":null}
    ]},
    {"name":"sdb","path":"/dev/sdb","type":"disk","size":15032385536,"rota":false,"tran":"sata","model":"Samsung SSD","serial":"S123","mountpoints":[null],"fstype":null,"state":"running"}
  ]
}`)
	adapter := NewHostAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "lsblk" {
			t.Fatalf("unexpected command %s %v", name, args)
		}
		if !strings.Contains(strings.Join(args, " "), "FSUSED,FSSIZE") {
			t.Fatalf("lsblk fields missing partition usage columns: %v", args)
		}
		return lsblk, nil
	})

	disks, err := adapter.Disks(context.Background())
	if err != nil {
		t.Fatalf("disks: %v", err)
	}
	if len(disks) != 2 {
		t.Fatalf("expected 2 physical disks, got %d: %#v", len(disks), disks)
	}
	if disks[0].Slot != "sda" || disks[0].DevicePath != "/dev/sda" || disks[0].MediaType != "虚拟 SSD" || disks[0].Interface != "block" || disks[0].MountPath != "/boot/efi" || !disks[0].SystemDisk {
		t.Fatalf("unexpected first lsblk disk: %#v", disks[0])
	}
	if len(disks[0].Partitions) != 2 {
		t.Fatalf("expected first disk partitions, got %#v", disks[0].Partitions)
	}
	if disks[0].Partitions[0].Name != "sda1" || disks[0].Partitions[0].FileSystem != "vfat" || !disks[0].Partitions[0].System || disks[0].Partitions[0].Used == "" || disks[0].Partitions[0].Total == "" {
		t.Fatalf("unexpected first partition: %#v", disks[0].Partitions[0])
	}
	if disks[1].Slot != "sdb" || disks[1].MediaType != "NVMe SSD" && disks[1].MediaType != "SSD" || disks[1].Interface != "sata" || disks[1].Serial != "S123" || disks[1].SystemDisk {
		t.Fatalf("unexpected second lsblk disk: %#v", disks[1])
	}
}
