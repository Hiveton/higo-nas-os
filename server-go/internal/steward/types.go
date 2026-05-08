package steward

import "time"

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type SuggestionStatus string

const (
	SuggestionPending   SuggestionStatus = "pending"
	SuggestionConfirmed SuggestionStatus = "confirmed"
	SuggestionDismissed SuggestionStatus = "dismissed"
)

type AuditResult string

const (
	AuditAllowed    AuditResult = "allowed"
	AuditConfirmed  AuditResult = "confirmed"
	AuditDismissed  AuditResult = "dismissed"
	AuditRolledBack AuditResult = "rolled_back"
)

type Suggestion struct {
	ID       string           `json:"id"`
	Title    string           `json:"title"`
	Detail   string           `json:"detail"`
	Count    string           `json:"count"`
	Risk     RiskLevel        `json:"risk"`
	Action   string           `json:"action"`
	Status   SuggestionStatus `json:"status"`
	UpdateAt time.Time        `json:"updatedAt"`
}

type PreviewRequest struct {
	ActorID string `json:"actorId"`
}

type SuggestionPreview struct {
	SuggestionID         string    `json:"suggestionId"`
	Impact               string    `json:"impact"`
	Risk                 RiskLevel `json:"risk"`
	RequiresConfirmation bool      `json:"requiresConfirmation"`
	ConfirmationID       string    `json:"confirmationId,omitempty"`
	RollbackID           string    `json:"rollbackId,omitempty"`
}

type ConfirmRequest struct {
	ActorID        string `json:"actorId"`
	ConfirmationID string `json:"confirmationId"`
}

type ConfirmResult struct {
	Suggestion Suggestion `json:"suggestion"`
	AuditEntry AuditEntry `json:"auditEntry"`
}

type DismissRequest struct {
	ActorID string `json:"actorId"`
	Reason  string `json:"reason"`
}

type RollbackRequest struct {
	ActorID string `json:"actorId"`
	Reason  string `json:"reason"`
}

type AuditEntry struct {
	ID             string      `json:"id"`
	SuggestionID   string      `json:"suggestionId,omitempty"`
	Message        string      `json:"message"`
	ActorID        string      `json:"actorId,omitempty"`
	Risk           RiskLevel   `json:"risk"`
	Result         AuditResult `json:"result"`
	ConfirmationID string      `json:"confirmationId,omitempty"`
	RollbackID     string      `json:"rollbackId,omitempty"`
	CreatedAt      time.Time   `json:"createdAt"`
	RolledBackAt   time.Time   `json:"rolledBackAt,omitempty"`
}
