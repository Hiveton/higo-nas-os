// Package activity is the user-activity log: meaningful frontend actions and
// page (window) visits, buffered client-side and synced here for cross-device,
// restart-safe history. Persisted via the JSON state layer; pruned to a
// configurable cap.
package activity

import (
	"path/filepath"
	"sort"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// Entry is one recorded activity (an action or a page visit).
type Entry struct {
	ID       int               `json:"id"`
	At       time.Time         `json:"at"`
	Actor    string            `json:"actor,omitempty"`
	Type     string            `json:"type"`               // "action" | "page"
	Category string            `json:"category,omitempty"` // window/domain id
	Action   string            `json:"action"`             // e.g. open-window, run-task, save-settings
	Target   string            `json:"target,omitempty"`
	Detail   string            `json:"detail,omitempty"`
	Meta     map[string]string `json:"meta,omitempty"`
}

const defaultMaxEntries = 2000

// Service owns the activity ledger.
type Service struct {
	mu         sync.RWMutex
	seq        int
	entries    []Entry
	statePath  string
	maxEntries int
}

type snapshot struct {
	Seq     int     `json:"seq"`
	Entries []Entry `json:"entries"`
}

// NewService builds an in-memory service (tests / no persistence).
func NewService() *Service {
	return &Service{maxEntries: defaultMaxEntries}
}

// NewServiceWithStateDir builds a service persisted to {stateDir}/activity.json.
func NewServiceWithStateDir(stateDir string) (*Service, error) {
	s := NewService()
	if stateDir == "" {
		return s, nil
	}
	s.statePath = filepath.Join(stateDir, "activity.json")
	var persisted snapshot
	if err := state.LoadJSON(s.statePath, &persisted); err != nil {
		return nil, err
	}
	s.seq = persisted.Seq
	s.entries = persisted.Entries
	if s.seq < len(s.entries) {
		s.seq = len(s.entries)
	}
	return s, nil
}

// Append records a batch of entries (assigning id + timestamp when absent),
// prunes to the cap, persists, and returns the stored entries.
func (s *Service) Append(in []Entry) []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	out := make([]Entry, 0, len(in))
	for _, e := range in {
		s.seq++
		e.ID = s.seq
		if e.At.IsZero() {
			e.At = now
		}
		if e.Type == "" {
			e.Type = "action"
		}
		s.entries = append(s.entries, e)
		out = append(out, e)
	}
	s.pruneLocked()
	s.persistLocked()
	return out
}

// List returns entries newest-first, optionally filtered by type, limited.
func (s *Service) List(typ string, limit int) []Entry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0, len(s.entries))
	for _, e := range s.entries {
		if typ != "" && e.Type != typ {
			continue
		}
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}

// Clear removes all entries.
func (s *Service) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
	s.persistLocked()
}

// SetRetention adjusts the max-entry cap (and prunes). Non-positive is ignored.
func (s *Service) SetRetention(maxEntries int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if maxEntries <= 0 {
		return
	}
	s.maxEntries = maxEntries
	s.pruneLocked()
	s.persistLocked()
}

func (s *Service) pruneLocked() {
	if s.maxEntries > 0 && len(s.entries) > s.maxEntries {
		s.entries = append([]Entry(nil), s.entries[len(s.entries)-s.maxEntries:]...)
	}
}

func (s *Service) persistLocked() {
	if s.statePath == "" {
		return
	}
	_ = state.SaveJSON(s.statePath, snapshot{Seq: s.seq, Entries: s.entries})
}
