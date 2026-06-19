package storage

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ZFS maintenance operations. These shell out to the real zfs/zpool tooling on
// the NAS host (Linux + zfsutils); on dev hosts or non-ZFS spaces the snapshot
// and repair tasks fall back to the staged-progress simulation. The command
// runner is injectable so the exact invocations are unit-testable without ZFS.

const (
	zfsBin   = "/usr/sbin/zfs"
	zpoolBin = "/usr/sbin/zpool"
)

// zfsSnapshot creates a point-in-time snapshot of pool and returns the full
// snapshot identifier (pool@higoos-<timestamp>).
func zfsSnapshot(ctx context.Context, runner commandRunner, pool string, now time.Time) (string, error) {
	snap := fmt.Sprintf("%s@higoos-%s", pool, now.UTC().Format("20060102-150405"))
	if _, err := runner(ctx, zfsBin, "snapshot", snap); err != nil {
		return "", fmt.Errorf("create zfs snapshot %s: %w", snap, err)
	}
	return snap, nil
}

// zfsScrub starts a scrub on pool — the real ZFS array consistency check/repair
// that the "repair" task represents.
func zfsScrub(ctx context.Context, runner commandRunner, pool string) error {
	if _, err := runner(ctx, zpoolBin, "scrub", pool); err != nil {
		return fmt.Errorf("start zpool scrub on %s: %w", pool, err)
	}
	return nil
}

// zfsPoolStat is the live capacity/health of a ZFS pool read from `zpool list`.
type zfsPoolStat struct {
	SizeBytes  int64
	AllocBytes int64
	Health     string // raw zfs health: ONLINE / DEGRADED / FAULTED / ...
}

// zfsPoolStatus reads a pool's real size, allocation and health via
// `zpool list -Hp -o size,alloc,health <pool>` (parsable bytes, no header).
func zfsPoolStatus(ctx context.Context, runner commandRunner, pool string) (zfsPoolStat, error) {
	out, err := runner(ctx, zpoolBin, "list", "-Hp", "-o", "size,alloc,health", pool)
	if err != nil {
		return zfsPoolStat{}, fmt.Errorf("read zpool status for %s: %w", pool, err)
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 3 {
		return zfsPoolStat{}, fmt.Errorf("unexpected zpool list output for %s: %q", pool, string(out))
	}
	size, _ := strconv.ParseInt(fields[0], 10, 64)
	alloc, _ := strconv.ParseInt(fields[1], 10, 64)
	return zfsPoolStat{SizeBytes: size, AllocBytes: alloc, Health: fields[2]}, nil
}

// zfsHealthLabel maps a raw zfs pool health to the UI's Chinese health vocabulary.
func zfsHealthLabel(raw string) Health {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "ONLINE":
		return HealthHealthy
	case "DEGRADED":
		return HealthWarning
	case "FAULTED", "UNAVAIL", "REMOVED":
		return HealthCritical
	default:
		return HealthHealthy
	}
}
