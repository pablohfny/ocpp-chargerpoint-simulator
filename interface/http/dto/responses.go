package dto

import (
	"EV-Client-Simulator/app/domain/entities"
	"time"
)

// StatusResponse represents the overall status response
type StatusResponse struct {
	Status     string `json:"status"`
	StationID  string `json:"stationId"`
	Connectors int    `json:"connectors"`
	Connected  bool   `json:"connected"`
	Uptime     string `json:"uptime,omitempty"`
}

// ConnectorResponse represents a connector status
type ConnectorResponse struct {
	ID                 int       `json:"id"`
	Status             string    `json:"status"`
	ErrorCode          string    `json:"errorCode"`
	ErrorInfo          string    `json:"errorInfo,omitempty"`
	VendorErrorCode    string    `json:"vendorErrorCode,omitempty"`
	CurrentTransaction int       `json:"currentTransaction,omitempty"`
	CurrentIdTag       string    `json:"currentIdTag,omitempty"`
	MeterValue         float64   `json:"meterValue"`
	Soc                int16     `json:"soc"`
	ReservationID      int       `json:"reservationId,omitempty"`
	ReservationIdTag   string    `json:"reservationIdTag,omitempty"`
	ReservationExpiry  time.Time `json:"reservationExpiry,omitempty"`
	CablePlugged       bool      `json:"cablePlugged"`
}

// TransactionResponse represents a transaction
type TransactionResponse struct {
	TransactionID int     `json:"transactionId"`
	ConnectorID   int     `json:"connectorId"`
	IdTag         string  `json:"idTag"`
	MeterValue    float64 `json:"meterValue"`
	Soc           int16   `json:"soc"`
	Status        string  `json:"status"`
}

// ConnectionResponse represents WebSocket connection status
type ConnectionResponse struct {
	Connected   bool   `json:"connected"`
	ServerAddr  string `json:"serverAddr"`
	ClientID    string `json:"clientId"`
	LastMessage string `json:"lastMessage,omitempty"`
}

// ConfigKeyResponse represents an OCPP config key
type ConfigKeyResponse struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Readonly bool   `json:"readonly"`
}

// SimulationSettingsResponse represents simulation settings
type SimulationSettingsResponse struct {
	ManualMode       bool                             `json:"manualMode"`
	Delays           entities.DelaySettings           `json:"delays"`
	FailureRates     entities.FailureRateSettings     `json:"failureRates"`
	ErrorInjection   entities.ErrorInjectionSettings  `json:"errorInjection"`
	ChargingBehavior entities.ChargingBehaviorSettings `json:"chargingBehavior"`
	Transaction      entities.TransactionSettings      `json:"transaction"`
}

// MessageLogResponse represents message log entries
type MessageLogResponse struct {
	Total   int                        `json:"total"`
	Offset  int                        `json:"offset"`
	Limit   int                        `json:"limit"`
	Entries []entities.MessageLogEntry `json:"entries"`
}

// ActionResponse represents a generic action response
type ActionResponse struct {
	Success        bool   `json:"success"`
	Message        string `json:"message,omitempty"`
	ConnectorID    int    `json:"connectorId,omitempty"`
	PreviousStatus string `json:"previousStatus,omitempty"`
	NewStatus      string `json:"newStatus,omitempty"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details
type ErrorDetail struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// MeterQueueResponse represents the meter value queue
type MeterQueueResponse struct {
	ConnectorID int                        `json:"connectorId"`
	QueueSize   int                        `json:"queueSize"`
	Entries     []entities.MeterQueueEntry `json:"entries"`
}

// MeterSentResponse represents a sent meter value
type MeterSentResponse struct {
	Success    bool    `json:"success"`
	MeterValue float64 `json:"meterValue"`
	Soc        int16   `json:"soc"`
	QueueSize  int     `json:"queueSize"`
}

// NewConnectorResponse creates a ConnectorResponse from a ChargerPoint
func NewConnectorResponse(point *entities.ChargerPoint) ConnectorResponse {
	return ConnectorResponse{
		ID:                 point.ID,
		Status:             string(point.Status),
		ErrorCode:          string(point.ErrorCode),
		ErrorInfo:          point.ErrorInfo,
		VendorErrorCode:    point.VendorErrorCode,
		CurrentTransaction: point.CurrentTransaction,
		CurrentIdTag:       point.CurrentIdTag,
		MeterValue:         point.MeterValue,
		Soc:                point.Soc,
		ReservationID:      point.ReservationID,
		ReservationIdTag:   point.ReservationIdTag,
		ReservationExpiry:  point.ReservationExpiry,
		CablePlugged:       point.CablePlugged,
	}
}
