package storage

import (
	"context"
	"fmt"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type commandRunner func(context.Context, string, ...string) ([]byte, error)

type HostAdapter struct {
	runner commandRunner
	now    func() time.Time
}

type hostVolume struct {
	filesystem string
	mount      string
	totalKB    int64
	usedKB     int64
	available  int64
	usedPct    int
}

func NewHostAdapter() *HostAdapter {
	return NewHostAdapterWithRunner(runCommand)
}

func NewHostAdapterWithRunner(runner commandRunner) *HostAdapter {
	if runner == nil {
		runner = runCommand
	}
	return &HostAdapter{
		runner: runner,
		now:    time.Now,
	}
}

func (a *HostAdapter) Pools(ctx context.Context) ([]StoragePool, error) {
	volumes, err := a.volumes(ctx)
	if err != nil {
		return nil, err
	}
	pools := make([]StoragePool, 0, len(volumes))
	for _, volume := range volumes {
		pools = append(pools, StoragePool{
			ID:          volumeID(volume),
			Name:        volumeName(volume.mount),
			Type:        hostVolumeType(volume.filesystem),
			UsedPercent: volume.usedPct,
			Total:       formatStorageBytes(volume.totalKB * 1024),
			Health:      healthFromUsage(volume.usedPct),
			Temperature: "N/A",
		})
	}
	return pools, nil
}

func (a *HostAdapter) Disks(ctx context.Context) ([]Disk, error) {
	volumes, err := a.volumes(ctx)
	if err != nil {
		return nil, err
	}
	disks := make([]Disk, 0, len(volumes))
	for index, volume := range volumes {
		disks = append(disks, Disk{
			Slot:        strconv.Itoa(index + 1),
			Size:        formatStorageBytes(volume.totalKB * 1024),
			State:       diskStateFromUsage(volume.usedPct),
			Temperature: "N/A",
			Serial:      volume.filesystem,
			Health:      healthFromUsage(volume.usedPct),
			Role:        "volume",
			PoolID:      volumeID(volume),
			Model:       volumeName(volume.mount),
			Interface:   "mount",
		})
	}
	return disks, nil
}

func (a *HostAdapter) SmartReports(ctx context.Context) ([]SmartReport, error) {
	volumes, err := a.volumes(ctx)
	if err != nil {
		return nil, err
	}
	updatedAt := a.now().UTC()
	reports := make([]SmartReport, 0, len(volumes))
	for index, volume := range volumes {
		health := healthFromUsage(volume.usedPct)
		reports = append(reports, SmartReport{
			DiskSlot:    strconv.Itoa(index + 1),
			Serial:      volume.filesystem,
			Health:      health,
			Temperature: "N/A",
			UpdatedAt:   updatedAt,
			Attributes: []SmartAttribute{
				{Name: "Filesystem_Usage", Value: volume.usedPct, Threshold: 90, Status: smartStatusFromHealth(health)},
			},
		})
	}
	return reports, nil
}

func (a *HostAdapter) volumes(ctx context.Context) ([]hostVolume, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	output, err := a.runner(ctx, "df", "-kP")
	if err != nil {
		return nil, fmt.Errorf("read host filesystems with df: %w", err)
	}
	volumes, err := parseDFOutput(string(output))
	if err != nil {
		return nil, err
	}
	if len(volumes) == 0 {
		return nil, fmt.Errorf("no host storage volumes found")
	}
	return volumes, nil
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

func parseDFOutput(output string) ([]hostVolume, error) {
	lines := strings.Split(output, "\n")
	volumes := make([]hostVolume, 0, len(lines))
	seenMounts := map[string]struct{}{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Filesystem") {
			continue
		}
		volume, ok := parseDFLine(line)
		if !ok || isPseudoFilesystem(volume.filesystem) || volume.totalKB <= 0 || volume.mount == "" {
			continue
		}
		if _, exists := seenMounts[volume.mount]; exists {
			continue
		}
		seenMounts[volume.mount] = struct{}{}
		volumes = append(volumes, volume)
	}
	return volumes, nil
}

func parseDFLine(line string) (hostVolume, bool) {
	fields := strings.Fields(line)
	for index := 1; index+4 < len(fields); index++ {
		totalKB, totalErr := strconv.ParseInt(fields[index], 10, 64)
		usedKB, usedErr := strconv.ParseInt(fields[index+1], 10, 64)
		availableKB, availErr := strconv.ParseInt(fields[index+2], 10, 64)
		usedPct, pctErr := parsePercent(fields[index+3])
		if totalErr != nil || usedErr != nil || availErr != nil || pctErr != nil {
			continue
		}
		return hostVolume{
			filesystem: strings.Join(fields[:index], " "),
			mount:      strings.Join(fields[index+4:], " "),
			totalKB:    totalKB,
			usedKB:     usedKB,
			available:  availableKB,
			usedPct:    usedPct,
		}, true
	}
	return hostVolume{}, false
}

func parsePercent(value string) (int, error) {
	return strconv.Atoi(strings.TrimSuffix(value, "%"))
}

func isPseudoFilesystem(filesystem string) bool {
	name := strings.ToLower(strings.TrimSpace(filesystem))
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, "map ") {
		return true
	}
	switch name {
	case "devfs", "proc", "procfs", "sysfs", "devtmpfs", "autofs", "fdesc", "linprocfs", "linsysfs":
		return true
	default:
		return false
	}
}

func volumeID(volume hostVolume) string {
	return "host-" + slugID(volume.filesystem+"-"+volume.mount)
}

func slugID(value string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(value) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}

func volumeName(mount string) string {
	cleaned := path.Clean(mount)
	if cleaned == "/" {
		return "系统根卷"
	}
	if strings.HasPrefix(cleaned, "/System/Volumes/") {
		return path.Base(cleaned) + " 卷"
	}
	if strings.HasPrefix(cleaned, "/Volumes/") {
		return strings.TrimPrefix(cleaned, "/Volumes/")
	}
	base := path.Base(cleaned)
	if base == "." || base == "/" || base == "" {
		return cleaned
	}
	return base
}

func hostVolumeType(filesystem string) string {
	if runtime.GOOS == "darwin" && strings.HasPrefix(filesystem, "/dev/disk") {
		return "APFS 卷"
	}
	if strings.HasPrefix(filesystem, "/dev/") {
		return "主机卷"
	}
	return "主机文件系统"
}

func healthFromUsage(usedPct int) Health {
	switch {
	case usedPct >= 97:
		return HealthCritical
	case usedPct >= 90:
		return HealthWarning
	default:
		return HealthHealthy
	}
}

func diskStateFromUsage(usedPct int) DiskState {
	if usedPct >= 97 {
		return DiskStateOffline
	}
	return DiskStateHealthy
}

func smartStatusFromHealth(health Health) string {
	switch health {
	case HealthHealthy:
		return "ok"
	case HealthWarning, HealthSyncing:
		return "warning"
	default:
		return "critical"
	}
}

func formatStorageBytes(bytes int64) string {
	if bytes <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	value := float64(bytes)
	unitIndex := 0
	for value >= 1000 && unitIndex < len(units)-1 {
		value /= 1000
		unitIndex++
	}
	if value >= 10 || unitIndex == 0 {
		return fmt.Sprintf("%.0f %s", value, units[unitIndex])
	}
	return fmt.Sprintf("%.1f %s", value, units[unitIndex])
}
