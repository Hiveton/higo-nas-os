package monitoring

import (
	"context"
	"strings"
	"testing"
)

func TestCurrentMetricsReturnsFrontendAlignedSnapshot(t *testing.T) {
	service := NewService(NewDevCollector())

	snapshot, err := service.CurrentMetrics(context.Background())
	if err != nil {
		t.Fatalf("current metrics: %v", err)
	}

	if len(snapshot.Metrics) != 6 {
		t.Fatalf("expected 6 metrics, got %d", len(snapshot.Metrics))
	}
	wantKeys := []string{"cpu", "memory", "network", "disk", "temperature", "fan"}
	for index, key := range wantKeys {
		metric := snapshot.Metrics[index]
		if metric.Key != key {
			t.Fatalf("metric %d key: got %q want %q", index, metric.Key, key)
		}
		if metric.Label == "" || metric.Unit == "" || metric.Detail == "" || metric.Tone == "" {
			t.Fatalf("metric %q is missing frontend fields: %#v", key, metric)
		}
	}
	if len(snapshot.Services) != 5 {
		t.Fatalf("expected 5 service states, got %d", len(snapshot.Services))
	}
	if snapshot.Services[0].Label != "容器" || snapshot.Services[0].Value == "" || snapshot.Services[0].Tone == "" {
		t.Fatalf("unexpected first service state: %#v", snapshot.Services[0])
	}
}

func TestMetricTrendHonorsRangeAndMetricScaling(t *testing.T) {
	service := NewService(NewDevCollector())

	cpuTrend, err := service.MetricTrend(context.Background(), "cpu", Range1H)
	if err != nil {
		t.Fatalf("cpu trend: %v", err)
	}
	tempTrend, err := service.MetricTrend(context.Background(), "temperature", Range7D)
	if err != nil {
		t.Fatalf("temperature trend: %v", err)
	}

	if cpuTrend.Range != Range1H || cpuTrend.Metric != "cpu" {
		t.Fatalf("unexpected cpu trend identity: %#v", cpuTrend)
	}
	if len(cpuTrend.Points) != 12 {
		t.Fatalf("expected 12 cpu trend points, got %d", len(cpuTrend.Points))
	}
	if tempTrend.Range != Range7D || tempTrend.Metric != "temperature" {
		t.Fatalf("unexpected temperature trend identity: %#v", tempTrend)
	}
	if len(tempTrend.Points) != 12 {
		t.Fatalf("expected 12 temperature trend points, got %d", len(tempTrend.Points))
	}
	if tempTrend.Points[0].Value >= cpuTrend.Points[0].Value {
		t.Fatalf("expected temperature trend to be scaled below cpu baseline, got temp=%v cpu=%v", tempTrend.Points[0].Value, cpuTrend.Points[0].Value)
	}
}

func TestCreateAlertDerivesSeverityAndMuteIsIdempotent(t *testing.T) {
	service := NewService(NewDevCollector())

	alert, err := service.CreateAlert(context.Background(), CreateAlertRequest{Metric: "temperature", Range: Range24H})
	if err != nil {
		t.Fatalf("create alert: %v", err)
	}
	if alert.ID == "" {
		t.Fatal("expected alert id")
	}
	if alert.Severity != SeverityMedium {
		t.Fatalf("expected medium severity for orange metric, got %q", alert.Severity)
	}
	if alert.State != "新建" || alert.Muted {
		t.Fatalf("unexpected created alert state: %#v", alert)
	}

	firstMute, err := service.MuteAlert(context.Background(), alert.ID, true)
	if err != nil {
		t.Fatalf("mute alert: %v", err)
	}
	secondMute, err := service.MuteAlert(context.Background(), alert.ID, true)
	if err != nil {
		t.Fatalf("mute alert again: %v", err)
	}
	if !firstMute.Muted || !secondMute.Muted {
		t.Fatalf("expected muted alert after repeated calls: first=%#v second=%#v", firstMute, secondMute)
	}
	if firstMute.State != "已静音" || secondMute.State != "已静音" {
		t.Fatalf("expected idempotent muted state, got first=%q second=%q", firstMute.State, secondMute.State)
	}

	unmuted, err := service.MuteAlert(context.Background(), alert.ID, false)
	if err != nil {
		t.Fatalf("unmute alert: %v", err)
	}
	if unmuted.Muted || unmuted.State != "待处理" {
		t.Fatalf("expected unmuted pending alert, got %#v", unmuted)
	}
}

func TestRunDiagnosticsCreatesTrackableRunWithSummary(t *testing.T) {
	service := NewService(NewDevCollector())

	run, err := service.RunDiagnostics(context.Background())
	if err != nil {
		t.Fatalf("run diagnostics: %v", err)
	}

	if run.ID == "" || !strings.HasPrefix(run.ID, "diag-") {
		t.Fatalf("expected trackable diagnostic id, got %q", run.ID)
	}
	if run.Status != "completed" {
		t.Fatalf("expected completed diagnostic, got %q", run.Status)
	}
	if run.Summary == "" || !strings.Contains(run.Summary, "指标") || !strings.Contains(run.Summary, "告警") {
		t.Fatalf("expected summary to mention metrics and alerts, got %q", run.Summary)
	}
	if len(run.Checks) == 0 {
		t.Fatal("expected diagnostic checks")
	}
}
