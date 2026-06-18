package llm

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// Store holds the configured providers. It mirrors the lifecycle of
// settings.Store: an in-memory copy guarded by a mutex, persisted to a JSON state
// file when a state directory is configured.
type Store struct {
	mu        sync.RWMutex
	now       func() time.Time
	nextID    int
	providers []Provider
	statePath string
}

type snapshot struct {
	NextID    int        `json:"nextId"`
	Providers []Provider `json:"providers"`
}

// NewStore returns an empty in-memory store (no persistence).
func NewStore() *Store {
	return &Store{now: time.Now}
}

// NewStoreWithStateDir returns a store backed by $stateDir/llm_providers.json,
// loading any previously persisted providers.
func NewStoreWithStateDir(stateDir string) (*Store, error) {
	store := NewStore()
	if stateDir == "" {
		return store, nil
	}
	store.statePath = filepath.Join(stateDir, "llm_providers.json")
	var persisted snapshot
	if err := state.LoadJSON(store.statePath, &persisted); err != nil {
		return nil, err
	}
	store.nextID = persisted.NextID
	store.providers = persisted.Providers
	return store, nil
}

// List returns masked views of every provider.
func (s *Store) List() []ProviderView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	views := make([]ProviderView, 0, len(s.providers))
	for _, p := range s.providers {
		views = append(views, p.toView())
	}
	return views
}

// Get returns the full provider (including its API key) by id.
func (s *Store) Get(id string) (Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.providers {
		if p.ID == id {
			return p, nil
		}
	}
	return Provider{}, fmt.Errorf("llm provider not found: %s", id)
}

// Default returns the provider marked default, or the single enabled provider
// when exactly one exists. It is the provider the assistant binds to.
func (s *Store) Default() (Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var enabled []Provider
	for _, p := range s.providers {
		if !p.Enabled {
			continue
		}
		if p.IsDefault {
			return p, nil
		}
		enabled = append(enabled, p)
	}
	if len(enabled) == 1 {
		return enabled[0], nil
	}
	return Provider{}, fmt.Errorf("no default llm provider configured")
}

// Create adds a new provider from the given input and returns its masked view.
func (s *Store) Create(in ProviderInput) (ProviderView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextID++
	p := Provider{
		ID:        fmt.Sprintf("provider-%03d", s.nextID),
		Enabled:   true,
		CreatedAt: s.now().UTC(),
	}
	applyInput(&p, in)
	if err := validate(p); err != nil {
		s.nextID--
		return ProviderView{}, err
	}
	s.providers = append(s.providers, p)
	s.enforceSingleDefaultLocked(p.ID, p.IsDefault)
	if err := s.saveLocked(); err != nil {
		return ProviderView{}, err
	}
	return s.byIDLocked(p.ID).toView(), nil
}

// Update mutates an existing provider. Fields left nil in the input are
// preserved; an empty APIKey string also preserves the existing key.
func (s *Store) Update(id string, in ProviderInput) (ProviderView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.providers {
		if s.providers[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ProviderView{}, fmt.Errorf("llm provider not found: %s", id)
	}
	updated := s.providers[idx]
	applyInput(&updated, in)
	if err := validate(updated); err != nil {
		return ProviderView{}, err
	}
	s.providers[idx] = updated
	s.enforceSingleDefaultLocked(id, updated.IsDefault)
	if err := s.saveLocked(); err != nil {
		return ProviderView{}, err
	}
	return s.byIDLocked(id).toView(), nil
}

// Delete removes a provider by id.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.providers {
		if s.providers[i].ID == id {
			s.providers = append(s.providers[:i], s.providers[i+1:]...)
			return s.saveLocked()
		}
	}
	return fmt.Errorf("llm provider not found: %s", id)
}

// enforceSingleDefaultLocked clears IsDefault on every provider except keepID
// when makeDefault is true. Callers hold s.mu.
func (s *Store) enforceSingleDefaultLocked(keepID string, makeDefault bool) {
	if !makeDefault {
		return
	}
	for i := range s.providers {
		s.providers[i].IsDefault = s.providers[i].ID == keepID
	}
}

func (s *Store) byIDLocked(id string) Provider {
	for _, p := range s.providers {
		if p.ID == id {
			return p
		}
	}
	return Provider{}
}

func (s *Store) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{NextID: s.nextID, Providers: s.providers})
}

// applyInput copies set fields from in onto p. APIKey is only overwritten when a
// non-empty value is supplied.
func applyInput(p *Provider, in ProviderInput) {
	if in.Name != nil {
		p.Name = strings.TrimSpace(*in.Name)
	}
	if in.Kind != nil {
		p.Kind = *in.Kind
	}
	if in.BaseURL != nil {
		p.BaseURL = strings.TrimRight(strings.TrimSpace(*in.BaseURL), "/")
	}
	if in.APIKey != nil && strings.TrimSpace(*in.APIKey) != "" {
		p.APIKey = strings.TrimSpace(*in.APIKey)
	}
	if in.Model != nil {
		p.Model = strings.TrimSpace(*in.Model)
	}
	if in.Enabled != nil {
		p.Enabled = *in.Enabled
	}
	if in.IsDefault != nil {
		p.IsDefault = *in.IsDefault
	}
}

func validate(p Provider) error {
	if p.Name == "" {
		return fmt.Errorf("llm provider name is required")
	}
	switch p.Kind {
	case KindOpenAI, KindAnthropic, KindGemini:
	default:
		return fmt.Errorf("llm provider kind must be one of openai, anthropic, gemini")
	}
	if p.Model == "" {
		return fmt.Errorf("llm provider model is required")
	}
	return nil
}
