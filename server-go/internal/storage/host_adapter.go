package storage

import (
	"context"
	"encoding/json"
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

type lsblkOutput struct {
	BlockDevices []blockDevice `json:"blockdevices"`
}

type blockDevice struct {
	Name        string          `json:"name"`
	Path        string          `json:"path"`
	Type        string          `json:"type"`
	Size        int64           `json:"size"`
	FSUsed      json.RawMessage `json:"fsused"`
	FSSize      json.RawMessage `json:"fssize"`
	Rotational  *bool           `json:"rota"`
	Transport   string          `json:"tran"`
	Model       string          `json:"model"`
	Serial      string          `json:"serial"`
	Mountpoints []string        `json:"mountpoints"`
	FileSystem  string          `json:"fstype"`
	State       string          `json:"state"`
	Children    []blockDevice   `json:"children"`
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
			MountPath:   volume.mount,
		})
	}
	return pools, nil
}

func (a *HostAdapter) Disks(ctx context.Context) ([]Disk, error) {
	blockDisks, err := a.blockDisks(ctx)
	if err == nil && len(blockDisks) > 0 {
		return blockDisks, nil
	}
	return a.volumeDisks(ctx)
}

func (a *HostAdapter) volumeDisks(ctx context.Context) ([]Disk, error) {
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
			DeviceType:  "volume",
			MediaType:   "挂载卷",
			MountPath:   volume.mount,
		})
	}
	return disks, nil
}

func (a *HostAdapter) blockDisks(ctx context.Context) ([]Disk, error) {
	output, err := a.runner(ctx, "lsblk", "-J", "-b", "-o", "NAME,PATH,TYPE,SIZE,FSUSED,FSSIZE,ROTA,TRAN,MODEL,SERIAL,MOUNTPOINTS,FSTYPE,STATE")
	if err != nil {
		return nil, fmt.Errorf("read host block devices with lsblk: %w", err)
	}
	var decoded lsblkOutput
	if err := json.Unmarshal(output, &decoded); err != nil {
		return nil, fmt.Errorf("parse lsblk json: %w", err)
	}
	disks := make([]Disk, 0, len(decoded.BlockDevices))
	for _, device := range decoded.BlockDevices {
		if strings.ToLower(device.Type) != "disk" {
			continue
		}
		disks = append(disks, diskFromBlockDevice(device))
	}
	return disks, nil
}

func diskFromBlockDevice(device blockDevice) Disk {
	slot := strings.TrimSpace(device.Name)
	if slot == "" {
		slot = strings.TrimPrefix(device.Path, "/dev/")
	}
	transport := strings.TrimSpace(device.Transport)
	mediaType := mediaTypeFromBlockDevice(device)
	return Disk{
		Slot:        slot,
		Size:        formatStorageBytes(device.Size),
		State:       diskStateFromBlockState(device.State),
		Temperature: "N/A",
		Serial:      strings.TrimSpace(device.Serial),
		Health:      HealthHealthy,
		Role:        "disk",
		PoolID:      "disk-" + slugID(firstNonEmpty(device.Path, slot)),
		Model:       firstNonEmpty(strings.TrimSpace(device.Model), slot),
		Interface:   firstNonEmpty(transport, "block"),
		DevicePath:  device.Path,
		DeviceType:  device.Type,
		MediaType:   mediaType,
		Rotational:  device.Rotational,
		SystemDisk:  hasSystemMount(device),
		FileSystem:  firstFilesystem(device),
		MountPath:   firstMountpoint(device),
		Partitions:  partitionsFromBlockDevice(device),
	}
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
	case "devfs", "proc", "procfs", "sysfs", "devtmpfs", "tmpfs", "efivarfs", "autofs", "fdesc", "linprocfs", "linsysfs":
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
	out := strings.Trim(builder.String(), "-")
	if out == "" {
		return "item"
	}
	return out
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

func diskStateFromBlockState(state string) DiskState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "", "running", "live":
		return DiskStateHealthy
	case "offline":
		return DiskStateOffline
	default:
		return DiskStateHealthy
	}
}

func mediaTypeFromBlockDevice(device blockDevice) string {
	model := strings.ToLower(device.Model)
	transport := strings.ToLower(device.Transport)
	switch {
	case strings.Contains(model, "nvme") || transport == "nvme":
		return "NVMe SSD"
	case strings.Contains(model, "virtual"):
		if device.Rotational != nil && !*device.Rotational {
			return "虚拟 SSD"
		}
		return "虚拟磁盘"
	case device.Rotational != nil && !*device.Rotational:
		return "SSD"
	case device.Rotational != nil && *device.Rotational:
		return "HDD"
	default:
		return "磁盘"
	}
}

func partitionsFromBlockDevice(device blockDevice) []DiskPartition {
	partitions := make([]DiskPartition, 0, len(device.Children))
	for _, child := range device.Children {
		if strings.ToLower(strings.TrimSpace(child.Type)) == "part" {
			partition := DiskPartition{
				Name:       firstNonEmpty(child.Name, strings.TrimPrefix(child.Path, "/dev/")),
				Path:       child.Path,
				Size:       formatStorageBytes(child.Size),
				FileSystem: strings.TrimSpace(child.FileSystem),
				MountPath:  firstMountpoint(child),
				System:     hasSystemMount(child),
			}
			if used := parseOptionalBlockBytes(child.FSUsed); used > 0 {
				partition.Used = formatStorageBytes(used)
			}
			if total := parseOptionalBlockBytes(child.FSSize); total > 0 {
				partition.Total = formatStorageBytes(total)
			}
			partitions = append(partitions, partition)
			continue
		}
		partitions = append(partitions, partitionsFromBlockDevice(child)...)
	}
	return partitions
}

func parseOptionalBlockBytes(raw json.RawMessage) int64 {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return 0
	}
	text = strings.Trim(text, `"`)
	if text == "" || text == "-" {
		return 0
	}
	if value, err := strconv.ParseInt(text, 10, 64); err == nil {
		return value
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0
	}
	return int64(value)
}

func firstMountpoint(device blockDevice) string {
	for _, mountpoint := range device.Mountpoints {
		if strings.TrimSpace(mountpoint) != "" {
			return mountpoint
		}
	}
	for _, child := range device.Children {
		if mountpoint := firstMountpoint(child); mountpoint != "" {
			return mountpoint
		}
	}
	return ""
}

func hasSystemMount(device blockDevice) bool {
	for _, mountpoint := range device.Mountpoints {
		switch path.Clean(strings.TrimSpace(mountpoint)) {
		case "/", "/boot", "/boot/efi":
			return true
		}
	}
	for _, child := range device.Children {
		if hasSystemMount(child) {
			return true
		}
	}
	return false
}

func firstFilesystem(device blockDevice) string {
	if strings.TrimSpace(device.FileSystem) != "" {
		return device.FileSystem
	}
	for _, child := range device.Children {
		if filesystem := firstFilesystem(child); filesystem != "" {
			return filesystem
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
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
