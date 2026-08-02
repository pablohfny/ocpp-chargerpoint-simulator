package handlers

import (
	"encoding/json"
	"net/http"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"
)

// SettingsHandler serves the Config tab: the editable settings plus the values
// that are fixed for the lifetime of the process.
type SettingsHandler struct {
	settings *services.AppSettingsService
	runtime  dto.RuntimeInfo
}

// NewSettingsHandler creates the settings handler.
func NewSettingsHandler(settings *services.AppSettingsService, runtime dto.RuntimeInfo) *SettingsHandler {
	runtime.SettingsPath = settings.StorePath()
	return &SettingsHandler{settings: settings, runtime: runtime}
}

// GetSettings returns the current settings and the read-only runtime values.
func (h *SettingsHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, dto.SettingsResponse{
		Settings: h.settings.Get(),
		Runtime:  h.runtime,
	})
}

// UpdateSettings validates and persists a full settings replacement.
func (h *SettingsHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var settings entities.AppSettings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	updated, err := h.settings.Update(settings)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_SETTINGS", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.SettingsResponse{
		Settings: updated,
		Runtime:  h.runtime,
	})
}
