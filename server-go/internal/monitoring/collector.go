package monitoring

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

var (
	ErrUnknownMetric = errors.New("unknown monitoring metric")
	ErrUnknownRange  = errors.New("unknown monitoring time range")
)

type Collector interface {
	CurrentMetrics(context.Context) (MetricsSnapshot, error)
	MetricTrend(context.Context, string, TimeRange) (MetricTrend, error)
	Logs(context.Context) ([]SystemLog, error)
	Alerts(context.Context) ([]Alert, error)
}

type DevCollector struct {
	now func() time.Time
}

func NewDevCollector() *DevCollector {
	return &DevCollector{now: func() time.Time { return time.Now().UTC() }}
}

func (c *DevCollector) CurrentMetrics(ctx context.Context) (MetricsSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return MetricsSnapshot{}, err
	}

	now := c.now()
	metrics := []Metric{
		{Key: "cpu", Label: "CPU", Value: 38, Unit: "%", Detail: "4C / 8T · 2.8GHz boost", Tone: "green", UpdatedAt: now},
		{Key: "memory", Label: "内存", Value: 62, Unit: "%", Detail: "19.8GB / 32GB · ZFS ARC 8.4GB", Tone: "blue", UpdatedAt: now},
		{Key: "network", Label: "网络", Value: 71, Unit: "%", Detail: "2.3Gbps 下行 · 840Mbps 上行", Tone: "blue", UpdatedAt: now},
		{Key: "disk", Label: "磁盘", Value: 46, Unit: "%", Detail: "主机卷 I/O · 812MB/s", Tone: "green", UpdatedAt: now},
		{Key: "temperature", Label: "温度", Value: 43, Unit: "°C", Detail: "CPU 43°C · 硬盘均值 36°C", Tone: "orange", UpdatedAt: now},
		{Key: "fan", Label: "风扇", Value: 1280, Unit: "RPM", Detail: "静音曲线 · 双风扇同步", Tone: "green", UpdatedAt: now},
	}
	services := []ServiceStatus{
		{Key: "containers", Label: "容器", Value: "18 / 20", Detail: "Plex 转码容器限速中", Tone: "orange"},
		{Key: "apps", Label: "应用", Value: "42", Detail: "2 个应用等待更新", Tone: "blue"},
		{Key: "tasks", Label: "任务", Value: "7", Detail: "照片识别队列运行中", Tone: "green"},
		{Key: "backups", Label: "备份", Value: "3", Detail: "MacBook Pro 增量备份 82%", Tone: "green"},
		{Key: "downloads", Label: "下载", Value: "11", Detail: "2 个任务因低速排队", Tone: "orange"},
	}

	return MetricsSnapshot{
		Metrics:     append([]Metric(nil), metrics...),
		Services:    append([]ServiceStatus(nil), services...),
		CollectedAt: now,
	}, nil
}

func (c *DevCollector) MetricTrend(ctx context.Context, metric string, rng TimeRange) (MetricTrend, error) {
	if err := ctx.Err(); err != nil {
		return MetricTrend{}, err
	}
	if !knownMetric(metric) {
		return MetricTrend{}, fmt.Errorf("%w: %s", ErrUnknownMetric, metric)
	}

	values, ok := trendBaselines()[rng]
	if !ok {
		return MetricTrend{}, fmt.Errorf("%w: %s", ErrUnknownRange, rng)
	}

	points := make([]TrendPoint, len(values))
	interval := rangeInterval(rng)
	start := c.now().Add(-interval * time.Duration(len(values)-1))
	scale := metricTrendScale(metric)
	for index, value := range values {
		points[index] = TrendPoint{
			At:    start.Add(interval * time.Duration(index)),
			Value: clamp(math.Round(value*scale), 0, metricTrendMax(metric)),
		}
	}

	return MetricTrend{Metric: metric, Range: rng, Points: points}, nil
}

func (c *DevCollector) Logs(ctx context.Context) ([]SystemLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	now := c.now()
	logs := []SystemLog{
		{ID: "log-1", Level: "info", Source: "备份中心", Message: "Time Machine 增量备份已校验 82%", At: "14:26", Timestamp: now.Add(-4 * time.Minute)},
		{ID: "log-2", Level: "warn", Source: "容器运行时", Message: "plex-transcoder CPU 峰值持续 6 分钟", At: "14:19", Timestamp: now.Add(-11 * time.Minute)},
		{ID: "log-3", Level: "info", Source: "下载服务", Message: "下载任务已切换到夜间限速策略", At: "14:08", Timestamp: now.Add(-22 * time.Minute)},
		{ID: "log-4", Level: "warn", Source: "硬盘健康", Message: "槽位 4 温度高于 38°C，风扇曲线已提升", At: "13:56", Timestamp: now.Add(-34 * time.Minute)},
	}
	return append([]SystemLog(nil), logs...), nil
}

func (c *DevCollector) Alerts(ctx context.Context) ([]Alert, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	now := c.now()
	alerts := []Alert{
		{
			ID:        "alert-1",
			Severity:  SeverityMedium,
			Title:     "容器 CPU 峰值",
			Source:    "plex-transcoder",
			Detail:    "过去 10 分钟 CPU 平均 78%，建议限制转码并发。",
			Muted:     false,
			State:     "待处理",
			Metric:    "cpu",
			Tone:      "orange",
			CreatedAt: now.Add(-10 * time.Minute),
		},
		{
			ID:        "alert-2",
			Severity:  SeverityLow,
			Title:     "下载任务低速",
			Source:    "下载中心",
			Detail:    "2 个任务低于 300KB/s，已等待下一轮自动重试。",
			Muted:     false,
			State:     "观察中",
			Metric:    "network",
			Tone:      "blue",
			CreatedAt: now.Add(-22 * time.Minute),
		},
	}
	return append([]Alert(nil), alerts...), nil
}

func trendBaselines() map[TimeRange][]float64 {
	return map[TimeRange][]float64{
		Range1H:  {34, 38, 41, 45, 43, 39, 42, 50, 47, 44, 40, 38},
		Range6H:  {28, 33, 39, 55, 48, 52, 61, 58, 46, 43, 49, 44},
		Range24H: {31, 44, 39, 36, 58, 64, 52, 47, 42, 56, 62, 49},
		Range7D:  {26, 38, 45, 41, 53, 69, 57, 51, 48, 59, 63, 54},
	}
}

func knownMetric(metric string) bool {
	switch metric {
	case "cpu", "memory", "network", "disk", "temperature", "fan":
		return true
	default:
		return false
	}
}

func rangeInterval(rng TimeRange) time.Duration {
	switch rng {
	case Range6H:
		return 30 * time.Minute
	case Range24H:
		return 2 * time.Hour
	case Range7D:
		return 14 * time.Hour
	default:
		return 5 * time.Minute
	}
}

func metricTrendScale(metric string) float64 {
	switch metric {
	case "temperature":
		return 0.72
	case "fan":
		return 20
	default:
		return 1
	}
}

func metricTrendMax(metric string) float64 {
	if metric == "fan" {
		return 2200
	}
	return 100
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
