package hardware

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LinuxAdapter reads real hardware descriptors from /proc, /sys and the Go
// stdlib. It is pure Go (no CGO), so it cross-compiles cleanly to linux/arm64.
// The configurable root mirrors monitoring.LinuxCollector for testability.
type LinuxAdapter struct {
	root string
	now  func() time.Time
}

func NewLinuxAdapter() *LinuxAdapter {
	return NewLinuxAdapterWithRoot("/")
}

func NewLinuxAdapterWithRoot(root string) *LinuxAdapter {
	if strings.TrimSpace(root) == "" {
		root = "/"
	}
	return &LinuxAdapter{
		root: root,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (a *LinuxAdapter) rootPath(name string) string {
	if a.root == "/" {
		return filepath.Join("/", name)
	}
	return filepath.Join(a.root, name)
}

func (a *LinuxAdapter) Inventory(ctx context.Context) (Inventory, error) {
	if err := ctx.Err(); err != nil {
		return Inventory{}, err
	}
	now := a.now()
	return Inventory{
		Host:        a.host(now),
		Board:       a.board(),
		CPU:         a.cpu(),
		Memory:      a.memory(),
		Network:     a.network(),
		Sensors:     a.sensors(),
		Adapter:     "linux",
		CollectedAt: now,
	}, nil
}

func (a *LinuxAdapter) host(now time.Time) HostInfo {
	host := HostInfo{Arch: runtime.GOARCH}
	host.Hostname, _ = os.Hostname()
	host.OSName = a.osPrettyName()
	host.Kernel = firstLine(a.readFile("proc/sys/kernel/osrelease"))
	if seconds, ok := a.uptimeSeconds(); ok {
		uptime := time.Duration(seconds) * time.Second
		host.Uptime = formatUptime(uptime)
		host.BootedAt = now.Add(-uptime)
	}
	return host
}

func (a *LinuxAdapter) osPrettyName() string {
	content := a.readFile("etc/os-release")
	if content == "" {
		return ""
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			value := strings.TrimPrefix(line, "PRETTY_NAME=")
			return strings.Trim(strings.TrimSpace(value), `"`)
		}
	}
	return ""
}

func (a *LinuxAdapter) uptimeSeconds() (float64, bool) {
	fields := strings.Fields(a.readFile("proc/uptime"))
	if len(fields) == 0 {
		return 0, false
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, false
	}
	return seconds, true
}

func (a *LinuxAdapter) board() BoardInfo {
	// DMI files are frequently root-only or absent; read each best-effort.
	board := BoardInfo{
		Vendor:  firstLine(a.readFile("sys/class/dmi/id/board_vendor")),
		Product: firstLine(a.readFile("sys/class/dmi/id/board_name")),
		Serial:  firstLine(a.readFile("sys/class/dmi/id/board_serial")),
		BIOS:    firstLine(a.readFile("sys/class/dmi/id/bios_version")),
	}
	// ARM SoC boards (Rockchip/Amlogic) have no DMI; the device tree is the
	// equivalent identity source.
	if board.Product == "" {
		board.Product = deviceTreeString(a.readFile("proc/device-tree/model"))
	}
	if board.Vendor == "" {
		// /proc/device-tree/compatible is a NUL-separated vendor,model list.
		compatible := strings.TrimRight(a.readFile("proc/device-tree/compatible"), "\x00")
		if first := strings.SplitN(compatible, "\x00", 2)[0]; first != "" {
			board.Vendor = strings.SplitN(first, ",", 2)[0]
		}
	}
	return board
}

func (a *LinuxAdapter) cpu() CPUInfo {
	cpu := CPUInfo{LogicalCores: runtime.NumCPU()}
	content := a.readFile("proc/cpuinfo")
	if content == "" {
		return cpu
	}
	logical := 0
	physicalSet := map[string]struct{}{}
	coreCounts := map[string]int{} // "implementer/part" -> processor count (ARM)
	var curPhysical, curCore, curImpl, curPart, arch string
	flush := func() {
		if curPhysical != "" || curCore != "" {
			physicalSet[curPhysical+"/"+curCore] = struct{}{}
		}
		if curImpl != "" && curPart != "" {
			coreCounts[strings.ToLower(curImpl)+"/"+strings.ToLower(curPart)]++
		}
		curPhysical, curCore, curImpl, curPart = "", "", "", ""
	}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		key, value, ok := splitCPUInfo(line)
		if !ok {
			continue
		}
		switch key {
		case "processor":
			logical++
		case "model name":
			if cpu.Model == "" {
				cpu.Model = value
			}
		case "Hardware", "Model":
			// Some ARM boards expose the SoC name here rather than "model name".
			if cpu.Model == "" {
				cpu.Model = value
			}
		case "cpu MHz":
			if cpu.MHz == 0 {
				if mhz, err := strconv.ParseFloat(value, 64); err == nil {
					cpu.MHz = mhz
				}
			}
		case "physical id":
			curPhysical = value
		case "core id":
			curCore = value
		case "CPU implementer":
			curImpl = value
		case "CPU part":
			curPart = value
		case "CPU architecture":
			if arch == "" {
				arch = value
			}
		}
	}
	flush()
	if logical > 0 {
		cpu.LogicalCores = logical
	}
	if len(physicalSet) > 0 {
		cpu.PhysicalCores = len(physicalSet)
	} else {
		cpu.PhysicalCores = cpu.LogicalCores
	}
	// ARM /proc/cpuinfo carries no "model name" or "cpu MHz"; derive both from the
	// implementer/part registers, the device tree, and sysfs cpufreq instead.
	if cpu.MHz == 0 {
		cpu.MHz = a.cpuMaxMHz()
	}
	if cpu.Model == "" {
		cpu.Model = deviceTreeString(a.readFile("proc/device-tree/model"))
	}
	if cpu.Model == "" {
		cpu.Model = buildArmModel(coreCounts, arch)
	}
	return cpu
}

// cpuMaxMHz reads the max CPU frequency (kHz) from sysfs cpufreq, present on real
// ARM/x86 hosts but typically absent in VMs. Returns 0 when unavailable.
func (a *LinuxAdapter) cpuMaxMHz() float64 {
	raw := firstLine(a.readFile("sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"))
	if raw == "" {
		return 0
	}
	khz, err := strconv.ParseFloat(raw, 64)
	if err != nil || khz <= 0 {
		return 0
	}
	return round2(khz / 1000)
}

// buildArmModel synthesizes a CPU label from the ARM implementer/part registers
// when no model-name string is present, e.g. "Cortex-A76 ×4 + Cortex-A55 ×4" or
// a generic "ARM 处理器 (ARMv8)" fallback.
func buildArmModel(coreCounts map[string]int, arch string) string {
	var named []string
	var implementer string
	for key, count := range coreCounts {
		impl, part, found := strings.Cut(key, "/")
		if !found {
			continue
		}
		if implementer == "" {
			implementer = armImplementerName(impl)
		}
		if name := armCoreName(impl, part); name != "" {
			named = append(named, fmt.Sprintf("%s ×%d", name, count))
		}
	}
	if len(named) > 0 {
		sort.Strings(named)
		return strings.Join(named, " + ")
	}
	archLabel := "ARM"
	switch strings.TrimSpace(arch) {
	case "8":
		archLabel = "ARMv8"
	case "9":
		archLabel = "ARMv9"
	}
	if implementer == "" {
		implementer = "ARM"
	}
	return fmt.Sprintf("%s 处理器 (%s)", implementer, archLabel)
}

func armImplementerName(impl string) string {
	switch strings.ToLower(strings.TrimSpace(impl)) {
	case "0x41":
		return "ARM"
	case "0x42":
		return "Broadcom"
	case "0x43":
		return "Cavium"
	case "0x48":
		return "HiSilicon"
	case "0x4e":
		return "Nvidia"
	case "0x51":
		return "Qualcomm"
	case "0x53":
		return "Samsung"
	case "0x56":
		return "Marvell"
	case "0x61":
		return "Apple"
	case "0xc0":
		return "Ampere"
	}
	return ""
}

// armCoreName maps common ARM implementer/part pairs (relevant to Rockchip /
// Amlogic NAS SoCs) to their core names.
func armCoreName(impl, part string) string {
	if strings.ToLower(strings.TrimSpace(impl)) != "0x41" {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(part)) {
	case "0xd03":
		return "Cortex-A53"
	case "0xd04":
		return "Cortex-A35"
	case "0xd05":
		return "Cortex-A55"
	case "0xd07":
		return "Cortex-A57"
	case "0xd08":
		return "Cortex-A72"
	case "0xd09":
		return "Cortex-A73"
	case "0xd0a":
		return "Cortex-A75"
	case "0xd0b":
		return "Cortex-A76"
	case "0xd0c":
		return "Neoverse-N1"
	case "0xd0d":
		return "Cortex-A77"
	case "0xd41":
		return "Cortex-A78"
	case "0xd44":
		return "Cortex-X1"
	case "0xd46":
		return "Cortex-A510"
	case "0xd47":
		return "Cortex-A710"
	case "0xd48":
		return "Cortex-X2"
	case "0xd4d":
		return "Cortex-A715"
	case "0xd4e":
		return "Cortex-X3"
	}
	return ""
}

// deviceTreeString trims the trailing NUL bytes device-tree property files carry.
func deviceTreeString(content string) string {
	return firstLine(strings.TrimRight(content, "\x00"))
}

func splitCPUInfo(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	return key, value, key != ""
}

func (a *LinuxAdapter) memory() MemoryInfo {
	content := a.readFile("proc/meminfo")
	if content == "" {
		return MemoryInfo{}
	}
	values := map[string]uint64{}
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		if v, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			values[key] = v * 1024 // meminfo is in kB
		}
	}
	mem := MemoryInfo{TotalBytes: values["MemTotal"]}
	available := values["MemAvailable"]
	if mem.TotalBytes > 0 && available <= mem.TotalBytes {
		mem.UsedBytes = mem.TotalBytes - available
		mem.UsedPercent = round2(float64(mem.UsedBytes) / float64(mem.TotalBytes) * 100)
	}
	return mem
}

func (a *LinuxAdapter) network() []NetworkInterface {
	ifaces, err := net.Interfaces()
	if err != nil {
		return []NetworkInterface{}
	}
	result := make([]NetworkInterface, 0, len(ifaces))
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 || skipInterface(iface.Name) {
			continue
		}
		entry := NetworkInterface{
			Name:  iface.Name,
			MAC:   iface.HardwareAddr.String(),
			IPv4:  []string{},
			State: a.interfaceState(iface),
		}
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			if ip4 := ipNet.IP.To4(); ip4 != nil {
				entry.IPv4 = append(entry.IPv4, ip4.String())
			}
		}
		entry.SpeedMbps = a.interfaceSpeed(iface.Name)
		result = append(result, entry)
	}
	return result
}

func (a *LinuxAdapter) interfaceState(iface net.Interface) string {
	if state := firstLine(a.readFile("sys/class/net/" + iface.Name + "/operstate")); state != "" && state != "unknown" {
		return state
	}
	if iface.Flags&net.FlagUp != 0 {
		return "up"
	}
	return "down"
}

func (a *LinuxAdapter) interfaceSpeed(name string) int {
	raw := firstLine(a.readFile("sys/class/net/" + name + "/speed"))
	if raw == "" {
		return 0
	}
	speed, err := strconv.Atoi(raw)
	if err != nil || speed < 0 {
		return 0
	}
	return speed
}

func (a *LinuxAdapter) sensors() SensorInfo {
	sensors := SensorInfo{Temperatures: []TempReading{}, Fans: []FanReading{}}
	for _, match := range a.glob("sys/class/hwmon/hwmon*/temp*_input") {
		value, ok := readNumberFile(match)
		if !ok {
			continue
		}
		if value > 1000 {
			value = value / 1000
		}
		sensors.Temperatures = append(sensors.Temperatures, TempReading{
			Label:   a.sensorLabel(match, "温度"),
			Celsius: round2(value),
		})
	}
	for _, match := range a.glob("sys/class/hwmon/hwmon*/fan*_input") {
		value, ok := readNumberFile(match)
		if !ok || value <= 0 {
			continue
		}
		sensors.Fans = append(sensors.Fans, FanReading{
			Label: a.sensorLabel(match, "风扇"),
			RPM:   round2(value),
		})
	}
	return sensors
}

// sensorLabel reads the sibling *_label file (e.g. temp1_label) when present,
// falling back to a generic label.
func (a *LinuxAdapter) sensorLabel(inputPath, fallback string) string {
	labelPath := strings.TrimSuffix(inputPath, "_input") + "_label"
	if label := firstLine(readWholeFile(labelPath)); label != "" {
		return label
	}
	return fallback
}

func (a *LinuxAdapter) glob(pattern string) []string {
	matches, _ := filepath.Glob(a.rootPath(pattern))
	return matches
}

func (a *LinuxAdapter) readFile(name string) string {
	return readWholeFile(a.rootPath(name))
}

func readWholeFile(path string) string {
	content, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(content)
}

func readNumberFile(path string) (float64, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
	return value, err == nil
}

func firstLine(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if idx := strings.IndexByte(content, '\n'); idx >= 0 {
		return strings.TrimSpace(content[:idx])
	}
	return content
}

// skipInterface mirrors the monitoring package's virtual-interface filter; kept
// local so this package has no cross-domain dependency on unexported helpers.
func skipInterface(name string) bool {
	if name == "lo" {
		return true
	}
	for _, prefix := range []string{"docker", "br-", "veth", "virbr", "tun", "tap"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func formatUptime(value time.Duration) string {
	if value < time.Minute {
		return "<1 分钟"
	}
	days := int(value.Hours()) / 24
	hours := int(value.Hours()) % 24
	minutes := int(value.Minutes()) % 60
	parts := []string{}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%d 天", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d 小时", hours))
	}
	if days == 0 && minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d 分钟", minutes))
	}
	if len(parts) == 0 {
		return "<1 分钟"
	}
	return strings.Join(parts, " ")
}

func round2(value float64) float64 {
	return float64(int64(value*100+0.5)) / 100
}
