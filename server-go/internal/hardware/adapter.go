package hardware

import (
	"context"
	"os"
	"runtime"
	"time"
)

// Adapter resolves the hardware inventory for the current host. Two
// implementations exist: the LinuxAdapter (real /proc, /sys and stdlib reads)
// and the DevAdapter (representative data for macOS development hosts).
type Adapter interface {
	Inventory(context.Context) (Inventory, error)
}

// NewDefaultAdapter mirrors monitoring.NewDefaultCollector: shell out to the
// Linux reader when the proc filesystem is present, otherwise fall back to the
// development stub.
func NewDefaultAdapter() Adapter {
	if _, err := os.Stat("/proc/cpuinfo"); err == nil {
		return NewLinuxAdapter()
	}
	return NewDevAdapter()
}

// DevAdapter returns plausible inventory for hosts without /proc (macOS dev).
// It uses the stdlib for whatever is genuinely available (hostname, arch,
// logical cores, NICs) and representative values for the rest.
type DevAdapter struct {
	now func() time.Time
}

func NewDevAdapter() *DevAdapter {
	return &DevAdapter{now: func() time.Time { return time.Now().UTC() }}
}

func (a *DevAdapter) Inventory(ctx context.Context) (Inventory, error) {
	if err := ctx.Err(); err != nil {
		return Inventory{}, err
	}
	now := a.now()

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "higoos-dev"
	}

	const totalMem = uint64(32) * 1024 * 1024 * 1024 // 32 GiB
	usedMem := totalMem * 62 / 100

	inv := Inventory{
		Host: HostInfo{
			Hostname: hostname,
			OSName:   "HiGoOS Dev (" + runtime.GOOS + ")",
			Kernel:   "dev-stub",
			Arch:     runtime.GOARCH,
			Uptime:   "开发桩数据",
			BootedAt: now.Add(-6 * time.Hour),
		},
		Board: BoardInfo{
			Vendor:  "HiGoOS",
			Product: "Dev Workstation",
			Serial:  "DEV-0000-0000",
			BIOS:    "dev",
		},
		CPU: CPUInfo{
			Model:         "Apple/Dev CPU (representative)",
			PhysicalCores: maxInt(1, runtime.NumCPU()/2),
			LogicalCores:  runtime.NumCPU(),
			MHz:           2800,
		},
		Memory: MemoryInfo{
			TotalBytes:  totalMem,
			UsedBytes:   usedMem,
			UsedPercent: 62,
		},
		Network: []NetworkInterface{
			{Name: "en0", MAC: "02:42:ac:11:00:02", IPv4: []string{"10.211.55.20"}, SpeedMbps: 1000, State: "up"},
			{Name: "en1", MAC: "02:42:ac:11:00:03", IPv4: []string{}, SpeedMbps: 0, State: "down"},
		},
		Sensors: SensorInfo{
			Temperatures: []TempReading{{Label: "CPU", Celsius: 43}},
			Fans:         []FanReading{{Label: "System", RPM: 1280}},
		},
		Adapter:     "dev",
		CollectedAt: now,
	}
	return inv, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
