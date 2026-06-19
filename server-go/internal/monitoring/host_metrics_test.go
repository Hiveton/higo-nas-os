package monitoring

import (
	"context"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestHostDiskUsageReportsRealValues(t *testing.T) {
	usedPct, usedGB, totalGB, ok := hostDiskUsage("/")
	if !ok {
		t.Skip("statfs unavailable on this platform")
	}
	if usedPct < 0 || usedPct > 100 {
		t.Fatalf("used percent out of range: %v", usedPct)
	}
	if totalGB <= 0 || usedGB < 0 || usedGB > totalGB {
		t.Fatalf("implausible disk figures: used=%v total=%v", usedGB, totalGB)
	}
}

func TestDevCollectorSurfacesRealDiskAndCPU(t *testing.T) {
	snap, err := NewDevCollector().CurrentMetrics(context.Background())
	if err != nil {
		t.Fatalf("current metrics: %v", err)
	}
	var cpu, disk *Metric
	for i := range snap.Metrics {
		switch snap.Metrics[i].Key {
		case "cpu":
			cpu = &snap.Metrics[i]
		case "disk":
			disk = &snap.Metrics[i]
		}
	}
	if cpu == nil || disk == nil {
		t.Fatalf("expected cpu and disk metrics, got %#v", snap.Metrics)
	}
	// CPU detail reflects the real core count.
	if !strings.Contains(cpu.Detail, strconv.Itoa(runtime.NumCPU())) {
		t.Fatalf("cpu detail should contain real core count %d: %q", runtime.NumCPU(), cpu.Detail)
	}
	// Disk value reflects the real filesystem (matches a fresh statfs reading).
	if usedPct, _, _, ok := hostDiskUsage("/"); ok {
		if disk.Value < 0 || disk.Value > 100 {
			t.Fatalf("disk metric out of range: %v", disk.Value)
		}
		if diff := disk.Value - usedPct; diff > 2 || diff < -2 {
			t.Fatalf("disk metric %v should be near real usage %v", disk.Value, usedPct)
		}
	}
}
