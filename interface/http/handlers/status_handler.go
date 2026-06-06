package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"

	"github.com/go-chi/chi/v5"
)

// StatusHandler handles status-related endpoints
type StatusHandler struct {
	stationService *services.ChargerStationService
	clientID       string
	serverAddr     string
	connected      *bool
}

// NewStatusHandler creates a new status handler
func NewStatusHandler(stationService *services.ChargerStationService, clientID, serverAddr string, connected *bool) *StatusHandler {
	return &StatusHandler{
		stationService: stationService,
		clientID:       clientID,
		serverAddr:     serverAddr,
		connected:      connected,
	}
}

// GetStatus returns overall simulator status
func (h *StatusHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	station := h.stationService.GetStation()
	response := dto.StatusResponse{
		Status:     "running",
		StationID:  station.ID,
		Connectors: len(station.ChargerPoints),
		Connected:  *h.connected,
	}
	writeJSON(w, http.StatusOK, response)
}

// GetStation returns station details
func (h *StatusHandler) GetStation(w http.ResponseWriter, r *http.Request) {
	station := h.stationService.GetStation()
	writeJSON(w, http.StatusOK, station)
}

// GetConnectors returns all connectors status
func (h *StatusHandler) GetConnectors(w http.ResponseWriter, r *http.Request) {
	station := h.stationService.GetStation()
	connectors := make([]dto.ConnectorResponse, len(station.ChargerPoints))
	for i, point := range station.ChargerPoints {
		connectors[i] = dto.NewConnectorResponse(point)
	}
	writeJSON(w, http.StatusOK, connectors)
}

// GetConnector returns a specific connector status
func (h *StatusHandler) GetConnector(w http.ResponseWriter, r *http.Request) {
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

	writeJSON(w, http.StatusOK, dto.NewConnectorResponse(point))
}

// GetTransactions returns all active transactions
func (h *StatusHandler) GetTransactions(w http.ResponseWriter, r *http.Request) {
	station := h.stationService.GetStation()
	active := station.GetActiveTransactions()

	transactions := make([]dto.TransactionResponse, len(active))
	for i, point := range active {
		transactions[i] = dto.TransactionResponse{
			TransactionID: point.CurrentTransaction,
			ConnectorID:   point.ID,
			IdTag:         point.CurrentIdTag,
			MeterValue:    point.MeterValue,
			Soc:           point.Soc,
			Status:        string(point.Status),
		}
	}
	writeJSON(w, http.StatusOK, transactions)
}

// GetConnection returns WebSocket connection status
func (h *StatusHandler) GetConnection(w http.ResponseWriter, r *http.Request) {
	response := dto.ConnectionResponse{
		Connected:  *h.connected,
		ServerAddr: h.serverAddr,
		ClientID:   h.clientID,
	}
	writeJSON(w, http.StatusOK, response)
}

// Helper functions
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(dto.ErrorResponse{
		Error: dto.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}
