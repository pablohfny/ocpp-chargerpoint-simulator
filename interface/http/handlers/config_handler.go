package handlers

import (
	"encoding/json"
	"net/http"

	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"

	"github.com/go-chi/chi/v5"
)

// ConfigHandler handles configuration-related endpoints
type ConfigHandler struct {
	configService *services.ConfigurationService
}

// NewConfigHandler creates a new config handler
func NewConfigHandler(configService *services.ConfigurationService) *ConfigHandler {
	return &ConfigHandler{
		configService: configService,
	}
}

// GetAllConfig returns all OCPP configuration keys
func (h *ConfigHandler) GetAllConfig(w http.ResponseWriter, r *http.Request) {
	keys := h.configService.GetAllKeys()
	response := make([]dto.ConfigKeyResponse, len(keys))
	for i, key := range keys {
		response[i] = dto.ConfigKeyResponse{
			Key:      key.Key,
			Value:    key.Value,
			Readonly: key.Readonly,
		}
	}
	writeJSON(w, http.StatusOK, response)
}

// GetConfig returns a specific configuration key
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	keyName := chi.URLParam(r, "key")
	key, exists := h.configService.GetKey(keyName)
	if !exists {
		writeError(w, http.StatusNotFound, "CONFIG_KEY_NOT_FOUND", "Configuration key not found")
		return
	}

	writeJSON(w, http.StatusOK, dto.ConfigKeyResponse{
		Key:      key.Key,
		Value:    key.Value,
		Readonly: key.Readonly,
	})
}

// SetConfig updates a configuration key
func (h *ConfigHandler) SetConfig(w http.ResponseWriter, r *http.Request) {
	keyName := chi.URLParam(r, "key")

	var req dto.ChangeConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	status := h.configService.ChangeConfiguration(keyName, req.Value)

	switch status {
	case "Accepted":
		writeJSON(w, http.StatusOK, dto.ActionResponse{
			Success: true,
			Message: "Configuration updated",
		})
	case "Rejected":
		writeError(w, http.StatusForbidden, "READONLY_CONFIGURATION", "Configuration key is read-only")
	case "NotSupported":
		writeError(w, http.StatusNotFound, "CONFIG_KEY_NOT_FOUND", "Configuration key not found")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Unexpected error")
	}
}
