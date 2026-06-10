package monitoring

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const bytesPerMiB = 1024 * 1024

type LinuxCollector struct {
	root        string
	now         func() time.Time
	sampleDelay time.Duration
	statfs      func(string) (diskUsage, error)

	mu          sync.Mutex
	last        resourceSample
	hasLast     bool
	history     map[string][]TrendPoint
	historySize int
}

type resourceSample struct {
	at          time.Time
	cpu         cpuSnapshot
	network     ioBytes
	disk        ioBytes
	memory      memorySnapshot
	diskUsage   diskUsage
	loadAverage string
	uptime      string
	temperature float64
	hasTemp     bool
	fanRPM      float64
	hasFan      bool
	interfaces  int
}

type cpuSnapshot struct {
	total uint64
	idle  uint64
}

type memorySnapshot struct {
	totalBytes     uint64
	availableBytes uint64
	arcBytes       uint64
}

type ioBytes struct {
	read  uint64
	write uint64
}

type diskUsage struct {
	usedPercent float64
	usedBytes   uint64
	totalBytes  uint64
}

func NewDefaultCollector() Collector {
	if _, err := os.Stat("/proc/stat"); err == nil {
		return NewLinuxCollector()
	}
	return NewDevCollector()
}

func NewLinuxCollector() *LinuxCollector {
	return NewLinuxCollectorWithRoot("/")
}

func NewLinuxCollectorWithRoot(root string) *LinuxCollector {
	if strings.TrimSpace(root) == "" {
		root = "/"
	}
	collector := &LinuxCollector{
		root:        root,
		now:         func() time.Time { return time.Now().UTC() },
		sampleDelay: 200 * time.Millisecond,
		history:     make(map[string][]TrendPoint),
		historySize: 720,
	}
	collector.statfs = collector.readDiskUsage
	return collector
}

func (c *LinuxCollector) CurrentMetrics(ctx context.Context) (MetricsSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return MetricsSnapshot{}, err
	}

	first, err := c.readSample()
	if err != nil {
		return MetricsSnapshot{}, err
	}

	current := first
	if c.sampleDelay > 0 {
		timer := time.NewTimer(c.sampleDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return MetricsSnapshot{}, ctx.Err()
		case <-timer.C:
		}
		current, err = c.readSample()
		if err != nil {
			return MetricsSnapshot{}, err
		}
	}

	c.mu.Lock()
	previous := c.last
	hasPrevious := c.hasLast
	if !hasPrevious && c.sampleDelay > 0 {
		previous = first
		hasPrevious = true
	}
	c.last = current
	c.hasLast = true
	c.mu.Unlock()

	elapsed := current.at.Sub(previous.at).Seconds()
	if !hasPrevious || elapsed <= 0 {
		elapsed = 0
	}

	cpuPercent := current.cpu.percent(previous.cpu, hasPrevious)
	memoryPercent := current.memory.usedPercent()
	networkDown := rateMiB(current.network.read, previous.network.read, elapsed)
	networkUp := rateMiB(current.network.write, previous.network.write, elapsed)
	diskRead := rateMiB(current.disk.read, previous.disk.read, elapsed)
	diskWrite := rateMiB(current.disk.write, previous.disk.write, elapsed)
	now := current.at

	metrics := []Metric{
		{
			Key:       "cpu",
			Label:     "CPU",
			Value:     round(cpuPercent, 1),
			Unit:      "%",
			Detail:    fmt.Sprintf("%d 核心 · 负载 %s", runtime.NumCPU(), valueOr(current.loadAverage, "未知")),
			Tone:      toneForPercent(cpuPercent, 75, 90),
			UpdatedAt: now,
		},
		{
			Key:       "memory",
			Label:     "内存",
			Value:     round(memoryPercent, 1),
			Unit:      "%",
			Detail:    memoryDetail(current.memory),
			Tone:      toneForPercent(memoryPercent, 75, 90),
			UpdatedAt: now,
		},
		{
			Key:       "network",
			Label:     "网络",
			Value:     round(math.Max(networkDown, networkUp), 2),
			Unit:      "MB/s",
			Detail:    fmt.Sprintf("↓ %s · ↑ %s · %d 接口", formatRate(networkDown), formatRate(networkUp), current.interfaces),
			Tone:      "blue",
			UpdatedAt: now,
		},
		{
			Key:       "disk",
			Label:     "磁盘",
			Value:     round(current.diskUsage.usedPercent, 1),
			Unit:      "%",
			Detail:    fmt.Sprintf("已用 %s / %s · R %s · W %s", formatBytes(current.diskUsage.usedBytes), formatBytes(current.diskUsage.totalBytes), formatRate(diskRead), formatRate(diskWrite)),
			Tone:      toneForPercent(current.diskUsage.usedPercent, 80, 92),
			UpdatedAt: now,
		},
		{
			Key:       "network_down",
			Label:     "下行",
			Value:     round(networkDown, 2),
			Unit:      "MB/s",
			Detail:    "网络下载速率",
			Tone:      "blue",
			UpdatedAt: now,
		},
		{
			Key:       "network_up",
			Label:     "上行",
			Value:     round(networkUp, 2),
			Unit:      "MB/s",
			Detail:    "网络上传速率",
			Tone:      "cyan",
			UpdatedAt: now,
		},
		{
			Key:       "disk_read",
			Label:     "读取",
			Value:     round(diskRead, 2),
			Unit:      "MB/s",
			Detail:    "磁盘读取速率",
			Tone:      "orange",
			UpdatedAt: now,
		},
		{
			Key:       "disk_write",
			Label:     "写入",
			Value:     round(diskWrite, 2),
			Unit:      "MB/s",
			Detail:    "磁盘写入速率",
			Tone:      "orange",
			UpdatedAt: now,
		},
	}
	if current.hasTemp {
		metrics = append(metrics, Metric{
			Key:       "temperature",
			Label:     "温度",
			Value:     round(current.temperature, 1),
			Unit:      "°C",
			Detail:    "硬件传感器最高温",
			Tone:      toneForPercent(current.temperature, 65, 80),
			UpdatedAt: now,
		})
	}
	if current.hasFan {
		metrics = append(metrics, Metric{
			Key:       "fan",
			Label:     "风扇",
			Value:     round(current.fanRPM, 0),
			Unit:      "RPM",
			Detail:    "硬件传感器平均转速",
			Tone:      "green",
			UpdatedAt: now,
		})
	}

	services := []ServiceStatus{
		{Key: "collector", Label: "采集器", Value: "实时", Detail: "Linux /proc 与 sysfs", Tone: "green"},
		{Key: "uptime", Label: "运行时间", Value: valueOr(current.uptime, "未知"), Detail: fmt.Sprintf("系统负载 %s", valueOr(current.loadAverage, "未知")), Tone: "blue"},
		{Key: "network", Label: "网络接口", Value: fmt.Sprintf("%d 个", current.interfaces), Detail: fmt.Sprintf("↓ %s · ↑ %s", formatRate(networkDown), formatRate(networkUp)), Tone: "blue"},
		{Key: "rootfs", Label: "根分区", Value: fmt.Sprintf("%s%%", trimFloat(current.diskUsage.usedPercent)), Detail: fmt.Sprintf("%s / %s", formatBytes(current.diskUsage.usedBytes), formatBytes(current.diskUsage.totalBytes)), Tone: toneForPercent(current.diskUsage.usedPercent, 80, 92)},
	}

	c.recordHistory(metrics)
	return MetricsSnapshot{Metrics: metrics, Services: services, CollectedAt: now}, nil
}

func (c *LinuxCollector) MetricTrend(ctx context.Context, metric string, rng TimeRange) (MetricTrend, error) {
	if err := ctx.Err(); err != nil {
		return MetricTrend{}, err
	}
	if !knownMetric(metric) {
		return MetricTrend{}, fmt.Errorf("%w: %s", ErrUnknownMetric, metric)
	}
	if _, ok := trendBaselines()[rng]; !ok {
		return MetricTrend{}, fmt.Errorf("%w: %s", ErrUnknownRange, rng)
	}

	if _, err := c.CurrentMetrics(ctx); err != nil {
		return MetricTrend{}, err
	}

	c.mu.Lock()
	points := append([]TrendPoint(nil), c.history[metric]...)
	c.mu.Unlock()
	return MetricTrend{Metric: metric, Range: rng, Points: selectTrendPoints(points, rng)}, nil
}

func (c *LinuxCollector) Logs(ctx context.Context) ([]SystemLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	for _, name := range []string{"var/log/syslog", "var/log/messages", "var/log/kern.log"} {
		logs, err := c.readSystemLog(name)
		if err == nil && len(logs) > 0 {
			return logs, nil
		}
	}
	now := c.now()
	return []SystemLog{{
		ID:        fmt.Sprintf("resource-%d", now.Unix()),
		Level:     "info",
		Source:    "资源采集",
		Message:   "Linux 资源采集器运行中，暂未读取到系统日志文件。",
		At:        now.Format("15:04"),
		Timestamp: now,
	}}, nil
}

func (c *LinuxCollector) Alerts(ctx context.Context) ([]Alert, error) {
	snapshot, err := c.CurrentMetrics(ctx)
	if err != nil {
		return nil, err
	}

	var alerts []Alert
	now := c.now()
	for _, metric := range snapshot.Metrics {
		severity := SeverityLow
		threshold := 0.0
		switch metric.Key {
		case "cpu", "memory":
			threshold = 90
		case "disk":
			threshold = 92
		case "temperature":
			threshold = 80
		default:
			continue
		}
		if metric.Value < threshold {
			continue
		}
		if metric.Value >= threshold+5 {
			severity = SeverityHigh
		} else {
			severity = SeverityMedium
		}
		alerts = append(alerts, Alert{
			ID:        fmt.Sprintf("resource-%s", metric.Key),
			Severity:  severity,
			Title:     fmt.Sprintf("%s 资源偏高", metric.Label),
			Source:    "资源采集",
			Detail:    fmt.Sprintf("%s 当前为 %s%s，已超过 %.0f%s。", metric.Label, trimFloat(metric.Value), metric.Unit, threshold, metric.Unit),
			Muted:     false,
			State:     "待处理",
			Metric:    metric.Key,
			Tone:      metric.Tone,
			CreatedAt: now,
		})
	}
	return alerts, nil
}

func (c *LinuxCollector) readSample() (resourceSample, error) {
	cpu, err := c.readCPU()
	if err != nil {
		return resourceSample{}, err
	}
	mem, _ := c.readMemory()
	net, interfaces, _ := c.readNetwork()
	disk, _ := c.readDiskIO()
	usage, _ := c.statfs(c.rootPath(""))
	temp, hasTemp := c.readTemperature()
	fan, hasFan := c.readFan()
	return resourceSample{
		at:          c.now(),
		cpu:         cpu,
		network:     net,
		disk:        disk,
		memory:      mem,
		diskUsage:   usage,
		loadAverage: c.readLoadAverage(),
		uptime:      c.readUptime(),
		temperature: temp,
		hasTemp:     hasTemp,
		fanRPM:      fan,
		hasFan:      hasFan,
		interfaces:  interfaces,
	}, nil
}

func (c *LinuxCollector) rootPath(name string) string {
	if c.root == "/" {
		return filepath.Join("/", name)
	}
	return filepath.Join(c.root, name)
}

func (c *LinuxCollector) readCPU() (cpuSnapshot, error) {
	content, err := os.ReadFile(c.rootPath("proc/stat"))
	if err != nil {
		return cpuSnapshot{}, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	if !scanner.Scan() {
		return cpuSnapshot{}, fmt.Errorf("empty /proc/stat")
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuSnapshot{}, fmt.Errorf("invalid cpu line in /proc/stat")
	}
	var values []uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuSnapshot{}, err
		}
		values = append(values, value)
	}
	var total uint64
	for _, value := range values {
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuSnapshot{total: total, idle: idle}, nil
}

func (s cpuSnapshot) percent(previous cpuSnapshot, hasPrevious bool) float64 {
	if hasPrevious && s.total > previous.total {
		total := s.total - previous.total
		idle := s.idle - previous.idle
		if total == 0 {
			return 0
		}
		return clamp((float64(total-idle)/float64(total))*100, 0, 100)
	}
	if s.total == 0 {
		return 0
	}
	return clamp((float64(s.total-s.idle)/float64(s.total))*100, 0, 100)
}

func (c *LinuxCollector) readMemory() (memorySnapshot, error) {
	content, err := os.ReadFile(c.rootPath("proc/meminfo"))
	if err != nil {
		return memorySnapshot{}, err
	}
	values := map[string]uint64{}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		fields := strings.Fields(strings.TrimSuffix(scanner.Text(), ":"))
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err == nil {
			values[key] = value * 1024
		}
	}
	mem := memorySnapshot{totalBytes: values["MemTotal"], availableBytes: values["MemAvailable"]}
	if arc, err := c.readZFSARC(); err == nil {
		mem.arcBytes = arc
	}
	return mem, scanner.Err()
}

func (m memorySnapshot) usedPercent() float64 {
	if m.totalBytes == 0 || m.availableBytes > m.totalBytes {
		return 0
	}
	return clamp((float64(m.totalBytes-m.availableBytes)/float64(m.totalBytes))*100, 0, 100)
}

func (c *LinuxCollector) readZFSARC() (uint64, error) {
	content, err := os.ReadFile(c.rootPath("proc/spl/kstat/zfs/arcstats"))
	if err != nil {
		return 0, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 && fields[0] == "size" {
			return strconv.ParseUint(fields[2], 10, 64)
		}
	}
	return 0, fmt.Errorf("zfs arc size not found")
}

func (c *LinuxCollector) readNetwork() (ioBytes, int, error) {
	content, err := os.ReadFile(c.rootPath("proc/net/dev"))
	if err != nil {
		return ioBytes{}, 0, err
	}
	var total ioBytes
	interfaces := 0
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if skipInterface(name) {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rx, errRx := strconv.ParseUint(fields[0], 10, 64)
		tx, errTx := strconv.ParseUint(fields[8], 10, 64)
		if errRx != nil || errTx != nil {
			continue
		}
		total.read += rx
		total.write += tx
		interfaces++
	}
	return total, interfaces, scanner.Err()
}

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

func (c *LinuxCollector) readDiskIO() (ioBytes, error) {
	content, err := os.ReadFile(c.rootPath("proc/diskstats"))
	if err != nil {
		return ioBytes{}, err
	}
	var total ioBytes
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 14 {
			continue
		}
		name := fields[2]
		if skipBlockDevice(name) {
			continue
		}
		readSectors, errRead := strconv.ParseUint(fields[5], 10, 64)
		writeSectors, errWrite := strconv.ParseUint(fields[9], 10, 64)
		if errRead != nil || errWrite != nil {
			continue
		}
		total.read += readSectors * 512
		total.write += writeSectors * 512
	}
	return total, scanner.Err()
}

func skipBlockDevice(name string) bool {
	for _, prefix := range []string{"loop", "ram", "sr", "fd", "dm-"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	last := name[len(name)-1]
	return last >= '0' && last <= '9' && !strings.HasPrefix(name, "nvme")
}

func (c *LinuxCollector) readDiskUsage(path string) (diskUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return diskUsage{}, err
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bavail * uint64(stat.Bsize)
	if total == 0 || free > total {
		return diskUsage{}, nil
	}
	used := total - free
	return diskUsage{usedPercent: (float64(used) / float64(total)) * 100, usedBytes: used, totalBytes: total}, nil
}

func (c *LinuxCollector) readTemperature() (float64, bool) {
	matches, _ := filepath.Glob(c.rootPath("sys/class/hwmon/hwmon*/temp*_input"))
	var maxTemp float64
	for _, match := range matches {
		value, ok := readNumberFile(match)
		if !ok {
			continue
		}
		temp := value
		if temp > 1000 {
			temp = temp / 1000
		}
		if temp > maxTemp {
			maxTemp = temp
		}
	}
	return maxTemp, maxTemp > 0
}

func (c *LinuxCollector) readFan() (float64, bool) {
	matches, _ := filepath.Glob(c.rootPath("sys/class/hwmon/hwmon*/fan*_input"))
	var sum float64
	count := 0
	for _, match := range matches {
		value, ok := readNumberFile(match)
		if !ok || value <= 0 {
			continue
		}
		sum += value
		count++
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

func readNumberFile(path string) (float64, bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, false
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(string(content)), 64)
	return value, err == nil
}

func (c *LinuxCollector) readLoadAverage() string {
	content, err := os.ReadFile(c.rootPath("proc/loadavg"))
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(content))
	if len(fields) < 3 {
		return ""
	}
	return strings.Join(fields[:3], " / ")
}

func (c *LinuxCollector) readUptime() string {
	content, err := os.ReadFile(c.rootPath("proc/uptime"))
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(content))
	if len(fields) == 0 {
		return ""
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return ""
	}
	return formatDuration(time.Duration(seconds) * time.Second)
}

func (c *LinuxCollector) readSystemLog(name string) ([]SystemLog, error) {
	content, err := os.ReadFile(c.rootPath(name))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) > 20 {
		lines = lines[len(lines)-20:]
	}
	logs := make([]SystemLog, 0, len(lines))
	now := c.now()
	for index := len(lines) - 1; index >= 0; index-- {
		line := strings.TrimSpace(lines[index])
		if line == "" {
			continue
		}
		level := "info"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "fail") {
			level = "error"
		} else if strings.Contains(lower, "warn") {
			level = "warn"
		}
		logs = append(logs, SystemLog{
			ID:        fmt.Sprintf("syslog-%d", index),
			Level:     level,
			Source:    "系统日志",
			Message:   compactLogLine(line),
			At:        now.Format("15:04"),
			Timestamp: now.Add(-time.Duration(len(lines)-1-index) * time.Minute),
		})
	}
	return logs, nil
}

func compactLogLine(line string) string {
	fields := strings.Fields(line)
	if len(fields) > 5 {
		line = strings.Join(fields[5:], " ")
	}
	if len([]rune(line)) > 120 {
		return string([]rune(line)[:120])
	}
	return line
}

func (c *LinuxCollector) recordHistory(metrics []Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, metric := range metrics {
		points := append(c.history[metric.Key], TrendPoint{At: metric.UpdatedAt, Value: metric.Value})
		if len(points) > c.historySize {
			points = points[len(points)-c.historySize:]
		}
		c.history[metric.Key] = points
	}
}

func selectTrendPoints(points []TrendPoint, rng TimeRange) []TrendPoint {
	if len(points) == 0 {
		return []TrendPoint{}
	}
	cutoff := points[len(points)-1].At.Add(-rangeDuration(rng))
	filtered := make([]TrendPoint, 0, len(points))
	for _, point := range points {
		if point.At.Equal(cutoff) || point.At.After(cutoff) {
			filtered = append(filtered, point)
		}
	}
	if len(filtered) == 0 {
		filtered = []TrendPoint{points[len(points)-1]}
	}
	if len(filtered) >= 12 {
		return downsampleTrend(filtered, 12)
	}
	return filtered
}

func downsampleTrend(points []TrendPoint, target int) []TrendPoint {
	if len(points) <= target {
		return points
	}
	next := make([]TrendPoint, 0, target)
	for index := 0; index < target; index++ {
		source := int(math.Round(float64(index) * float64(len(points)-1) / float64(target-1)))
		next = append(next, points[source])
	}
	return next
}

func rangeDuration(rng TimeRange) time.Duration {
	switch rng {
	case Range6H:
		return 6 * time.Hour
	case Range24H:
		return 24 * time.Hour
	case Range7D:
		return 7 * 24 * time.Hour
	default:
		return time.Hour
	}
}

func rateMiB(current uint64, previous uint64, elapsed float64) float64 {
	if elapsed <= 0 || current < previous {
		return 0
	}
	return (float64(current-previous) / bytesPerMiB) / elapsed
}

func memoryDetail(mem memorySnapshot) string {
	used := uint64(0)
	if mem.totalBytes >= mem.availableBytes {
		used = mem.totalBytes - mem.availableBytes
	}
	detail := fmt.Sprintf("%s / %s", formatBytes(used), formatBytes(mem.totalBytes))
	if mem.arcBytes > 0 {
		detail += fmt.Sprintf(" · ZFS ARC %s", formatBytes(mem.arcBytes))
	}
	return detail
}

func formatRate(value float64) string {
	if value >= 100 {
		return fmt.Sprintf("%.0f MB/s", value)
	}
	if value >= 10 {
		return fmt.Sprintf("%.1f MB/s", value)
	}
	if value >= 1 {
		return fmt.Sprintf("%.2f MB/s", value)
	}
	return fmt.Sprintf("%.0f KB/s", value*1024)
}

func formatBytes(value uint64) string {
	if value == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB", "PB"}
	next := float64(value)
	unit := 0
	for next >= 1024 && unit < len(units)-1 {
		next /= 1024
		unit++
	}
	if next >= 10 || unit == 0 {
		return fmt.Sprintf("%.0f %s", next, units[unit])
	}
	return fmt.Sprintf("%.1f %s", next, units[unit])
}

func formatDuration(value time.Duration) string {
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

func toneForPercent(value float64, warn float64, high float64) string {
	if value >= high {
		return "orange"
	}
	if value >= warn {
		return "blue"
	}
	return "green"
}

func round(value float64, precision int) float64 {
	pow := math.Pow10(precision)
	return math.Round(value*pow) / pow
}

func trimFloat(value float64) string {
	if math.Abs(value-math.Round(value)) < 0.01 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.1f", value)
}

func valueOr(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
