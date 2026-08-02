package entities

import "time"

// OCPI command types issued by an eMSP towards a CPO.
const (
	CommandStartSession = "START_SESSION"
	CommandStopSession  = "STOP_SESSION"
)

// OCPIToken is the token object sent inside a START_SESSION command.
type OCPIToken struct {
	CountryCode string    `json:"country_code"`
	PartyID     string    `json:"party_id"`
	UID         string    `json:"uid"`
	Type        string    `json:"type"`
	ContractID  string    `json:"contract_id"`
	Issuer      string    `json:"issuer"`
	Valid       bool      `json:"valid"`
	Whitelist   string    `json:"whitelist"`
	LastUpdated time.Time `json:"last_updated"`
}

// StartSessionCommand is the OCPI 2.2.1 START_SESSION request body.
type StartSessionCommand struct {
	ResponseURL string    `json:"response_url"`
	Token       OCPIToken `json:"token"`
	LocationID  string    `json:"location_id"`
	EvseUID     string    `json:"evse_uid,omitempty"`
	ConnectorID string    `json:"connector_id,omitempty"`
}

// StopSessionCommand is the OCPI 2.2.1 STOP_SESSION request body.
type StopSessionCommand struct {
	ResponseURL string `json:"response_url"`
	SessionID   string `json:"session_id"`
}

// OCPIEnvelope is the standard OCPI response wrapper.
type OCPIEnvelope struct {
	Data          interface{} `json:"data"`
	StatusCode    int         `json:"status_code"`
	StatusMessage string      `json:"status_message"`
	Timestamp     string      `json:"timestamp"`
}

// OCPI status codes used by the receiver endpoints.
const (
	OCPIStatusSuccess           = 1000
	OCPIStatusInvalidParameters = 2001
)

// NewOCPIEnvelope builds an OCPI response envelope stamped at the given time.
func NewOCPIEnvelope(data interface{}, statusCode int, statusMessage string, now time.Time) OCPIEnvelope {
	if data == nil {
		data = struct{}{}
	}
	return OCPIEnvelope{
		Data:          data,
		StatusCode:    statusCode,
		StatusMessage: statusMessage,
		Timestamp:     now.UTC().Format(time.RFC3339),
	}
}
