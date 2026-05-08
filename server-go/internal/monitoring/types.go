package monitoring

import "time"

type Severity string

const (
	SeverityLow    Severity = "低风险"
	SeverityMedium Severity = "中风险"
	SeverityHigh   Severity = "高风险"
)

type TimeRange string

const (
	Range1H  TimeRange = "1H"
	Range6H  TimeRange = "6H"
	Range24H TimeRange = "24H"
	Range7D  TimeRange = "7D"
)

type Metric struct {
	Key       string    `json:"key"`
	Label     string    `json:"label"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Detail    string    `json:"detail"`
	Tone      string    `json:"tone"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ServiceStatus struct {
	Key    string `json:"key"`
	Label  string `json:"label"`
	Value  string `json:"value"`
	Detail string `json:"detail"`
	Tone   string `json:"tone"`
}

type MetricsSnapshot struct {
	Metrics     []Metric        `json:"metrics"`
	Services    []ServiceStatus `json:"services"`
	CollectedAt time.Time       `json:"collectedAt"`
}

type TrendPoint struct {
	At    time.Time `json:"at"`
	Value float64   `json:"value"`
}

type MetricTrend struct {
	Metric string       `json:"metric"`
	Range  TimeRange    `json:"range"`
	Points []TrendPoint `json:"points"`
}

type SystemLog struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	At        string    `json:"at"`
	Timestamp time.Time `json:"timestamp"`
}

type Alert struct {
	ID        string     `json:"id"`
	Severity  Severity   `json:"severity"`
	Title     string     `json:"title"`
	Source    string     `json:"source"`
	Detail    string     `json:"detail"`
	Muted     bool       `json:"muted"`
	State     string     `json:"state"`
	Metric    string     `json:"metric"`
	Tone      string     `json:"tone"`
	CreatedAt time.Time  `json:"createdAt"`
	MutedAt   *time.Time `json:"mutedAt,omitempty"`
}

type DiagnosticCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Detail  string `json:"detail"`
	Elapsed string `json:"elapsed"`
}

type DiagnosticRun struct {
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	Summary     string            `json:"summary"`
	StartedAt   time.Time         `json:"startedAt"`
	CompletedAt time.Time         `json:"completedAt"`
	Checks      []DiagnosticCheck `json:"checks"`
}

type CreateAlertRequest struct {
	Metric string    `json:"metric"`
	Range  TimeRange `json:"range"`
	Title  string    `json:"title,omitempty"`
	Source string    `json:"source,omitempty"`
	Detail string    `json:"detail,omitempty"`
}
