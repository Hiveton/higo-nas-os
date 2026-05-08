package audit

import (
	"fmt"
	"sync"
	"time"
)

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type AuditResult string

const (
	ResultAllowed    AuditResult = "allowed"
	ResultDenied     AuditResult = "denied"
	ResultConfirmed  AuditResult = "confirmed"
	ResultBlocked    AuditResult = "blocked"
	ResultFailed     AuditResult = "failed"
	ResultRolledBack AuditResult = "rolled_back"
)

type RiskActionStatus string

const (
	RiskActionPending   RiskActionStatus = "pending"
	RiskActionConfirmed RiskActionStatus = "confirmed"
	RiskActionBlocked   RiskActionStatus = "blocked"
)

type RollbackStatus string

const (
	RollbackPending   RollbackStatus = "pending"
	RollbackSucceeded RollbackStatus = "succeeded"
	RollbackFailed    RollbackStatus = "failed"
)

type AuditEvent struct {
	ID                  string
	Time                time.Time
	RequestID           string
	ActorID             string
	DeviceID            string
	SessionID           string
	SourceIP            string
	Domain              string
	Action              string
	TargetIDs           []string
	TargetScope         string
	BeforeSummary       string
	AfterSummary        string
	Risk                RiskLevel
	ConfirmationID      string
	PolicyDecision      string
	RollbackOperationID string
	Result              AuditResult
}

type RiskAction struct {
	ID                 string
	ActorID            string
	Action             string
	TargetScope        string
	AffectedItemCount  int
	Risk               RiskLevel
	ImpactSummary      string
	RequiredPermission string
	Status             RiskActionStatus
	CreatedAt          time.Time
	ConfirmedAt        time.Time
	ConfirmedBy        string
	BlockedAt          time.Time
	BlockedBy          string
	BlockReason        string
	Rollback           *RollbackOperation
}

type RollbackOperation struct {
	ID       string
	Type     string
	ActorID  string
	TargetID string
	Before   string
	After    string
	Status   RollbackStatus
	Result   string
	MarkedAt time.Time
}

type Store struct {
	mu           sync.RWMutex
	nextEvent    int
	nextAction   int
	nextRollback int
	events       []AuditEvent
	actions      map[string]RiskAction
}

func NewStore() *Store {
	return &Store{actions: make(map[string]RiskAction)}
}

func (s *Store) Append(event AuditEvent) AuditEvent {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextEvent++
	if event.ID == "" {
		event.ID = fmt.Sprintf("audit-event-%d", s.nextEvent)
	}
	if event.Time.IsZero() {
		event.Time = time.Now().UTC()
	}
	event.TargetIDs = append([]string(nil), event.TargetIDs...)
	s.events = append(s.events, event)
	return event
}

func (s *Store) List() []AuditEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]AuditEvent, len(s.events))
	for i := range s.events {
		events[i] = s.events[i]
		events[i].TargetIDs = append([]string(nil), s.events[i].TargetIDs...)
	}
	return events
}

func (s *Store) CreateRiskAction(action RiskAction) RiskAction {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextAction++
	if action.ID == "" {
		action.ID = fmt.Sprintf("risk-action-%d", s.nextAction)
	}
	if action.Status == "" {
		action.Status = RiskActionPending
	}
	if action.CreatedAt.IsZero() {
		action.CreatedAt = time.Now().UTC()
	}
	if action.Rollback != nil {
		s.nextRollback++
		rollback := *action.Rollback
		if rollback.ID == "" {
			rollback.ID = fmt.Sprintf("rollback-%d", s.nextRollback)
		}
		if rollback.ActorID == "" {
			rollback.ActorID = action.ActorID
		}
		if rollback.Status == "" {
			rollback.Status = RollbackPending
		}
		action.Rollback = &rollback
	}
	s.actions[action.ID] = action
	return cloneRiskAction(action)
}

func (s *Store) ConfirmRiskAction(id, confirmedBy string, at time.Time) (RiskAction, bool) {
	if at.IsZero() {
		at = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	action, ok := s.actions[id]
	if !ok || action.Status != RiskActionPending {
		return RiskAction{}, false
	}
	action.Status = RiskActionConfirmed
	action.ConfirmedBy = confirmedBy
	action.ConfirmedAt = at
	s.actions[id] = action
	return cloneRiskAction(action), true
}

func (s *Store) BlockRiskAction(id, blockedBy, reason string) (RiskAction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	action, ok := s.actions[id]
	if !ok || action.Status != RiskActionPending {
		return RiskAction{}, false
	}
	action.Status = RiskActionBlocked
	action.BlockedBy = blockedBy
	action.BlockReason = reason
	action.BlockedAt = time.Now().UTC()
	s.actions[id] = action
	return cloneRiskAction(action), true
}

func (s *Store) MarkRollback(actionID string, status RollbackStatus, result string) (RiskAction, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	action, ok := s.actions[actionID]
	if !ok || action.Rollback == nil {
		return RiskAction{}, false
	}
	rollback := *action.Rollback
	rollback.Status = status
	rollback.Result = result
	rollback.MarkedAt = time.Now().UTC()
	action.Rollback = &rollback
	s.actions[actionID] = action
	return cloneRiskAction(action), true
}

func cloneRiskAction(action RiskAction) RiskAction {
	if action.Rollback != nil {
		rollback := *action.Rollback
		action.Rollback = &rollback
	}
	return action
}
