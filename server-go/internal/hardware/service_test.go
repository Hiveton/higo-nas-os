package hardware

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDevAdapterInventoryShape(t *testing.T) {
	service := NewService(NewDevAdapter())

	inv, err := service.Inventory(context.Background())
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}

	if inv.Adapter != "dev" {
		t.Fatalf("adapter: got %q want %q", inv.Adapter, "dev")
	}
	if inv.Host.Hostname == "" {
		t.Fatal("host hostname is empty")
	}
	if inv.Host.Arch == "" {
		t.Fatal("host arch is empty")
	}
	if inv.CPU.LogicalCores <= 0 {
		t.Fatalf("logical cores: got %d want > 0", inv.CPU.LogicalCores)
	}
	if inv.CPU.PhysicalCores <= 0 {
		t.Fatalf("physical cores: got %d want > 0", inv.CPU.PhysicalCores)
	}
	if inv.Memory.TotalBytes == 0 {
		t.Fatal("memory total is zero")
	}
	if inv.Memory.UsedPercent < 0 || inv.Memory.UsedPercent > 100 {
		t.Fatalf("memory used percent out of range: %v", inv.Memory.UsedPercent)
	}
	if len(inv.Network) == 0 {
		t.Fatal("expected at least one network interface")
	}
	if inv.Network[0].MAC == "" {
		t.Fatal("first network interface has empty MAC")
	}
	if len(inv.Sensors.Temperatures) == 0 {
		t.Fatal("expected at least one temperature reading")
	}
	if inv.CollectedAt.IsZero() {
		t.Fatal("collectedAt is zero")
	}
}

func TestLinuxAdapterParsesFixtureRoot(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "proc/cpuinfo", sampleCPUInfo)
	writeFixture(t, root, "proc/meminfo", sampleMemInfo)
	writeFixture(t, root, "proc/uptime", "123456.78 987654.32\n")
	writeFixture(t, root, "proc/sys/kernel/osrelease", "6.8.0-higo-arm64\n")
	writeFixture(t, root, "etc/os-release", "NAME=\"Ubuntu\"\nPRETTY_NAME=\"Ubuntu 24.04 LTS\"\n")
	writeFixture(t, root, "sys/class/dmi/id/board_vendor", "ACME\n")
	writeFixture(t, root, "sys/class/hwmon/hwmon0/temp1_input", "45000\n")
	writeFixture(t, root, "sys/class/hwmon/hwmon0/temp1_label", "CPU Package\n")
	writeFixture(t, root, "sys/class/hwmon/hwmon0/fan1_input", "1500\n")

	adapter := NewLinuxAdapterWithRoot(root)
	inv, err := adapter.Inventory(context.Background())
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}

	if inv.Adapter != "linux" {
		t.Fatalf("adapter: got %q want linux", inv.Adapter)
	}
	if inv.CPU.Model != "Test CPU @ 2.40GHz" {
		t.Fatalf("cpu model: got %q", inv.CPU.Model)
	}
	if inv.CPU.LogicalCores != 4 {
		t.Fatalf("logical cores: got %d want 4", inv.CPU.LogicalCores)
	}
	if inv.CPU.PhysicalCores != 2 {
		t.Fatalf("physical cores: got %d want 2", inv.CPU.PhysicalCores)
	}
	if inv.Memory.TotalBytes != 8192*1024 {
		t.Fatalf("memory total: got %d", inv.Memory.TotalBytes)
	}
	if inv.Host.OSName != "Ubuntu 24.04 LTS" {
		t.Fatalf("os name: got %q", inv.Host.OSName)
	}
	if inv.Host.Kernel != "6.8.0-higo-arm64" {
		t.Fatalf("kernel: got %q", inv.Host.Kernel)
	}
	if inv.Board.Vendor != "ACME" {
		t.Fatalf("board vendor: got %q", inv.Board.Vendor)
	}
	if len(inv.Sensors.Temperatures) != 1 || inv.Sensors.Temperatures[0].Celsius != 45 {
		t.Fatalf("temperatures: %#v", inv.Sensors.Temperatures)
	}
	if inv.Sensors.Temperatures[0].Label != "CPU Package" {
		t.Fatalf("temp label: got %q", inv.Sensors.Temperatures[0].Label)
	}
	if len(inv.Sensors.Fans) != 1 || inv.Sensors.Fans[0].RPM != 1500 {
		t.Fatalf("fans: %#v", inv.Sensors.Fans)
	}
}

func TestLinuxAdapterArmCpuFallback(t *testing.T) {
	root := t.TempDir()
	// ARM cpuinfo: no "model name"/"cpu MHz", big.LITTLE implementer/part pairs.
	writeFixture(t, root, "proc/cpuinfo", sampleArmCPUInfo)
	writeFixture(t, root, "proc/meminfo", sampleMemInfo)
	writeFixture(t, root, "proc/device-tree/model", "Rockchip RK3588 NAS\x00")
	writeFixture(t, root, "sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq", "2400000\n")

	adapter := NewLinuxAdapterWithRoot(root)
	inv, err := adapter.Inventory(context.Background())
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if inv.CPU.LogicalCores != 4 {
		t.Fatalf("logical cores: got %d want 4", inv.CPU.LogicalCores)
	}
	if inv.CPU.MHz != 2400 {
		t.Fatalf("mhz from cpufreq: got %v want 2400", inv.CPU.MHz)
	}
	// device-tree model is the SoC identity → used as CPU model on ARM.
	if inv.CPU.Model != "Rockchip RK3588 NAS" {
		t.Fatalf("cpu model: got %q", inv.CPU.Model)
	}
	if inv.Board.Product != "Rockchip RK3588 NAS" {
		t.Fatalf("board product: got %q", inv.Board.Product)
	}
}

func TestBuildArmModelDecodesCores(t *testing.T) {
	model := buildArmModel(map[string]int{"0x41/0xd0b": 4, "0x41/0xd05": 4}, "8")
	if model != "Cortex-A55 ×4 + Cortex-A76 ×4" {
		t.Fatalf("decoded model: got %q", model)
	}
	generic := buildArmModel(map[string]int{"0x41/0x000": 4}, "8")
	if generic != "ARM 处理器 (ARMv8)" {
		t.Fatalf("generic model: got %q", generic)
	}
}

const sampleArmCPUInfo = `processor	: 0
BogoMIPS	: 48.00
CPU implementer	: 0x41
CPU architecture: 8
CPU variant	: 0x0
CPU part	: 0xd0b
CPU revision	: 0

processor	: 1
CPU implementer	: 0x41
CPU architecture: 8
CPU part	: 0xd0b

processor	: 2
CPU implementer	: 0x41
CPU architecture: 8
CPU part	: 0xd05

processor	: 3
CPU implementer	: 0x41
CPU architecture: 8
CPU part	: 0xd05
`

const sampleCPUInfo = `processor	: 0
model name	: Test CPU @ 2.40GHz
cpu MHz		: 2400.000
physical id	: 0
core id		: 0

processor	: 1
model name	: Test CPU @ 2.40GHz
physical id	: 0
core id		: 1

processor	: 2
model name	: Test CPU @ 2.40GHz
physical id	: 0
core id		: 0

processor	: 3
model name	: Test CPU @ 2.40GHz
physical id	: 0
core id		: 1
`

const sampleMemInfo = `MemTotal:        8192 kB
MemFree:         1024 kB
MemAvailable:    4096 kB
`

func writeFixture(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", full, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}
