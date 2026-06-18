package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// MonitoringMetricTrendInput parameterizes a metric trend query.
type MonitoringMetricTrendInput struct {
	Metric string `json:"metric,omitempty" jsonschema:"Metric key to fetch the trend for (e.g. cpu)"`
	Range  string `json:"range,omitempty" jsonschema:"Time range for the trend (e.g. 1h, 6h, 24h, 7d)"`
}

// MonitoringCreateAlertInput describes a new alert rule.
type MonitoringCreateAlertInput struct {
	Metric string `json:"metric" jsonschema:"Metric the alert watches"`
	Range  string `json:"range,omitempty" jsonschema:"Time range the alert evaluates over"`
	Title  string `json:"title,omitempty" jsonschema:"Human-readable alert title"`
	Source string `json:"source,omitempty" jsonschema:"Source the alert originates from"`
	Detail string `json:"detail,omitempty" jsonschema:"Detail describing the alert condition"`
}

// MonitoringMuteAlertInput mutes or unmutes an alert.
type MonitoringMuteAlertInput struct {
	ID    string `json:"id" jsonschema:"ID of the alert to mute or unmute"`
	Muted *bool  `json:"muted,omitempty" jsonschema:"Whether the alert should be muted (defaults to true)"`
}

func registerMonitoring(r *registry) {
	addTool(r, "monitoring", "higo.monitoring.metrics.current",
		"Get the current monitoring metric values.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MonitoringCurrentMetrics(ctx)
		})

	addTool(r, "monitoring", "higo.monitoring.metrics.snapshot",
		"Get the full monitoring metrics snapshot including services.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MonitoringMetricsSnapshot(ctx)
		})

	addTool(r, "monitoring", "higo.monitoring.metrics.trend",
		"Get the trend points for a metric over a time range.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in MonitoringMetricTrendInput) (json.RawMessage, error) {
			return c.MonitoringMetricTrend(ctx, in.Metric, in.Range)
		})

	addTool(r, "monitoring", "higo.monitoring.logs",
		"Get recent system log entries.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MonitoringLogs(ctx)
		})

	addTool(r, "monitoring", "higo.monitoring.alerts.list",
		"List the configured monitoring alerts.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MonitoringAlerts(ctx)
		})

	addTool(r, "monitoring", "higo.monitoring.alerts.create",
		"Create a monitoring alert rule.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MonitoringCreateAlertInput) (json.RawMessage, error) {
			return c.MonitoringCreateAlert(ctx, in)
		})

	addTool(r, "monitoring", "higo.monitoring.alerts.mute",
		"Mute or unmute a monitoring alert.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in MonitoringMuteAlertInput) (json.RawMessage, error) {
			return c.MonitoringMuteAlert(ctx, in.ID, in)
		})

	addTool(r, "monitoring", "higo.monitoring.diagnostics.run",
		"Run a system diagnostics pass.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.MonitoringDiagnostics(ctx)
		})
}
