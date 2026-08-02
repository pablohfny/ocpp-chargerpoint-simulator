package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"

	"EV-Client-Simulator/app/domain/entities"
)

// AppSettingsStore persists the UI-editable settings as a JSON object on disk.
type AppSettingsStore struct {
	path string
}

// NewAppSettingsStore creates a store backed by the given file path.
func NewAppSettingsStore(path string) *AppSettingsStore {
	return &AppSettingsStore{path: path}
}

// Path returns the backing file path, shown read-only in the Config tab.
func (s *AppSettingsStore) Path() string {
	return s.path
}

// Load reads the persisted settings. A missing or empty file yields zero
// values, which the service treats as "no override, keep the env default".
func (s *AppSettingsStore) Load() (entities.AppSettings, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return entities.AppSettings{}, nil
	}
	if err != nil {
		return entities.AppSettings{}, err
	}

	if len(data) == 0 {
		return entities.AppSettings{}, nil
	}

	var settings entities.AppSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return entities.AppSettings{}, err
	}
	return settings, nil
}

// Save writes the settings, creating the parent directory if needed. The write
// is atomic so a crash cannot leave a truncated settings file.
func (s *AppSettingsStore) Save(settings entities.AppSettings) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
