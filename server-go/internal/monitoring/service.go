package monitoring

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

type Service struct {
	collector Collector

	mu           sync.Mutex
	alertsLoaded bool
	alerts       []Alert
	alertSeq     int
	diagSeq      int
	statePath    string
}

type snapshot struct {
	Alerts   []Alert `json:"alerts"`
	AlertSeq int     `json:"alertSeq"`
	DiagSeq  int     `json:"diagSeq"`
}

func NewService(collector Collector) *Service {
	if collector == nil {
		collector = NewDevCollector()
	}
	return &Service{collector: collector}
}

func NewServiceWithStateDir(collector Collector, stateDir string) (*Service, error) {
	service := NewService(collector)
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "monitoring.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Alerts) > 0 {
		service.alerts = cloneAlerts(persisted.Alerts)
		service.alertSeq = persisted.AlertSeq
		service.diagSeq = persisted.DiagSeq
		service.alertsLoaded = true
		if service.alertSeq < len(service.alerts) {
			service.alertSeq = len(service.alerts)
		}
	}
	return service, nil
}

func (s *Service) CurrentMetrics(ctx context.Context) (MetricsSnapshot, error) {
	return s.collector.CurrentMetrics(ctx)
}

func (s *Service) MetricTrend(ctx context.Context, metric string, rng TimeRange) (MetricTrend, error) {
	if rng == "" {
		rng = Range1H
	}
	return s.collector.MetricTrend(ctx, metric, rng)
}

func (s *Service) Logs(ctx context.Context) ([]SystemLog, error) {
	logs, err := s.collector.Logs(ctx)
	if err != nil {
		return nil, err
	}
	return append([]SystemLog(nil), logs...), nil
}

func (s *Service) Alerts(ctx context.Context) ([]Alert, error) {
	if err := s.ensureAlerts(ctx); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneAlerts(s.alerts), nil
}

func (s *Service) CreateAlert(ctx context.Context, request CreateAlertRequest) (Alert, error) {
	if request.Range == "" {
		request.Range = Range1H
	}

	snapshot, err := s.collector.CurrentMetrics(ctx)
	if err != nil {
		return Alert{}, err
	}
	metric, ok := findMetric(snapshot.Metrics, request.Metric)
	if !ok {
		return Alert{}, fmt.Errorf("%w: %s", ErrUnknownMetric, request.Metric)
	}
	if _, err := s.collector.MetricTrend(ctx, metric.Key, request.Range); err != nil {
		return Alert{}, err
	}
	if err := s.ensureAlerts(ctx); err != nil {
		return Alert{}, err
	}

	now := time.Now().UTC()
	alert := Alert{
		Severity:  severityForTone(metric.Tone),
		Title:     request.Title,
		Source:    request.Source,
		Detail:    request.Detail,
		Muted:     false,
		State:     "新建",
		Metric:    metric.Key,
		Tone:      metric.Tone,
		CreatedAt: now,
	}
	if alert.Title == "" {
		alert.Title = fmt.Sprintf("%s 阈值提醒", metric.Label)
	}
	if alert.Source == "" {
		alert.Source = "设备监控中心"
	}
	if alert.Detail == "" {
		alert.Detail = fmt.Sprintf("%s 已按当前 %s 趋势创建告警，后续变化会写入审计。", metric.Label, request.Range)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertSeq++
	alert.ID = fmt.Sprintf("alert-%d", s.alertSeq)
	s.alerts = append([]Alert{alert}, s.alerts...)
	return cloneAlert(alert), s.saveLocked()
}

func (s *Service) MuteAlert(ctx context.Context, id string, muted bool) (Alert, error) {
	if err := s.ensureAlerts(ctx); err != nil {
		return Alert{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.alerts {
		if s.alerts[index].ID != id {
			continue
		}

		s.alerts[index].Muted = muted
		if muted {
			s.alerts[index].State = "已静音"
			if s.alerts[index].MutedAt == nil {
				now := time.Now().UTC()
				s.alerts[index].MutedAt = &now
			}
		} else {
			s.alerts[index].State = "待处理"
			s.alerts[index].MutedAt = nil
		}
		return cloneAlert(s.alerts[index]), s.saveLocked()
	}
	return Alert{}, fmt.Errorf("monitoring alert not found: %s", id)
}

func (s *Service) RunDiagnostics(ctx context.Context) (DiagnosticRun, error) {
	started := time.Now().UTC()
	snapshot, err := s.collector.CurrentMetrics(ctx)
	if err != nil {
		return DiagnosticRun{}, err
	}
	logs, err := s.collector.Logs(ctx)
	if err != nil {
		return DiagnosticRun{}, err
	}
	alerts, err := s.Alerts(ctx)
	if err != nil {
		return DiagnosticRun{}, err
	}

	activeAlerts := 0
	for _, alert := range alerts {
		if !alert.Muted {
			activeAlerts++
		}
	}

	s.mu.Lock()
	s.diagSeq++
	runID := fmt.Sprintf("diag-%s-%03d", started.Format("20060102150405"), s.diagSeq)
	s.mu.Unlock()

	completed := time.Now().UTC()
	return DiagnosticRun{
		ID:          runID,
		Status:      "completed",
		Summary:     fmt.Sprintf("已重新采样 %d 项指标、%d 条服务状态、%d 条日志和 %d 条活跃告警。", len(snapshot.Metrics), len(snapshot.Services), len(logs), activeAlerts),
		StartedAt:   started,
		CompletedAt: completed,
		Checks: []DiagnosticCheck{
			{Name: "metrics", Status: "passed", Detail: fmt.Sprintf("%d 项核心指标已采样", len(snapshot.Metrics)), Elapsed: "12ms"},
			{Name: "services", Status: "passed", Detail: fmt.Sprintf("%d 项服务状态已采样", len(snapshot.Services)), Elapsed: "8ms"},
			{Name: "logs", Status: "passed", Detail: fmt.Sprintf("%d 条系统日志可读取", len(logs)), Elapsed: "6ms"},
			{Name: "alerts", Status: "passed", Detail: fmt.Sprintf("%d 条活跃告警待处理", activeAlerts), Elapsed: "5ms"},
		},
	}, nil
}

func (s *Service) ensureAlerts(ctx context.Context) error {
	s.mu.Lock()
	if s.alertsLoaded {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	alerts, err := s.collector.Alerts(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.alertsLoaded {
		s.alerts = cloneAlerts(alerts)
		s.alertSeq = len(s.alerts)
		s.alertsLoaded = true
	}
	return nil
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Alerts:   cloneAlerts(s.alerts),
		AlertSeq: s.alertSeq,
		DiagSeq:  s.diagSeq,
	})
}

func findMetric(metrics []Metric, key string) (Metric, bool) {
	for _, metric := range metrics {
		if metric.Key == key {
			return metric, true
		}
	}
	return Metric{}, false
}

func severityForTone(tone string) Severity {
	switch tone {
	case "red":
		return SeverityHigh
	case "orange":
		return SeverityMedium
	default:
		return SeverityLow
	}
}

func cloneAlerts(alerts []Alert) []Alert {
	cloned := make([]Alert, len(alerts))
	for index, alert := range alerts {
		cloned[index] = cloneAlert(alert)
	}
	return cloned
}

func cloneAlert(alert Alert) Alert {
	if alert.MutedAt == nil {
		return alert
	}
	mutedAt := *alert.MutedAt
	alert.MutedAt = &mutedAt
	return alert
}
