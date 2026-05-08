package settings

import (
	"fmt"
	"path/filepath"
	"sync"

	"higoos/server-go/internal/state"
)

type ModelMode string

const (
	ModelModeFamilyHybrid    ModelMode = "family_hybrid"
	ModelModeProvider        ModelMode = "provider"
	ModelModeEnterpriseLocal ModelMode = "enterprise_local"
)

type Settings struct {
	Model   ModelPolicy   `json:"model"`
	Privacy PrivacyPolicy `json:"privacy"`
}

type ModelPolicy struct {
	Mode         ModelMode `json:"mode"`
	Provider     string    `json:"provider"`
	LocalModel   string    `json:"localModel"`
	CloudModel   string    `json:"cloudModel"`
	CloudEnabled bool      `json:"cloudEnabled"`
}

type PrivacyPolicy struct {
	SensitiveDataLocalOnly bool `json:"sensitiveDataLocalOnly"`
	AuditRetentionDays     int  `json:"auditRetentionDays"`
}

type Store struct {
	mu        sync.RWMutex
	settings  Settings
	defaults  Settings
	statePath string
}

func NewStore() *Store {
	defaults := DefaultSettings()
	return &Store{settings: defaults, defaults: defaults}
}

func NewStoreWithStateDir(stateDir string) (*Store, error) {
	store := NewStore()
	if stateDir == "" {
		return store, nil
	}
	store.statePath = filepath.Join(stateDir, "settings.json")
	var persisted Settings
	if err := state.LoadJSON(store.statePath, &persisted); err != nil {
		return nil, err
	}
	if !isZeroSettings(persisted) {
		normalized, err := normalize(persisted)
		if err != nil {
			return nil, err
		}
		store.settings = normalized
	}
	return store, nil
}

func DefaultSettings() Settings {
	return Settings{
		Model: ModelPolicy{
			Mode:         ModelModeFamilyHybrid,
			Provider:     "local",
			LocalModel:   "higo-local",
			CloudEnabled: false,
		},
		Privacy: PrivacyPolicy{
			SensitiveDataLocalOnly: true,
			AuditRetentionDays:     90,
		},
	}
}

func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Store) Update(next Settings) (Settings, error) {
	normalized, err := normalize(next)
	if err != nil {
		return Settings{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.settings = normalized
	if err := s.saveLocked(); err != nil {
		return Settings{}, err
	}
	return s.settings, nil
}

func (s *Store) RestoreDefaults() (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.settings = s.defaults
	if err := s.saveLocked(); err != nil {
		return Settings{}, err
	}
	return s.settings, nil
}

func (s *Store) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, s.settings)
}

func normalize(settings Settings) (Settings, error) {
	if settings.Model.Mode == "" {
		settings.Model.Mode = ModelModeFamilyHybrid
	}
	if settings.Privacy.AuditRetentionDays == 0 {
		settings.Privacy.AuditRetentionDays = 90
	}
	if settings.Privacy.AuditRetentionDays < 1 {
		return Settings{}, fmt.Errorf("settings: audit retention days must be positive")
	}
	if !settings.Privacy.SensitiveDataLocalOnly {
		return Settings{}, fmt.Errorf("settings: sensitive data must remain local-only")
	}
	if settings.Model.Mode == ModelModeEnterpriseLocal {
		settings.Model.CloudEnabled = false
	}
	return settings, nil
}

func isZeroSettings(settings Settings) bool {
	return settings.Model.Mode == "" &&
		settings.Model.Provider == "" &&
		settings.Model.LocalModel == "" &&
		settings.Model.CloudModel == "" &&
		!settings.Model.CloudEnabled &&
		!settings.Privacy.SensitiveDataLocalOnly &&
		settings.Privacy.AuditRetentionDays == 0
}
