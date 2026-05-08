package agents

import "time"

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type RunStatus string

const (
	RunPending             RunStatus = "pending"
	RunWaitingConfirmation RunStatus = "waiting_confirmation"
	RunRunning             RunStatus = "running"
	RunCompleted           RunStatus = "completed"
	RunCanceled            RunStatus = "canceled"
)

type Template struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Tools       []Tool    `json:"tools"`
	DefaultRisk RiskLevel `json:"defaultRisk"`
}

type Tool struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Scope       string    `json:"scope"`
	Risk        RiskLevel `json:"risk"`
	Schema      string    `json:"schema"`
}

type WorkflowNode struct {
	ID      string    `json:"id"`
	Label   string    `json:"label"`
	Value   string    `json:"value"`
	ToolID  string    `json:"toolId,omitempty"`
	Risk    RiskLevel `json:"risk"`
	Confirm bool      `json:"confirm"`
}

type WorkflowPreviewRequest struct {
	ActorID    string   `json:"actorId"`
	TemplateID string   `json:"templateId"`
	Goal       string   `json:"goal"`
	Scopes     []string `json:"scopes"`
}

type WorkflowPreview struct {
	ID                   string         `json:"id"`
	TemplateID           string         `json:"templateId"`
	Goal                 string         `json:"goal"`
	Risk                 RiskLevel      `json:"risk"`
	Impact               string         `json:"impact"`
	Nodes                []WorkflowNode `json:"nodes"`
	Checkpoints          []Checkpoint   `json:"checkpoints"`
	RequiresConfirmation bool           `json:"requiresConfirmation"`
	ConfirmationID       string         `json:"confirmationId,omitempty"`
}

type Checkpoint struct {
	NodeID  string    `json:"nodeId"`
	Summary string    `json:"summary"`
	Risk    RiskLevel `json:"risk"`
}

type WorkflowRunRequest struct {
	ActorID    string   `json:"actorId"`
	TemplateID string   `json:"templateId"`
	Goal       string   `json:"goal"`
	Scopes     []string `json:"scopes"`
}

type ConfirmRunRequest struct {
	ActorID        string `json:"actorId"`
	ConfirmationID string `json:"confirmationId"`
}

type CancelRunRequest struct {
	ActorID string `json:"actorId"`
	Reason  string `json:"reason"`
}

type WorkflowRun struct {
	ID                   string    `json:"id"`
	TemplateID           string    `json:"templateId"`
	Goal                 string    `json:"goal"`
	ActorID              string    `json:"actorId"`
	Status               RunStatus `json:"status"`
	Risk                 RiskLevel `json:"risk"`
	RequiresConfirmation bool      `json:"requiresConfirmation"`
	ConfirmationID       string    `json:"confirmationId,omitempty"`
	RollbackID           string    `json:"rollbackId,omitempty"`
	StartedAt            time.Time `json:"startedAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

type WorkflowEvent struct {
	ID        string    `json:"id"`
	RunID     string    `json:"runId"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	NodeID    string    `json:"nodeId,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}
