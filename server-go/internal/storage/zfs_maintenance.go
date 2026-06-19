package storage

import (
	"context"
	"fmt"
	"sort"
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

// ZFSSnapshot is one point-in-time snapshot of a ZFS pool/dataset.
type ZFSSnapshot struct {
	Name      string    `json:"name"` // pool@snap
	Pool      string    `json:"pool"`
	UsedBytes int64     `json:"usedBytes"`
	Used      string    `json:"used"`
	CreatedAt time.Time `json:"createdAt"`
}

// zfsListSnapshots lists snapshots under pool via
// `zfs list -t snapshot -H -p -o name,used,creation -r <pool>` (tab-separated,
// parsable bytes/unix-time). Newest first.
func zfsListSnapshots(ctx context.Context, runner commandRunner, pool string) ([]ZFSSnapshot, error) {
	out, err := runner(ctx, zfsBin, "list", "-t", "snapshot", "-H", "-p", "-o", "name,used,creation", "-r", pool)
	if err != nil {
		return nil, fmt.Errorf("list zfs snapshots for %s: %w", pool, err)
	}
	var snaps []ZFSSnapshot
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}
		used, _ := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
		created, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
		snaps = append(snaps, ZFSSnapshot{
			Name:      fields[0],
			Pool:      pool,
			UsedBytes: used,
			Used:      formatStorageBytes(used),
			CreatedAt: time.Unix(created, 0).UTC(),
		})
	}
	sort.SliceStable(snaps, func(i, j int) bool { return snaps[i].CreatedAt.After(snaps[j].CreatedAt) })
	return snaps, nil
}

// zfsRollback rolls a pool back to snapshot (pool@snap). This is destructive —
// changes made since the snapshot are discarded — so callers must gate it.
func zfsRollback(ctx context.Context, runner commandRunner, snapshot string) error {
	if !strings.Contains(snapshot, "@") {
		return fmt.Errorf("invalid snapshot name %q (expected pool@snapshot)", snapshot)
	}
	if _, err := runner(ctx, zfsBin, "rollback", snapshot); err != nil {
		return fmt.Errorf("rollback to %s: %w", snapshot, err)
	}
	return nil
}

// ZFSPoolDetail is the rich health/efficiency view of a ZFS pool, surfacing the
// figures that make ZFS valuable (compression, fragmentation, dedup).
type ZFSPoolDetail struct {
	Pool          string `json:"pool"`
	SizeBytes     int64  `json:"sizeBytes"`
	AllocBytes    int64  `json:"allocBytes"`
	FreeBytes     int64  `json:"freeBytes"`
	CapacityPct   int    `json:"capacityPct"`
	Fragmentation int    `json:"fragmentation"`
	DedupRatio    string `json:"dedupRatio"`
	CompressRatio string `json:"compressRatio"`
	Health        Health `json:"health"`
}

// zfsPoolDetail reads a pool's efficiency/health figures via
// `zpool list -Hp -o size,alloc,free,cap,frag,dedup,health` plus the dataset
// `compressratio`.
func zfsPoolDetail(ctx context.Context, runner commandRunner, pool string) (ZFSPoolDetail, error) {
	out, err := runner(ctx, zpoolBin, "list", "-Hp", "-o", "size,alloc,free,cap,frag,dedup,health", pool)
	if err != nil {
		return ZFSPoolDetail{}, fmt.Errorf("read zpool detail for %s: %w", pool, err)
	}
	fields := strings.Split(strings.TrimSpace(string(out)), "\t")
	if len(fields) < 7 {
		return ZFSPoolDetail{}, fmt.Errorf("unexpected zpool list output for %s: %q", pool, string(out))
	}
	size, _ := strconv.ParseInt(strings.TrimSpace(fields[0]), 10, 64)
	alloc, _ := strconv.ParseInt(strings.TrimSpace(fields[1]), 10, 64)
	free, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
	cap, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(fields[3], "%")))
	frag, _ := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(fields[4], "%")))
	detail := ZFSPoolDetail{
		Pool:          pool,
		SizeBytes:     size,
		AllocBytes:    alloc,
		FreeBytes:     free,
		CapacityPct:   cap,
		Fragmentation: frag,
		DedupRatio:    normalizeRatio(fields[5]),
		Health:        zfsHealthLabel(fields[6]),
	}
	if cout, err := runner(ctx, zfsBin, "get", "-Hp", "-o", "value", "compressratio", pool); err == nil {
		detail.CompressRatio = normalizeRatio(string(cout))
	}
	return detail, nil
}

// normalizeRatio renders a zfs ratio value (e.g. "1.00" or "1.45x") as "N.NNx".
func normalizeRatio(raw string) string {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ""
	}
	if !strings.HasSuffix(v, "x") {
		v += "x"
	}
	return v
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
