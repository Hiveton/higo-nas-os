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
	ID         string           `json:"id"`
	Title      string           `json:"title"`
	Detail     string           `json:"detail"`
	Count      string           `json:"count"`
	Risk       RiskLevel        `json:"risk"`
	Action     string           `json:"action"`
	Status     SuggestionStatus `json:"status"`
	Operations []SuggestionOp   `json:"operations,omitempty"`
	UpdateAt   time.Time        `json:"updatedAt"`
}

// SuggestionOp is a concrete, executable file operation behind a suggestion.
// When present (and a file executor is attached) confirming the suggestion
// performs the real operation; rollback reverses it.
//
//   - Type "delete": moves the file to the recycle bin, reversed by restoring it.
//   - Type "move":   moves the file to Dest (a space display path such as
//     "备份归档"), reversed by moving it back to its original path.
type SuggestionOp struct {
	Type   string `json:"type"`
	FileID string `json:"fileId"`
	Dest   string `json:"dest,omitempty"`
}

// ExecutedOp records one operation that actually ran, with enough information
// to reverse it. A move changes the file's id (ids are path-derived), so we
// keep the post-move id plus the original path; a delete keeps the original id
// (restorable from the recycle bin).
type ExecutedOp struct {
	Type     string `json:"type"`
	FileID   string `json:"fileId"`
	FromPath string `json:"fromPath,omitempty"`
	ToPath   string `json:"toPath,omitempty"`
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
	ID              string       `json:"id"`
	SuggestionID    string       `json:"suggestionId,omitempty"`
	Message         string       `json:"message"`
	ActorID         string       `json:"actorId,omitempty"`
	Risk            RiskLevel    `json:"risk"`
	Result          AuditResult  `json:"result"`
	ConfirmationID  string       `json:"confirmationId,omitempty"`
	RollbackID      string       `json:"rollbackId,omitempty"`
	ExecutedFileIDs []string     `json:"executedFileIds,omitempty"`
	ExecutedOps     []ExecutedOp `json:"executedOps,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
	RolledBackAt    time.Time    `json:"rolledBackAt,omitempty"`
}
