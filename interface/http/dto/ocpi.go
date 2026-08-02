package dto

import "EV-Client-Simulator/app/domain/entities"

// StartSessionRequest is the body of the UI's "start charging" action.
type StartSessionRequest struct {
	LocationID  string `json:"locationId"`
	EvseUID     string `json:"evseUid"`
	ConnectorID string `json:"connectorId"`
}

// StopSessionRequest is the body of the UI's "stop charging" action.
type StopSessionRequest struct {
	SessionID string `json:"sessionId"`
}

// OCPIEventsResponse is the polled event feed for a partner.
type OCPIEventsResponse struct {
	PartnerSlug string               `json:"partnerSlug"`
	LastID      int64                `json:"lastId"`
	Events      []entities.OCPIEvent `json:"events"`
}

// OCPIDefaultsResponse exposes the env-provided form defaults to the UI.
type OCPIDefaultsResponse struct {
	LocationID    string `json:"locationId"`
	EvseUID       string `json:"evseUid"`
	OCPIBaseURL   string `json:"ocpiBaseUrl"`
	PublicBaseURL string `json:"publicBaseUrl"`
}

// BatteryResponse is the simplified page's view of a connector.
type BatteryResponse struct {
	ConnectorID    int     `json:"connectorId"`
	Status         string  `json:"status"`
	BatteryPercent int     `json:"batteryPercent"`
	MeterValueWh   float64 `json:"meterValueWh"`
	CapacityKWh    float64 `json:"capacityKWh"`
	Charging       bool    `json:"charging"`
	CablePlugged   bool    `json:"cablePlugged"`
	TransactionID  int     `json:"transactionId,omitempty"`
	Connected      bool    `json:"connected"`
}
