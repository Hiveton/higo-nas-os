package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type SpaceProvisionPlan struct {
	Name       string
	Mode       SpaceMode
	FileSystem FileSystem
	FormatDisk bool
	Disks      []Disk
	MountPath  string
}

type SpaceProvisioner interface {
	Provision(context.Context, SpaceProvisionPlan) error
	// Deprovision releases a space's storage (unmount + drop fstab entry, or
	// destroy the ZFS pool) so its disks become reusable. It must be safe to
	// call when nothing is mounted (idempotent teardown).
	Deprovision(context.Context, SpaceProvisionPlan) error
}

type commandSpaceProvisioner struct {
	runner    commandRunner
	fstabPath string
}

func NewCommandSpaceProvisioner() SpaceProvisioner {
	return &commandSpaceProvisioner{
		runner:    runCommand,
		fstabPath: "/etc/fstab",
	}
}

func (p *commandSpaceProvisioner) Provision(ctx context.Context, plan SpaceProvisionPlan) error {
	if err := validateProvisionPlan(plan); err != nil {
		return err
	}
	if err := os.MkdirAll(plan.MountPath, 0o755); err != nil {
		return fmt.Errorf("create mount path: %w", err)
	}
	if plan.FileSystem == FileSystemZFS {
		return p.provisionZFS(ctx, plan)
	}
	disk := plan.Disks[0]
	device := disk.DevicePath
	mountFS := string(plan.FileSystem)
	if plan.FormatDisk {
		if _, err := p.runner(ctx, "/usr/sbin/wipefs", "-a", device); err != nil {
			return fmt.Errorf("wipe disk signatures on %s: %w", device, err)
		}
		switch plan.FileSystem {
		case FileSystemEXT4:
			if _, err := p.runner(ctx, "/usr/sbin/mkfs.ext4", "-F", "-L", filesystemLabel(plan.Name), device); err != nil {
				return fmt.Errorf("format %s as ext4: %w", device, err)
			}
		case FileSystemBTRFS:
			if _, err := p.runner(ctx, "/usr/sbin/mkfs.btrfs", "-f", "-L", filesystemLabel(plan.Name), device); err != nil {
				return fmt.Errorf("format %s as btrfs: %w", device, err)
			}
		default:
			return fmt.Errorf("real formatting currently supports ext4 and btrfs only")
		}
	}
	uuidOutput, err := p.runner(ctx, "/usr/sbin/blkid", "-s", "UUID", "-o", "value", device)
	if err != nil {
		return fmt.Errorf("read filesystem uuid for %s: %w", device, err)
	}
	uuid := strings.TrimSpace(string(uuidOutput))
	if uuid == "" {
		return fmt.Errorf("filesystem uuid is empty for %s; choose format disk if this is a new disk", device)
	}
	if !plan.FormatDisk {
		typeOutput, err := p.runner(ctx, "/usr/sbin/blkid", "-s", "TYPE", "-o", "value", device)
		if err != nil {
			return fmt.Errorf("read filesystem type for %s: %w", device, err)
		}
		mountFS = strings.TrimSpace(string(typeOutput))
		if mountFS == "" {
			return fmt.Errorf("filesystem type is empty for %s; choose format disk if this is a new disk", device)
		}
	}
	if err := p.ensureFstabEntry(uuid, plan.MountPath, mountFS); err != nil {
		return err
	}
	if _, err := p.runner(ctx, "/usr/bin/mount", plan.MountPath); err != nil {
		return fmt.Errorf("mount %s at %s: %w", device, plan.MountPath, err)
	}
	return nil
}

func (p *commandSpaceProvisioner) provisionZFS(ctx context.Context, plan SpaceProvisionPlan) error {
	if !plan.FormatDisk {
		return fmt.Errorf("mounting an existing zfs pool is not supported by this create flow")
	}
	if _, err := p.runner(ctx, "/usr/sbin/zpool", "version"); err != nil {
		return fmt.Errorf("zfs tool zpool is not installed; install zfsutils-linux first: %w", err)
	}
	for _, disk := range plan.Disks {
		if _, err := p.runner(ctx, "/usr/sbin/wipefs", "-a", disk.DevicePath); err != nil {
			return fmt.Errorf("wipe disk signatures on %s: %w", disk.DevicePath, err)
		}
	}
	args := []string{
		"create",
		"-f",
		"-o", "ashift=12",
		"-O", "compression=lz4",
		"-O", "atime=off",
		"-m", plan.MountPath,
		zfsPoolName(plan.Name),
	}
	args = append(args, zfsVdevLayout(plan.Mode, plan.Disks)...)
	if _, err := p.runner(ctx, "/usr/sbin/zpool", args...); err != nil {
		return fmt.Errorf("create zfs pool for %s: %w", plan.MountPath, err)
	}
	return nil
}

// Deprovision tears down a space's storage so its disks can be reused. For
// ext4/btrfs it unmounts (when mounted) and drops the fstab entry; for ZFS it
// destroys the pool. Missing tooling (dev hosts without the storage stack) is
// tolerated so the teardown is a no-op there rather than a hard failure.
func (p *commandSpaceProvisioner) Deprovision(ctx context.Context, plan SpaceProvisionPlan) error {
	if plan.FileSystem == FileSystemZFS {
		if _, err := p.runner(ctx, "/usr/sbin/zpool", "destroy", "-f", zfsPoolName(plan.Name)); err != nil && !isMissingBinary(err) {
			// A non-existent pool is already torn down; only real failures matter.
			if exists, _ := p.runner(ctx, "/usr/sbin/zpool", "list", "-H", zfsPoolName(plan.Name)); len(bytes.TrimSpace(exists)) > 0 {
				return fmt.Errorf("destroy zfs pool %s: %w", plan.Name, err)
			}
		}
		return p.removeFstabEntry(plan.MountPath)
	}
	if strings.TrimSpace(plan.MountPath) == "" {
		return nil
	}
	// Only attempt umount when the path is actually a mountpoint (findmnt prints
	// the target on a hit, nothing on a miss / when the tool is absent).
	if target, _ := p.runner(ctx, "/usr/bin/findmnt", "-rno", "TARGET", plan.MountPath); len(bytes.TrimSpace(target)) > 0 {
		if _, err := p.runner(ctx, "/usr/bin/umount", plan.MountPath); err != nil && !isMissingBinary(err) {
			return fmt.Errorf("unmount %s: %w", plan.MountPath, err)
		}
	}
	return p.removeFstabEntry(plan.MountPath)
}

// removeFstabEntry drops any /etc/fstab line whose mount point matches mountPath.
func (p *commandSpaceProvisioner) removeFstabEntry(mountPath string) error {
	if strings.TrimSpace(mountPath) == "" {
		return nil
	}
	content, err := os.ReadFile(p.fstabPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read fstab: %w", err)
	}
	target := escapeFstabField(mountPath)
	lines := strings.Split(string(content), "\n")
	kept := make([]string, 0, len(lines))
	removed := false
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == target {
			removed = true
			continue
		}
		kept = append(kept, line)
	}
	if !removed {
		return nil
	}
	return os.WriteFile(p.fstabPath, []byte(strings.Join(kept, "\n")), 0o644)
}

// isMissingBinary reports whether err is "executable not found", letting dev
// hosts without the storage CLIs treat teardown steps as no-ops.
func isMissingBinary(err error) bool {
	return errors.Is(err, exec.ErrNotFound)
}

func validateProvisionPlan(plan SpaceProvisionPlan) error {
	if plan.FileSystem != FileSystemZFS {
		if plan.Mode != SpaceModeBasic || len(plan.Disks) != 1 {
			return fmt.Errorf("real formatting currently supports Basic mode with one disk only for ext4 and btrfs")
		}
		if plan.FormatDisk && plan.FileSystem != FileSystemEXT4 && plan.FileSystem != FileSystemBTRFS {
			return fmt.Errorf("real formatting currently supports ext4, btrfs, and zfs only")
		}
	} else if !plan.FormatDisk {
		return fmt.Errorf("mounting an existing zfs pool is not supported by this create flow")
	}
	if strings.TrimSpace(plan.MountPath) == "" {
		return fmt.Errorf("mount path is required")
	}
	if !safeMountPath(plan.MountPath) {
		return fmt.Errorf("mount path must stay under %s", defaultSpaceRoot())
	}
	for _, disk := range plan.Disks {
		if disk.SystemDisk {
			return fmt.Errorf("system disk cannot be used for storage space: %s", disk.Slot)
		}
		if disk.MountPath != "" {
			return fmt.Errorf("disk %s is already mounted at %s", disk.Slot, disk.MountPath)
		}
		if strings.TrimSpace(disk.DevicePath) == "" || strings.ToLower(disk.DeviceType) != "disk" {
			return fmt.Errorf("disk %s must be an unmounted block device", disk.Slot)
		}
		if !strings.HasPrefix(filepath.Clean(disk.DevicePath), "/dev/") {
			return fmt.Errorf("disk %s has unsafe device path: %s", disk.Slot, disk.DevicePath)
		}
	}
	return nil
}

func (p *commandSpaceProvisioner) ensureFstabEntry(uuid string, mountPath string, fileSystem string) error {
	line := fmt.Sprintf("UUID=%s %s %s defaults,nofail 0 2\n", uuid, escapeFstabField(mountPath), fileSystem)
	content, err := os.ReadFile(p.fstabPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read fstab: %w", err)
	}
	if bytes.Contains(content, []byte("UUID="+uuid+" ")) || bytes.Contains(content, []byte(" "+escapeFstabField(mountPath)+" ")) {
		return nil
	}
	file, err := os.OpenFile(p.fstabPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open fstab: %w", err)
	}
	defer file.Close()
	if len(content) > 0 && !bytes.HasSuffix(content, []byte("\n")) {
		if _, err := file.WriteString("\n"); err != nil {
			return fmt.Errorf("write fstab newline: %w", err)
		}
	}
	if _, err := file.WriteString(line); err != nil {
		return fmt.Errorf("write fstab entry: %w", err)
	}
	return nil
}

func filesystemLabel(name string) string {
	label := slugID(name)
	if len(label) > 16 {
		return label[:16]
	}
	return label
}

func zfsPoolName(name string) string {
	pool := "higoos-" + slugID(name)
	if len(pool) > 48 {
		return pool[:48]
	}
	return pool
}

func zfsVdevLayout(mode SpaceMode, disks []Disk) []string {
	devices := make([]string, 0, len(disks)*2)
	switch mode {
	case SpaceModeRAID1:
		devices = append(devices, "mirror")
		for _, disk := range disks {
			devices = append(devices, disk.DevicePath)
		}
	case SpaceModeRAID5:
		devices = append(devices, "raidz1")
		for _, disk := range disks {
			devices = append(devices, disk.DevicePath)
		}
	case SpaceModeRAID6:
		devices = append(devices, "raidz2")
		for _, disk := range disks {
			devices = append(devices, disk.DevicePath)
		}
	case SpaceModeRAID10:
		for index := 0; index+1 < len(disks); index += 2 {
			devices = append(devices, "mirror", disks[index].DevicePath, disks[index+1].DevicePath)
		}
	default:
		for _, disk := range disks {
			devices = append(devices, disk.DevicePath)
		}
	}
	return devices
}

func safeMountPath(value string) bool {
	root := filepath.Clean(defaultSpaceRoot())
	path := filepath.Clean(value)
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != "." && !strings.HasPrefix(rel, "..")
}

func escapeFstabField(value string) string {
	return strings.ReplaceAll(filepath.Clean(value), " ", "\\040")
}
