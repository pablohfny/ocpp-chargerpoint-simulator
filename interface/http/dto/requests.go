package dto

// PlugCableRequest represents a request to plug/unplug cable
type PlugCableRequest struct {
	ConnectorID int `json:"connectorId"`
}

// StartChargingRequest represents a request to start local charging
type StartChargingRequest struct {
	ConnectorID int    `json:"connectorId"`
	IdTag       string `json:"idTag"`
}

// StopChargingRequest represents a request to stop charging
type StopChargingRequest struct {
	ConnectorID int `json:"connectorId"`
}

// AuthorizeRequest represents a request to authorize
type AuthorizeRequest struct {
	ConnectorID int    `json:"connectorId"`
	IdTag       string `json:"idTag"`
}

// SetFaultRequest represents a request to set a connector fault
type SetFaultRequest struct {
	ErrorCode       string `json:"errorCode"`
	Info            string `json:"info,omitempty"`
	VendorErrorCode string `json:"vendorErrorCode,omitempty"`
}

// SetReservationRequest represents a request to set a reservation
type SetReservationRequest struct {
	ReservationID int    `json:"reservationId"`
	IdTag         string `json:"idTag"`
	ExpiryMinutes int    `json:"expiryMinutes"`
}

// ChangeConfigRequest represents a request to change configuration
type ChangeConfigRequest struct {
	Value string `json:"value"`
}

// DelaySettingsRequest represents delay settings update request
type DelaySettingsRequest struct {
	BootNotificationDelayMs   int `json:"bootNotificationDelayMs"`
	AuthorizationDelayMs      int `json:"authorizationDelayMs"`
	StartTransactionDelayMs   int `json:"startTransactionDelayMs"`
	StopTransactionDelayMs    int `json:"stopTransactionDelayMs"`
	StatusNotificationDelayMs int `json:"statusNotificationDelayMs"`
	HeartbeatDelayMs          int `json:"heartbeatDelayMs"`
	MeterValuesDelayMs        int `json:"meterValuesDelayMs"`
	RemoteStartDelayMs        int `json:"remoteStartDelayMs"`
	RemoteStopDelayMs         int `json:"remoteStopDelayMs"`
}

// FailureRatesRequest represents failure rates update request
type FailureRatesRequest struct {
	AuthorizationFailureRate    float64 `json:"authorizationFailureRate"`
	StartTransactionFailureRate float64 `json:"startTransactionFailureRate"`
	StopTransactionFailureRate  float64 `json:"stopTransactionFailureRate"`
	MeterValuesFailureRate      float64 `json:"meterValuesFailureRate"`
	HeartbeatFailureRate        float64 `json:"heartbeatFailureRate"`
}

// ErrorInjectionRequest represents error injection settings update request
type ErrorInjectionRequest struct {
	Enabled              bool    `json:"enabled"`
	InternalErrorRate    float64 `json:"internalErrorRate"`
	NotImplementedRate   float64 `json:"notImplementedRate"`
	NotSupportedRate     float64 `json:"notSupportedRate"`
	MessageTimeoutRate   float64 `json:"messageTimeoutRate"`
	MalformedMessageRate float64 `json:"malformedMessageRate"`
}

// TransactionSettingsRequest represents a request to update transaction settings
type TransactionSettingsRequest struct {
	MeterStepWh          float64 `json:"meterStepWh"`
	StartSOC             int     `json:"startSOC"`
	FinalSOC             int     `json:"finalSOC"`
	FinalSOCBehavior     string  `json:"finalSOCBehavior"`
	StopDelaySec         int     `json:"stopDelaySec"`
	PreparingDurationSec int     `json:"preparingDurationSec"`
}

// TriggerMessageRequest represents a request to trigger an OCPP message
type TriggerMessageRequest struct {
	ConnectorID int `json:"connectorId,omitempty"`
}

// TriggerAuthorizeRequest represents a request to send an Authorize message
// with a caller-chosen idTag (used to simulate the auto-charge flow).
type TriggerAuthorizeRequest struct {
	ConnectorID int    `json:"connectorId,omitempty"` // optional, defaults to 1
	IdTag       string `json:"idTag"`                 // required
}

// TriggerStartTransactionRequest represents a request to send a
// StartTransaction message with a caller-chosen idTag.
type TriggerStartTransactionRequest struct {
	ConnectorID int     `json:"connectorId,omitempty"` // optional, defaults to 1
	IdTag       string  `json:"idTag"`                 // required
	MeterStart  float64 `json:"meterStart,omitempty"`  // optional, defaults to 0
}

// ManualModeRequest represents a request to toggle manual mode
type ManualModeRequest struct {
	Enabled bool `json:"enabled"`
}

// SetStatusRequest represents a request to manually set connector status
type SetStatusRequest struct {
	Status string `json:"status"`
}

// SendMeterRequest represents a request to send next meter value with optional overrides
type SendMeterRequest struct {
	MeterValue *float64 `json:"meterValue,omitempty"`
	Soc        *int16   `json:"soc,omitempty"`
}
