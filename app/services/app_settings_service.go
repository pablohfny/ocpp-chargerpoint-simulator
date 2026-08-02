package services

import (
	"sync"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/infrastructure/persistence"
)

// AppSettingsService owns the runtime-editable settings and their persistence.
//
// Precedence is fixed: the environment (and flags) provide the bootstrap
// defaults handed to NewAppSettingsService, and Load overlays whatever the
// persisted file holds on top of them. So a value saved from the Config tab
// survives restarts even when the env var still carries the old value.
type AppSettingsService struct {
	mu       sync.RWMutex
	settings entities.AppSettings
	store    *persistence.AppSettingsStore
}

// NewAppSettingsService creates the service seeded with the bootstrap defaults.
// The store may be nil, in which case settings live in memory only.
func NewAppSettingsService(defaults entities.AppSettings, store *persistence.AppSettingsStore) *AppSettingsService {
	defaults.Normalize()
	return &AppSettingsService{settings: defaults, store: store}
}

// Load overlays the persisted settings on top of the bootstrap defaults.
func (s *AppSettingsService) Load() error {
	if s.store == nil {
		return nil
	}

	persisted, err := s.store.Load()
	if err != nil {
		return err
	}
	persisted.Normalize()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings.MergeOverrides(persisted)
	return nil
}

// Get returns a copy of the current settings.
func (s *AppSettingsService) Get() entities.AppSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// Update validates, stores and persists a full settings replacement.
func (s *AppSettingsService) Update(settings entities.AppSettings) (entities.AppSettings, error) {
	settings.Normalize()
	if err := settings.Validate(); err != nil {
		return entities.AppSettings{}, err
	}

	s.mu.Lock()
	s.settings = settings
	s.mu.Unlock()

	if s.store != nil {
		if err := s.store.Save(settings); err != nil {
			return entities.AppSettings{}, err
		}
	}
	return settings, nil
}

// BatteryCapacityKWh returns the virtual battery size used to derive the
// battery percentage reported for a connector.
func (s *AppSettingsService) BatteryCapacityKWh() float64 {
	return s.Get().BatteryCapacityKWh
}

// StorePath returns the settings file path, or an empty string when settings
// are memory only.
func (s *AppSettingsService) StorePath() string {
	if s.store == nil {
		return ""
	}
	return s.store.Path()
}
