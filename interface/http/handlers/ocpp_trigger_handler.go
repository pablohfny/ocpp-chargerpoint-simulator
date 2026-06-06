package handlers

import (
	"encoding/json"
	"net/http"

	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"
)

// OCPPTriggerHandler handles manual OCPP message triggering
type OCPPTriggerHandler struct {
	stationService *services.ChargerStationService
}

// NewOCPPTriggerHandler creates a new OCPP trigger handler
func NewOCPPTriggerHandler(stationService *services.ChargerStationService) *OCPPTriggerHandler {
	return &OCPPTriggerHandler{
		stationService: stationService,
	}
}

// TriggerBootNotification triggers a BootNotification message
func (h *OCPPTriggerHandler) TriggerBootNotification(w http.ResponseWriter, r *http.Request) {
	h.stationService.TriggerBootNotification()
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "BootNotification triggered",
	})
}

// TriggerStatusNotification triggers a StatusNotification message
func (h *OCPPTriggerHandler) TriggerStatusNotification(w http.ResponseWriter, r *http.Request) {
	var req dto.TriggerMessageRequest
	json.NewDecoder(r.Body).Decode(&req)

	if req.ConnectorID > 0 {
		h.stationService.TriggerStatusNotification(req.ConnectorID)
	} else {
		h.stationService.NotifyStatuses()
	}

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:     true,
		Message:     "StatusNotification triggered",
		ConnectorID: req.ConnectorID,
	})
}

// TriggerMeterValues triggers a MeterValues message
func (h *OCPPTriggerHandler) TriggerMeterValues(w http.ResponseWriter, r *http.Request) {
	var req dto.TriggerMessageRequest
	json.NewDecoder(r.Body).Decode(&req)

	if req.ConnectorID > 0 {
		h.stationService.TriggerMeterValues(req.ConnectorID)
		writeJSON(w, http.StatusOK, dto.ActionResponse{
			Success:     true,
			Message:     "MeterValues triggered",
			ConnectorID: req.ConnectorID,
		})
	} else {
		writeError(w, http.StatusBadRequest, "CONNECTOR_REQUIRED", "Connector ID is required for MeterValues")
	}
}

// TriggerHeartbeat triggers a Heartbeat message
func (h *OCPPTriggerHandler) TriggerHeartbeat(w http.ResponseWriter, r *http.Request) {
	h.stationService.TriggerHeartbeat()
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Heartbeat triggered",
	})
}

// TriggerAuthorize sends an Authorize message with a caller-chosen idTag.
func (h *OCPPTriggerHandler) TriggerAuthorize(w http.ResponseWriter, r *http.Request) {
	var req dto.TriggerAuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.IdTag == "" {
		writeError(w, http.StatusBadRequest, "MISSING_IDTAG", "idTag is required")
		return
	}

	connectorID := req.ConnectorID
	if connectorID == 0 {
		connectorID = 1
	}

	h.stationService.SendAuthorize(connectorID, req.IdTag)
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:     true,
		Message:     "Authorize sent (idTag: " + req.IdTag + ")",
		ConnectorID: connectorID,
	})
}

// TriggerStartTransaction sends a StartTransaction message with a
// caller-chosen idTag (and optional meterStart).
func (h *OCPPTriggerHandler) TriggerStartTransaction(w http.ResponseWriter, r *http.Request) {
	var req dto.TriggerStartTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.IdTag == "" {
		writeError(w, http.StatusBadRequest, "MISSING_IDTAG", "idTag is required")
		return
	}

	connectorID := req.ConnectorID
	if connectorID == 0 {
		connectorID = 1
	}

	h.stationService.SendStartTransaction(connectorID, req.IdTag, req.MeterStart)
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:     true,
		Message:     "StartTransaction sent (idTag: " + req.IdTag + ")",
		ConnectorID: connectorID,
	})
}
