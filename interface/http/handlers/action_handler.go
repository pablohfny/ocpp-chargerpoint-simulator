package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"

	"github.com/go-chi/chi/v5"
)

// ActionHandler handles action-related endpoints
type ActionHandler struct {
	stationService *services.ChargerStationService
	reconnectFunc  func() error
	disconnectFunc func() error
}

// NewActionHandler creates a new action handler
func NewActionHandler(stationService *services.ChargerStationService, reconnect, disconnect func() error) *ActionHandler {
	return &ActionHandler{
		stationService: stationService,
		reconnectFunc:  reconnect,
		disconnectFunc: disconnect,
	}
}

// PlugCable simulates plugging cable
func (h *ActionHandler) PlugCable(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.PlugCable(); err != nil {
		writeError(w, http.StatusBadRequest, "PLUG_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Cable plugged",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// UnplugCable simulates unplugging cable
func (h *ActionHandler) UnplugCable(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.UnplugCable(); err != nil {
		writeError(w, http.StatusBadRequest, "UNPLUG_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Cable unplugged",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// SetPreparing sets connector to Preparing status (simulates cable plugged before remote start)
func (h *ActionHandler) SetPreparing(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.StartRemoteTransaction(); err != nil {
		writeError(w, http.StatusBadRequest, "PREPARING_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Connector set to Preparing",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// StartCharging starts local charging
func (h *ActionHandler) StartCharging(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	var req dto.StartChargingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Use default idTag if not provided
		req.IdTag = "LOCAL_START"
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)

	// Simulate local start flow
	if err := point.StartRemoteTransaction(); err != nil {
		writeError(w, http.StatusBadRequest, "START_FAILED", err.Error())
		return
	}

	point.Authorize(req.IdTag)
	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Charging initiated, awaiting authorization",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// StopCharging stops charging
func (h *ActionHandler) StopCharging(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)

	if err := point.StopTransaction(); err != nil {
		writeError(w, http.StatusBadRequest, "STOP_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Charging stopped",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// Authorize simulates RFID authorization
func (h *ActionHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	var req dto.AuthorizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.IdTag = "RFID_TAG"
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	if err := point.Authorize(req.IdTag); err != nil {
		writeError(w, http.StatusBadRequest, "AUTHORIZE_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:     true,
		Message:     "Authorization simulated",
		ConnectorID: connectorID,
	})
}

// SetFault sets a connector fault
func (h *ActionHandler) SetFault(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	var req dto.SetFaultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	point.SetFault(entities.ChargePointErrorCode(req.ErrorCode), req.Info, req.VendorErrorCode)
	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Fault set",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// ClearFault clears a connector fault
func (h *ActionHandler) ClearFault(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.ClearFault(); err != nil {
		writeError(w, http.StatusBadRequest, "CLEAR_FAULT_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Fault cleared",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// SetUnavailable sets connector to unavailable
func (h *ActionHandler) SetUnavailable(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.SetUnavailable(); err != nil {
		writeError(w, http.StatusBadRequest, "SET_UNAVAILABLE_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Connector set to unavailable",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// SetAvailable sets connector to available
func (h *ActionHandler) SetAvailable(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.SetAvailable(); err != nil {
		writeError(w, http.StatusBadRequest, "SET_AVAILABLE_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Connector set to available",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// SetReservation sets a reservation on a connector
func (h *ActionHandler) SetReservation(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	var req dto.SetReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	expiry := time.Now().Add(time.Duration(req.ExpiryMinutes) * time.Minute)
	if err := point.SetReservation(req.ReservationID, req.IdTag, expiry); err != nil {
		writeError(w, http.StatusBadRequest, "RESERVATION_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Reservation set",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// CancelReservation cancels a reservation
func (h *ActionHandler) CancelReservation(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.CancelReservation(); err != nil {
		writeError(w, http.StatusBadRequest, "CANCEL_RESERVATION_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Reservation cancelled",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// SuspendEV suspends charging from EV side
func (h *ActionHandler) SuspendEV(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.SuspendEV(); err != nil {
		writeError(w, http.StatusBadRequest, "SUSPEND_EV_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Charging suspended (EV)",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// SuspendEVSE suspends charging from EVSE side
func (h *ActionHandler) SuspendEVSE(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.SuspendEVSE(); err != nil {
		writeError(w, http.StatusBadRequest, "SUSPEND_EVSE_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Charging suspended (EVSE)",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// ResumeCharging resumes charging from suspended state
func (h *ActionHandler) ResumeCharging(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := point.ResumeCharging(); err != nil {
		writeError(w, http.StatusBadRequest, "RESUME_FAILED", err.Error())
		return
	}

	h.stationService.TriggerStatusNotification(connectorID)

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Charging resumed",
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// SendBootNotification sends a boot notification
func (h *ActionHandler) SendBootNotification(w http.ResponseWriter, r *http.Request) {
	h.stationService.TriggerBootNotification()
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Boot notification sent",
	})
}

// SendHeartbeat sends a heartbeat
func (h *ActionHandler) SendHeartbeat(w http.ResponseWriter, r *http.Request) {
	h.stationService.TriggerHeartbeat()
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Heartbeat sent",
	})
}

// Reset performs a station reset
func (h *ActionHandler) Reset(w http.ResponseWriter, r *http.Request) {
	station := h.stationService.GetStation()
	station.Reset(entities.ResetTypeSoft)
	h.stationService.NotifyStatuses()
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Station reset",
	})
}

// Reconnect forces WebSocket reconnection
func (h *ActionHandler) Reconnect(w http.ResponseWriter, r *http.Request) {
	if h.reconnectFunc != nil {
		if err := h.reconnectFunc(); err != nil {
			writeError(w, http.StatusInternalServerError, "RECONNECT_FAILED", err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Reconnection initiated",
	})
}

// Disconnect disconnects WebSocket
func (h *ActionHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	if h.disconnectFunc != nil {
		if err := h.disconnectFunc(); err != nil {
			writeError(w, http.StatusInternalServerError, "DISCONNECT_FAILED", err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Disconnected",
	})
}

// ManualSetStatus manually sets connector status and sends StatusNotification
func (h *ActionHandler) ManualSetStatus(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	var req dto.SetStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Status == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must include 'status'")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	previousStatus := string(point.Status)
	if err := h.stationService.ManualSetStatus(connectorID, req.Status); err != nil {
		writeError(w, http.StatusBadRequest, "SET_STATUS_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:        true,
		Message:        "Status set to " + req.Status,
		ConnectorID:    connectorID,
		PreviousStatus: previousStatus,
		NewStatus:      string(point.Status),
	})
}

// ManualSendNextMeter sends the next queued meter value
func (h *ActionHandler) ManualSendNextMeter(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	var req dto.SendMeterRequest
	json.NewDecoder(r.Body).Decode(&req) // optional body

	sent, err := h.stationService.ManualSendNextMeter(connectorID, req.MeterValue, req.Soc)
	if err != nil {
		writeError(w, http.StatusBadRequest, "SEND_METER_FAILED", err.Error())
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	queueSize := len(point.GetMeterQueueSnapshot())

	writeJSON(w, http.StatusOK, dto.MeterSentResponse{
		Success:    true,
		MeterValue: sent.MeterValue,
		Soc:        sent.Soc,
		QueueSize:  queueSize,
	})
}

// ManualGetMeterQueue returns the pending meter value queue
func (h *ActionHandler) ManualGetMeterQueue(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	queue := point.GetMeterQueueSnapshot()
	writeJSON(w, http.StatusOK, dto.MeterQueueResponse{
		ConnectorID: connectorID,
		QueueSize:   len(queue),
		Entries:     queue,
	})
}

// ManualFlushMeterQueue clears the meter value queue
func (h *ActionHandler) ManualFlushMeterQueue(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	station := h.stationService.GetStation()
	point := station.GetPoint(connectorID)
	if point == nil {
		writeError(w, http.StatusNotFound, "CONNECTOR_NOT_FOUND", "Connector not found")
		return
	}

	count := point.FlushMeterQueue()
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:     true,
		Message:     fmt.Sprintf("Flushed %d meter values", count),
		ConnectorID: connectorID,
	})
}

// ManualSendStopTransaction sends StopTransaction for the active transaction
func (h *ActionHandler) ManualSendStopTransaction(w http.ResponseWriter, r *http.Request) {
	connectorID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_CONNECTOR_ID", "Invalid connector ID")
		return
	}

	if err := h.stationService.ManualSendStopTransaction(connectorID); err != nil {
		writeError(w, http.StatusBadRequest, "STOP_TRANSACTION_FAILED", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success:     true,
		Message:     "StopTransaction sent",
		ConnectorID: connectorID,
	})
}
