package entities

import (
	"errors"
	"strings"
)

// AppSettings holds every runtime-editable setting of the simulator. Environment
// variables only bootstrap the defaults: once a settings file exists, it wins.
type AppSettings struct {
	// OCPIBaseURL is the platform base URL partners command by default.
	OCPIBaseURL string `json:"ocpiBaseUrl"`
	// PublicBaseURL is this simulator's own public URL, handed to the platform
	// as the response_url and receiver base.
	PublicBaseURL string `json:"publicBaseUrl"`
	// DefaultLocationID/DefaultEvseUID prefill the START_SESSION context.
	DefaultLocationID string `json:"defaultLocationId"`
	DefaultEvseUID    string `json:"defaultEvseUid"`
	// DefaultConnectorID is the OCPI connector id used in START_SESSION.
	DefaultConnectorID string `json:"defaultConnectorId"`
	// BatteryCapacityKWh sizes the virtual EV battery behind the reported
	// battery percentage.
	BatteryCapacityKWh float64 `json:"batteryCapacityKwh"`
}

// Normalize trims the URLs so they concatenate cleanly.
func (s *AppSettings) Normalize() {
	s.OCPIBaseURL = strings.TrimRight(strings.TrimSpace(s.OCPIBaseURL), "/")
	s.PublicBaseURL = strings.TrimRight(strings.TrimSpace(s.PublicBaseURL), "/")
	s.DefaultLocationID = strings.TrimSpace(s.DefaultLocationID)
	s.DefaultEvseUID = strings.TrimSpace(s.DefaultEvseUID)
	s.DefaultConnectorID = strings.TrimSpace(s.DefaultConnectorID)
}

// Validate checks the settings a user can break from the UI.
func (s *AppSettings) Validate() error {
	if s.OCPIBaseURL == "" {
		return errors.New("ocpiBaseUrl is required")
	}
	if s.PublicBaseURL == "" {
		return errors.New("publicBaseUrl is required")
	}
	if s.BatteryCapacityKWh <= 0 {
		return errors.New("batteryCapacityKwh must be greater than zero")
	}
	return nil
}

// MergeOverrides overlays the non-empty fields of an override on top of these
// settings. It is how a persisted file wins over the environment defaults
// without a missing key wiping a default.
func (s *AppSettings) MergeOverrides(override AppSettings) {
	if override.OCPIBaseURL != "" {
		s.OCPIBaseURL = override.OCPIBaseURL
	}
	if override.PublicBaseURL != "" {
		s.PublicBaseURL = override.PublicBaseURL
	}
	if override.DefaultLocationID != "" {
		s.DefaultLocationID = override.DefaultLocationID
	}
	if override.DefaultEvseUID != "" {
		s.DefaultEvseUID = override.DefaultEvseUID
	}
	if override.DefaultConnectorID != "" {
		s.DefaultConnectorID = override.DefaultConnectorID
	}
	if override.BatteryCapacityKWh > 0 {
		s.BatteryCapacityKWh = override.BatteryCapacityKWh
	}
}
