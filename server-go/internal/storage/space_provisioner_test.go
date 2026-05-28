package storage

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestCommandSpaceProvisionerFormatsMountsAndPersistsFstab(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	fstab := root + "/fstab"
	var commands []string
	provisioner := &commandSpaceProvisioner{
		fstabPath: fstab,
		runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			commands = append(commands, strings.Join(append([]string{name}, args...), " "))
			if name == "/usr/sbin/blkid" {
				return []byte("uuid-123\n"), nil
			}
			return nil, nil
		},
	}

	err := provisioner.Provision(context.Background(), SpaceProvisionPlan{
		Name:       "家庭照片",
		Mode:       SpaceModeBasic,
		FileSystem: FileSystemBTRFS,
		FormatDisk: true,
		MountPath:  root + "/family",
		Disks: []Disk{{
			Slot:       "sdb",
			Size:       "15 GB",
			State:      DiskStateHealthy,
			Health:     HealthHealthy,
			Role:       "disk",
			DevicePath: "/dev/sdb",
			DeviceType: "disk",
		}},
	})
	if err != nil {
		t.Fatalf("provision: %v", err)
	}

	wantCommands := []string{
		"/usr/sbin/wipefs -a /dev/sdb",
		"/usr/sbin/mkfs.btrfs -f -L item /dev/sdb",
		"/usr/sbin/blkid -s UUID -o value /dev/sdb",
		"/usr/bin/mount " + root + "/family",
	}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", commands, wantCommands)
	}
	content, err := os.ReadFile(fstab)
	if err != nil {
		t.Fatalf("read fstab: %v", err)
	}
	if !strings.Contains(string(content), "UUID=uuid-123 "+root+"/family btrfs defaults,nofail 0 2") {
		t.Fatalf("unexpected fstab content: %q", string(content))
	}
}

func TestCommandSpaceProvisionerMountsExistingFilesystemWithoutFormatting(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	fstab := root + "/fstab"
	var commands []string
	provisioner := &commandSpaceProvisioner{
		fstabPath: fstab,
		runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			commands = append(commands, strings.Join(append([]string{name}, args...), " "))
			if name == "/usr/sbin/blkid" && len(args) >= 2 && args[1] == "UUID" {
				return []byte("existing-uuid\n"), nil
			}
			if name == "/usr/sbin/blkid" && len(args) >= 2 && args[1] == "TYPE" {
				return []byte("ext4\n"), nil
			}
			return nil, nil
		},
	}

	err := provisioner.Provision(context.Background(), SpaceProvisionPlan{
		Name:       "已有资料",
		Mode:       SpaceModeBasic,
		FileSystem: FileSystemBTRFS,
		FormatDisk: false,
		MountPath:  root + "/existing",
		Disks: []Disk{{
			Slot:       "sdc",
			Size:       "30 GB",
			State:      DiskStateHealthy,
			Health:     HealthHealthy,
			Role:       "disk",
			DevicePath: "/dev/sdc",
			DeviceType: "disk",
		}},
	})
	if err != nil {
		t.Fatalf("provision existing filesystem: %v", err)
	}

	for _, command := range commands {
		if strings.Contains(command, "wipefs") || strings.Contains(command, "mkfs") {
			t.Fatalf("expected no destructive formatting commands, got %#v", commands)
		}
	}
	wantCommands := []string{
		"/usr/sbin/blkid -s UUID -o value /dev/sdc",
		"/usr/sbin/blkid -s TYPE -o value /dev/sdc",
		"/usr/bin/mount " + root + "/existing",
	}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", commands, wantCommands)
	}
	content, err := os.ReadFile(fstab)
	if err != nil {
		t.Fatalf("read fstab: %v", err)
	}
	if !strings.Contains(string(content), "UUID=existing-uuid "+root+"/existing ext4 defaults,nofail 0 2") {
		t.Fatalf("unexpected fstab content: %q", string(content))
	}
}

func TestCommandSpaceProvisionerCreatesZFSPoolWithoutFstab(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	fstab := root + "/fstab"
	var commands []string
	provisioner := &commandSpaceProvisioner{
		fstabPath: fstab,
		runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			commands = append(commands, strings.Join(append([]string{name}, args...), " "))
			return nil, nil
		},
	}

	err := provisioner.Provision(context.Background(), SpaceProvisionPlan{
		Name:       "media",
		Mode:       SpaceModeRAID1,
		FileSystem: FileSystemZFS,
		FormatDisk: true,
		MountPath:  root + "/media",
		Disks: []Disk{
			{Slot: "sdb", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sdb", DeviceType: "disk"},
			{Slot: "sdc", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sdc", DeviceType: "disk"},
		},
	})
	if err != nil {
		t.Fatalf("provision zfs: %v", err)
	}

	wantCommands := []string{
		"/usr/sbin/zpool version",
		"/usr/sbin/wipefs -a /dev/sdb",
		"/usr/sbin/wipefs -a /dev/sdc",
		"/usr/sbin/zpool create -f -o ashift=12 -O compression=lz4 -O atime=off -m " + root + "/media higoos-media mirror /dev/sdb /dev/sdc",
	}
	if !reflect.DeepEqual(commands, wantCommands) {
		t.Fatalf("commands = %#v, want %#v", commands, wantCommands)
	}
	if _, err := os.Stat(fstab); !os.IsNotExist(err) {
		t.Fatalf("zfs provisioning should not write fstab, stat err=%v", err)
	}
}

func TestCommandSpaceProvisionerBuildsZFSRAID10Layout(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	var commands []string
	provisioner := &commandSpaceProvisioner{
		fstabPath: root + "/fstab",
		runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			commands = append(commands, strings.Join(append([]string{name}, args...), " "))
			return nil, nil
		},
	}

	err := provisioner.Provision(context.Background(), SpaceProvisionPlan{
		Name:       "team",
		Mode:       SpaceModeRAID10,
		FileSystem: FileSystemZFS,
		FormatDisk: true,
		MountPath:  root + "/team",
		Disks: []Disk{
			{Slot: "sdb", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sdb", DeviceType: "disk"},
			{Slot: "sdc", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sdc", DeviceType: "disk"},
			{Slot: "sdd", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sdd", DeviceType: "disk"},
			{Slot: "sde", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sde", DeviceType: "disk"},
		},
	})
	if err != nil {
		t.Fatalf("provision zfs raid10: %v", err)
	}
	last := commands[len(commands)-1]
	want := "/usr/sbin/zpool create -f -o ashift=12 -O compression=lz4 -O atime=off -m " + root + "/team higoos-team mirror /dev/sdb /dev/sdc mirror /dev/sdd /dev/sde"
	if last != want {
		t.Fatalf("zfs raid10 command = %q, want %q", last, want)
	}
}

func TestCommandSpaceProvisionerChecksZFSBeforeWiping(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	var commands []string
	provisioner := &commandSpaceProvisioner{
		fstabPath: root + "/fstab",
		runner: func(_ context.Context, name string, args ...string) ([]byte, error) {
			commands = append(commands, strings.Join(append([]string{name}, args...), " "))
			if name == "/usr/sbin/zpool" && len(args) == 1 && args[0] == "version" {
				return nil, errors.New("missing")
			}
			return nil, nil
		},
	}

	err := provisioner.Provision(context.Background(), SpaceProvisionPlan{
		Name:       "media",
		Mode:       SpaceModeBasic,
		FileSystem: FileSystemZFS,
		FormatDisk: true,
		MountPath:  root + "/media",
		Disks:      []Disk{{Slot: "sdb", Size: "15 GB", State: DiskStateHealthy, Health: HealthHealthy, Role: "disk", DevicePath: "/dev/sdb", DeviceType: "disk"}},
	})
	if err == nil || !strings.Contains(err.Error(), "zfsutils-linux") {
		t.Fatalf("expected missing zfs tools error, got %v", err)
	}
	if !reflect.DeepEqual(commands, []string{"/usr/sbin/zpool version"}) {
		t.Fatalf("expected zfs check before destructive commands, got %#v", commands)
	}
}
