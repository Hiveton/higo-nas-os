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
	Tools                []ToolTrace `json:"tools,omitempty"`
	ActionID             string      `json:"actionId,omitempty"`
	RequiresConfirmation bool        `json:"requiresConfirmation"`
	CreatedAt            time.Time   `json:"createdAt"`
	ModelPolicy          string      `json:"modelPolicy,omitempty"`
}

// ToolTrace is a compact record of one tool the agent called while producing a
// message. It is persisted so the analysis steps remain visible on reload.
type ToolTrace struct {
	Name    string `json:"name"`
	Summary string `json:"summary,omitempty"`
}

// StreamEvent is one item emitted while a reply is generated: either a text
// Delta, or a Tool activity update. Exactly one field is set per event.
type StreamEvent struct {
	Delta string
	Tool  *ToolEvent
}

// ToolEvent reports tool-call activity for the analysis view. Phase is "start"
// (Name/Args set) or "done" (Name + Summary/Error set).
type ToolEvent struct {
	Phase   string
	Name    string
	Args    string
	Summary string
	Error   string
}

// ThreadSummary is the lightweight projection used by the session list.
type ThreadSummary struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	UpdatedAt    time.Time `json:"updatedAt"`
	MessageCount int       `json:"messageCount"`
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
	// ToolName/ToolArgs carry a confirmation-gated write tool call. When set,
	// confirming the action executes this tool against the API.
	ToolName    string    `json:"toolName,omitempty"`
	ToolArgs    string    `json:"toolArgs,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	ConfirmedAt time.Time `json:"confirmedAt,omitempty"`
	ConfirmedBy string    `json:"confirmedBy,omitempty"`
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
