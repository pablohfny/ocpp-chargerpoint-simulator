package services

import (
	"path/filepath"
	"testing"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/infrastructure/persistence"
)

// bootstrapSettings stands in for what the environment provides at boot.
var bootstrapSettings = entities.AppSettings{
	OCPIBaseURL:        "https://ocpi-dev.nucharge.com.br",
	PublicBaseURL:      "http://localhost:8080",
	DefaultLocationID:  "LOC-ENV",
	DefaultEvseUID:     "EVSE-ENV",
	DefaultConnectorID: "1",
	BatteryCapacityKWh: 60,
}

// TestSettingsPersistedFileWinsOverEnv is the whole point of the Config tab:
// what you save survives a restart even when the env var still says otherwise.
func TestSettingsPersistedFileWinsOverEnv(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "settings.json")

	originalService := NewAppSettingsService(bootstrapSettings, persistence.NewAppSettingsStore(inputPath))
	if _, err := originalService.Update(entities.AppSettings{
		OCPIBaseURL:        "https://ocpi-vps.nucharge.com.br",
		PublicBaseURL:      "https://sim-dev.nucharge.com.br",
		DefaultLocationID:  "LOC-UI",
		DefaultEvseUID:     "EVSE-UI",
		DefaultConnectorID: "2",
		BatteryCapacityKWh: 82,
	}); err != nil {
		t.Fatalf("Update() returned an error: %v", err)
	}

	restartedService := NewAppSettingsService(bootstrapSettings, persistence.NewAppSettingsStore(inputPath))
	if err := restartedService.Load(); err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	actualSettings := restartedService.Get()
	if actualSettings.OCPIBaseURL != "https://ocpi-vps.nucharge.com.br" {
		t.Errorf("OCPIBaseURL = %q, expected the persisted value", actualSettings.OCPIBaseURL)
	}
	if actualSettings.DefaultLocationID != "LOC-UI" || actualSettings.DefaultEvseUID != "EVSE-UI" {
		t.Errorf("defaults = %+v, expected the persisted ones", actualSettings)
	}
	if actualSettings.BatteryCapacityKWh != 82 {
		t.Errorf("BatteryCapacityKWh = %v, expected 82", actualSettings.BatteryCapacityKWh)
	}
}

// TestSettingsKeepEnvDefaultsForBlankFields keeps a half-filled file from
// wiping values the environment still provides.
func TestSettingsKeepEnvDefaultsForBlankFields(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "settings.json")
	if err := persistence.NewAppSettingsStore(inputPath).Save(entities.AppSettings{
		DefaultLocationID: "LOC-UI",
	}); err != nil {
		t.Fatalf("Save() returned an error: %v", err)
	}

	actualService := NewAppSettingsService(bootstrapSettings, persistence.NewAppSettingsStore(inputPath))
	if err := actualService.Load(); err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	actualSettings := actualService.Get()
	if actualSettings.DefaultLocationID != "LOC-UI" {
		t.Errorf("DefaultLocationID = %q, expected the persisted override", actualSettings.DefaultLocationID)
	}
	if actualSettings.OCPIBaseURL != bootstrapSettings.OCPIBaseURL {
		t.Errorf("OCPIBaseURL = %q, expected the env bootstrap to survive", actualSettings.OCPIBaseURL)
	}
	if actualSettings.BatteryCapacityKWh != 60 {
		t.Errorf("BatteryCapacityKWh = %v, expected the env bootstrap to survive", actualSettings.BatteryCapacityKWh)
	}
}

func TestSettingsValidation(t *testing.T) {
	tests := []struct {
		name          string
		inputSettings entities.AppSettings
		expectedError bool
	}{
		{name: "complete settings", inputSettings: bootstrapSettings},
		{
			name:          "missing ocpi base url",
			inputSettings: entities.AppSettings{PublicBaseURL: "http://localhost:8080", BatteryCapacityKWh: 60},
			expectedError: true,
		},
		{
			name:          "missing public base url",
			inputSettings: entities.AppSettings{OCPIBaseURL: "https://x", BatteryCapacityKWh: 60},
			expectedError: true,
		},
		{
			name:          "zero battery capacity",
			inputSettings: entities.AppSettings{OCPIBaseURL: "https://x", PublicBaseURL: "https://y"},
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualService := NewAppSettingsService(bootstrapSettings, nil)

			_, err := actualService.Update(test.inputSettings)

			if (err != nil) != test.expectedError {
				t.Errorf("Update() error = %v, expected an error: %v", err, test.expectedError)
			}
		})
	}
}

// TestSettingsUpdateTrimsTrailingSlash keeps the URLs safe to concatenate.
func TestSettingsUpdateTrimsTrailingSlash(t *testing.T) {
	actualService := NewAppSettingsService(bootstrapSettings, nil)

	actualSettings, err := actualService.Update(entities.AppSettings{
		OCPIBaseURL:        "https://ocpi-dev.nucharge.com.br/",
		PublicBaseURL:      "https://sim-dev.nucharge.com.br/",
		BatteryCapacityKWh: 60,
	})
	if err != nil {
		t.Fatalf("Update() returned an error: %v", err)
	}

	if actualSettings.OCPIBaseURL != "https://ocpi-dev.nucharge.com.br" {
		t.Errorf("OCPIBaseURL = %q, expected the trailing slash to be trimmed", actualSettings.OCPIBaseURL)
	}
	if actualSettings.PublicBaseURL != "https://sim-dev.nucharge.com.br" {
		t.Errorf("PublicBaseURL = %q, expected the trailing slash to be trimmed", actualSettings.PublicBaseURL)
	}
}
