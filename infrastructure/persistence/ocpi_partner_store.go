package persistence

import (
	"encoding/json"
	"os"
	"path/filepath"

	"EV-Client-Simulator/app/domain/entities"
)

// OCPIPartnerStore persists partner profiles as a JSON array on disk.
type OCPIPartnerStore struct {
	path string
}

// NewOCPIPartnerStore creates a store backed by the given file path.
func NewOCPIPartnerStore(path string) *OCPIPartnerStore {
	return &OCPIPartnerStore{path: path}
}

// Path returns the backing file path.
func (s *OCPIPartnerStore) Path() string {
	return s.path
}

// Load reads all partner profiles. A missing file yields an empty slice.
func (s *OCPIPartnerStore) Load() ([]entities.OCPIPartner, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return []entities.OCPIPartner{}, nil
	}
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return []entities.OCPIPartner{}, nil
	}

	var partners []entities.OCPIPartner
	if err := json.Unmarshal(data, &partners); err != nil {
		return nil, err
	}
	return partners, nil
}

// Save writes all partner profiles, creating the parent directory if needed.
// The write is atomic so a crash cannot leave a truncated profile file.
func (s *OCPIPartnerStore) Save(partners []entities.OCPIPartner) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(partners, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
