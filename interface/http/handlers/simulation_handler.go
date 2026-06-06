package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"
)

// SimulationHandler handles simulation settings endpoints
type SimulationHandler struct {
	simulationService *services.SimulationService
}

// NewSimulationHandler creates a new simulation handler
func NewSimulationHandler(simulationService *services.SimulationService) *SimulationHandler {
	return &SimulationHandler{
		simulationService: simulationService,
	}
}

// GetSimulationSettings returns all simulation settings
func (h *SimulationHandler) GetSimulationSettings(w http.ResponseWriter, r *http.Request) {
	settings := h.simulationService.GetSettings()
	response := dto.SimulationSettingsResponse{
		ManualMode:       settings.GetManualMode(),
		Delays:           settings.GetDelays(),
		FailureRates:     settings.GetFailureRates(),
		ErrorInjection:   settings.GetErrorInjection(),
		ChargingBehavior: settings.GetChargingBehavior(),
		Transaction:      settings.GetTransaction(),
	}
	writeJSON(w, http.StatusOK, response)
}

// SetSimulationSettings updates all simulation settings
func (h *SimulationHandler) SetSimulationSettings(w http.ResponseWriter, r *http.Request) {
	var req dto.SimulationSettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.simulationService.SetManualMode(req.ManualMode)
	h.simulationService.SetDelays(req.Delays)
	h.simulationService.SetFailureRates(req.FailureRates)
	h.simulationService.SetErrorInjection(req.ErrorInjection)
	h.simulationService.SetChargingBehavior(req.ChargingBehavior)
	h.simulationService.SetTransactionSettings(req.Transaction)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Simulation settings updated",
	})
}

// GetDelays returns delay settings
func (h *SimulationHandler) GetDelays(w http.ResponseWriter, r *http.Request) {
	delays := h.simulationService.GetDelays()
	writeJSON(w, http.StatusOK, delays)
}

// SetDelays updates delay settings
func (h *SimulationHandler) SetDelays(w http.ResponseWriter, r *http.Request) {
	var req dto.DelaySettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.simulationService.SetDelays(entities.DelaySettings{
		BootNotificationDelayMs:   req.BootNotificationDelayMs,
		AuthorizationDelayMs:      req.AuthorizationDelayMs,
		StartTransactionDelayMs:   req.StartTransactionDelayMs,
		StopTransactionDelayMs:    req.StopTransactionDelayMs,
		StatusNotificationDelayMs: req.StatusNotificationDelayMs,
		HeartbeatDelayMs:          req.HeartbeatDelayMs,
		MeterValuesDelayMs:        req.MeterValuesDelayMs,
		RemoteStartDelayMs:        req.RemoteStartDelayMs,
		RemoteStopDelayMs:         req.RemoteStopDelayMs,
	})

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Delay settings updated",
	})
}

// GetFailureRates returns failure rate settings
func (h *SimulationHandler) GetFailureRates(w http.ResponseWriter, r *http.Request) {
	rates := h.simulationService.GetFailureRates()
	writeJSON(w, http.StatusOK, rates)
}

// SetFailureRates updates failure rate settings
func (h *SimulationHandler) SetFailureRates(w http.ResponseWriter, r *http.Request) {
	var req dto.FailureRatesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.simulationService.SetFailureRates(entities.FailureRateSettings{
		AuthorizationFailureRate:    req.AuthorizationFailureRate,
		StartTransactionFailureRate: req.StartTransactionFailureRate,
		StopTransactionFailureRate:  req.StopTransactionFailureRate,
		MeterValuesFailureRate:      req.MeterValuesFailureRate,
		HeartbeatFailureRate:        req.HeartbeatFailureRate,
	})

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Failure rate settings updated",
	})
}

// GetErrorInjection returns error injection settings
func (h *SimulationHandler) GetErrorInjection(w http.ResponseWriter, r *http.Request) {
	settings := h.simulationService.GetErrorInjection()
	writeJSON(w, http.StatusOK, settings)
}

// SetErrorInjection updates error injection settings
func (h *SimulationHandler) SetErrorInjection(w http.ResponseWriter, r *http.Request) {
	var req dto.ErrorInjectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.simulationService.SetErrorInjection(entities.ErrorInjectionSettings{
		Enabled:              req.Enabled,
		InternalErrorRate:    req.InternalErrorRate,
		NotImplementedRate:   req.NotImplementedRate,
		NotSupportedRate:     req.NotSupportedRate,
		MessageTimeoutRate:   req.MessageTimeoutRate,
		MalformedMessageRate: req.MalformedMessageRate,
	})

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Error injection settings updated",
	})
}

// GetTransactionSettings returns transaction settings
func (h *SimulationHandler) GetTransactionSettings(w http.ResponseWriter, r *http.Request) {
	tx := h.simulationService.GetTransactionSettings()
	writeJSON(w, http.StatusOK, tx)
}

// SetTransactionSettings updates transaction settings
func (h *SimulationHandler) SetTransactionSettings(w http.ResponseWriter, r *http.Request) {
	var req dto.TransactionSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.simulationService.SetTransactionSettings(entities.TransactionSettings{
		MeterStepWh:          req.MeterStepWh,
		StartSOC:             req.StartSOC,
		FinalSOC:             req.FinalSOC,
		FinalSOCBehavior:     req.FinalSOCBehavior,
		StopDelaySec:         req.StopDelaySec,
		PreparingDurationSec: req.PreparingDurationSec,
	})

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Transaction settings updated",
	})
}

// ResetSimulation resets simulation settings to defaults
func (h *SimulationHandler) ResetSimulation(w http.ResponseWriter, r *http.Request) {
	h.simulationService.Reset()
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Simulation settings reset to defaults",
	})
}

// GetManualMode returns the manual mode status
func (h *SimulationHandler) GetManualMode(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": h.simulationService.IsManualMode()})
}

// SetManualMode toggles manual mode
func (h *SimulationHandler) SetManualMode(w http.ResponseWriter, r *http.Request) {
	var req dto.ManualModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.simulationService.SetManualMode(req.Enabled)
	status := "disabled"
	if req.Enabled {
		status = "enabled"
	}
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: fmt.Sprintf("Manual mode %s", status),
	})
}
