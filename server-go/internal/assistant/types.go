package assistant

import "time"

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
)

type ActionStatus string

const (
	ActionPending   ActionStatus = "pending"
	ActionConfirmed ActionStatus = "confirmed"
	ActionCanceled  ActionStatus = "canceled"
)

type Thread struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	Messages       []Message `json:"messages"`
	PendingActions []Action  `json:"pendingActions"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type Message struct {
	ID                   string      `json:"id"`
	ThreadID             string      `json:"threadId"`
	Role                 MessageRole `json:"role"`
	Text                 string      `json:"text"`
	Citations            []Citation  `json:"citations,omitempty"`
	ActionID             string      `json:"actionId,omitempty"`
	RequiresConfirmation bool        `json:"requiresConfirmation"`
	CreatedAt            time.Time   `json:"createdAt"`
	ModelPolicy          string      `json:"modelPolicy,omitempty"`
}

type Action struct {
	ID             string       `json:"id"`
	ThreadID       string       `json:"threadId"`
	MessageID      string       `json:"messageId"`
	ActorID        string       `json:"actorId"`
	Intent         string       `json:"intent"`
	Risk           RiskLevel    `json:"risk"`
	Status         ActionStatus `json:"status"`
	ConfirmationID string       `json:"confirmationId"`
	Impact         string       `json:"impact"`
	RollbackID     string       `json:"rollbackId,omitempty"`
	CreatedAt      time.Time    `json:"createdAt"`
	ConfirmedAt    time.Time    `json:"confirmedAt,omitempty"`
	ConfirmedBy    string       `json:"confirmedBy,omitempty"`
}

type MessageRequest struct {
	ActorID     string   `json:"actorId"`
	Text        string   `json:"text"`
	Scopes      []string `json:"scopes"`
	ModelPolicy string   `json:"modelPolicy"`
}

type MessageResult struct {
	Thread               Thread  `json:"thread"`
	UserMessage          Message `json:"userMessage"`
	AssistantMessage     Message `json:"assistantMessage"`
	Action               *Action `json:"action,omitempty"`
	RequiresConfirmation bool    `json:"requiresConfirmation"`
}

type ConfirmActionRequest struct {
	ActorID string `json:"actorId"`
	Intent  string `json:"intent"`
}

type SemanticSearchRequest struct {
	ActorID string   `json:"actorId"`
	Query   string   `json:"query"`
	Scopes  []string `json:"scopes"`
	Limit   int      `json:"limit"`
}

type SemanticSearchResponse struct {
	Answer    string       `json:"answer"`
	Items     []SearchItem `json:"items"`
	Citations []Citation   `json:"citations"`
}

type SearchItem struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Path    string  `json:"path"`
	Scope   string  `json:"scope"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

type Citation struct {
	ItemID string `json:"itemId"`
	Title  string `json:"title"`
	Path   string `json:"path"`
	Scope  string `json:"scope"`
	Quote  string `json:"quote"`
}
