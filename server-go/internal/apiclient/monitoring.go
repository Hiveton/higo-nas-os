package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// MonitoringCurrentMetrics returns the current metric values
// (GET /api/v1/monitoring/metrics/current).
func (c *Client) MonitoringCurrentMetrics(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/monitoring/metrics/current", nil, nil)
}

// MonitoringMetricsSnapshot returns the full metrics snapshot
// (GET /api/v1/monitoring/metrics/snapshot).
func (c *Client) MonitoringMetricsSnapshot(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/monitoring/metrics/snapshot", nil, nil)
}

// MonitoringMetricTrend returns the trend points for a metric over a time range
// (GET /api/v1/monitoring/metrics/trend).
func (c *Client) MonitoringMetricTrend(ctx context.Context, metric, timeRange string) (json.RawMessage, error) {
	query := url.Values{}
	if metric != "" {
		query.Set("metric", metric)
	}
	if timeRange != "" {
		query.Set("range", timeRange)
	}
	return c.Do(ctx, http.MethodGet, "/api/v1/monitoring/metrics/trend", query, nil)
}

// MonitoringLogs returns recent system log entries (GET /api/v1/monitoring/logs).
func (c *Client) MonitoringLogs(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/monitoring/logs", nil, nil)
}

// MonitoringAlerts returns the configured alerts (GET /api/v1/monitoring/alerts).
func (c *Client) MonitoringAlerts(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/monitoring/alerts", nil, nil)
}

// MonitoringCreateAlert creates an alert (POST /api/v1/monitoring/alerts).
func (c *Client) MonitoringCreateAlert(ctx context.Context, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/monitoring/alerts", nil, body)
}

// MonitoringMuteAlert mutes (or unmutes) an alert
// (POST /api/v1/monitoring/alerts/{id}/mute).
func (c *Client) MonitoringMuteAlert(ctx context.Context, id string, body any) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/monitoring/alerts/"+url.PathEscape(id)+"/mute", nil, body)
}

// MonitoringDiagnostics runs a diagnostics pass (POST /api/v1/monitoring/diagnostics).
func (c *Client) MonitoringDiagnostics(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/monitoring/diagnostics", nil, nil)
}
