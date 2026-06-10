package monitoring

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxCollectorCurrentMetricsReadsProcAndSysfs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "proc/stat", "cpu  100 0 100 800 0 0 0 0 0 0\n")
	writeFile(t, root, "proc/meminfo", "MemTotal:       1000000 kB\nMemAvailable:    250000 kB\n")
	writeFile(t, root, "proc/net/dev", `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
    lo: 1000 0 0 0 0 0 0 0 1000 0 0 0 0 0 0 0
  eth0: 1000000 0 0 0 0 0 0 0 2000000 0 0 0 0 0 0 0
`)
	writeFile(t, root, "proc/diskstats", "   8       0 sda 10 0 1000 0 20 0 2000 0 0 0 0 0 0 0 0 0 0\n")
	writeFile(t, root, "proc/loadavg", "0.10 0.20 0.30 1/100 123\n")
	writeFile(t, root, "proc/uptime", "3600.00 1000.00\n")
	writeFile(t, root, "sys/class/hwmon/hwmon0/temp1_input", "42000\n")
	writeFile(t, root, "sys/class/hwmon/hwmon0/fan1_input", "1234\n")

	collector := NewLinuxCollectorWithRoot(root)
	collector.now = func() time.Time { return time.Unix(1700000000, 0).UTC() }
	collector.sampleDelay = 0
	collector.statfs = func(string) (diskUsage, error) {
		return diskUsage{usedPercent: 42, usedBytes: 420 * 1024 * 1024, totalBytes: 1000 * 1024 * 1024}, nil
	}

	snapshot, err := collector.CurrentMetrics(context.Background())
	if err != nil {
		t.Fatalf("current metrics: %v", err)
	}

	assertMetric(t, snapshot.Metrics, "cpu", 20, "%")
	assertMetric(t, snapshot.Metrics, "memory", 75, "%")
	assertMetric(t, snapshot.Metrics, "network", 0, "MB/s")
	assertMetric(t, snapshot.Metrics, "disk", 42, "%")
	assertMetric(t, snapshot.Metrics, "temperature", 42, "°C")
	assertMetric(t, snapshot.Metrics, "fan", 1234, "RPM")
	assertMetric(t, snapshot.Metrics, "network_down", 0, "MB/s")
	assertMetric(t, snapshot.Metrics, "disk_write", 0, "MB/s")

	if len(snapshot.Services) == 0 {
		t.Fatal("expected derived service status rows")
	}
}

func TestLinuxCollectorProducesRatesFromConsecutiveSnapshots(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "proc/stat", "cpu  100 0 100 800 0 0 0 0 0 0\n")
	writeFile(t, root, "proc/meminfo", "MemTotal:       1000000 kB\nMemAvailable:    900000 kB\n")
	writeFile(t, root, "proc/net/dev", `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 1048576 0 0 0 0 0 0 0 2097152 0 0 0 0 0 0 0
`)
	writeFile(t, root, "proc/diskstats", "   8       0 sda 0 0 2048 0 0 0 4096 0 0 0 0 0 0 0 0 0 0\n")
	writeFile(t, root, "proc/loadavg", "0.10 0.20 0.30 1/100 123\n")
	writeFile(t, root, "proc/uptime", "10.00 1.00\n")

	collector := NewLinuxCollectorWithRoot(root)
	tick := time.Unix(1700000000, 0).UTC()
	collector.now = func() time.Time { return tick }
	collector.sampleDelay = 0
	collector.statfs = func(string) (diskUsage, error) {
		return diskUsage{usedPercent: 10, usedBytes: 10, totalBytes: 100}, nil
	}

	if _, err := collector.CurrentMetrics(context.Background()); err != nil {
		t.Fatalf("first sample: %v", err)
	}

	tick = tick.Add(time.Second)
	writeFile(t, root, "proc/stat", "cpu  150 0 150 900 0 0 0 0 0 0\n")
	writeFile(t, root, "proc/net/dev", `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0: 3145728 0 0 0 0 0 0 0 3145728 0 0 0 0 0 0 0
`)
	writeFile(t, root, "proc/diskstats", "   8       0 sda 0 0 4096 0 0 0 8192 0 0 0 0 0 0 0 0 0 0\n")

	snapshot, err := collector.CurrentMetrics(context.Background())
	if err != nil {
		t.Fatalf("second sample: %v", err)
	}

	assertMetric(t, snapshot.Metrics, "cpu", 50, "%")
	assertMetric(t, snapshot.Metrics, "network_down", 2, "MB/s")
	assertMetric(t, snapshot.Metrics, "network_up", 1, "MB/s")
	assertMetric(t, snapshot.Metrics, "disk_read", 1, "MB/s")
	assertMetric(t, snapshot.Metrics, "disk_write", 2, "MB/s")
}

func writeFile(t *testing.T, root string, name string, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertMetric(t *testing.T, metrics []Metric, key string, value float64, unit string) {
	t.Helper()
	metric, ok := findMetric(metrics, key)
	if !ok {
		t.Fatalf("missing metric %q in %#v", key, metrics)
	}
	if metric.Value != value || metric.Unit != unit {
		t.Fatalf("metric %s: got %v%s want %v%s (%#v)", key, metric.Value, metric.Unit, value, unit, metric)
	}
	if metric.Label == "" || metric.Detail == "" || metric.Tone == "" || metric.UpdatedAt.IsZero() {
		t.Fatalf("metric %s missing display fields: %#v", key, metric)
	}
}
