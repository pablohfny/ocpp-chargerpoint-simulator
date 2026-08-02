package entities

import (
	"encoding/json"
	"time"
)

// OCPIEventKind classifies what an event represents.
type OCPIEventKind string

const (
	OCPIEventSession       OCPIEventKind = "session"
	OCPIEventCDR           OCPIEventKind = "cdr"
	OCPIEventLocation      OCPIEventKind = "location"
	OCPIEventTariff        OCPIEventKind = "tariff"
	OCPIEventCommandResult OCPIEventKind = "command_result"
	OCPIEventCommandSent   OCPIEventKind = "command_sent"
	OCPIEventAuthFailed    OCPIEventKind = "auth_failed"
)

// OCPI event directions.
const (
	OCPIDirectionIn  = "in"
	OCPIDirectionOut = "out"
)

// OCPIEvent is a single recorded interaction with a partner profile, either an
// inbound push from our platform or an outbound command we issued.
type OCPIEvent struct {
	ID          int64           `json:"id"`
	PartnerSlug string          `json:"partnerSlug"`
	Timestamp   time.Time       `json:"timestamp"`
	Direction   string          `json:"direction"`
	Kind        OCPIEventKind   `json:"kind"`
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Body        json.RawMessage `json:"body,omitempty"`

	// Correlation fields, extracted from the body when present.
	SessionID  string `json:"sessionId,omitempty"`
	CommandUID string `json:"commandUid,omitempty"`
	ContractID string `json:"contractId,omitempty"`
	TokenUID   string `json:"tokenUid,omitempty"`
	TokenType  string `json:"tokenType,omitempty"`

	// AuthFailed marks a receiver call that presented a wrong or missing token.
	AuthFailed bool `json:"authFailed,omitempty"`
	// StatusCode is the HTTP status we replied with (inbound) or received
	// (outbound).
	StatusCode int `json:"statusCode,omitempty"`
	// ResponseBody is what the platform answered an outbound command with. It
	// is kept because the HTTP status alone does not decide success: OCPI
	// carries its own verdict in the envelope.
	ResponseBody json.RawMessage `json:"responseBody,omitempty"`
	// OCPIStatusCode is the envelope's status_code from that response, where
	// 1000 is success and anything else is a rejection served over HTTP 200.
	// Zero means the reply carried no parseable envelope.
	OCPIStatusCode int `json:"ocpiStatusCode,omitempty"`
	// OCPIStatusMessage is the envelope's human readable status_message.
	OCPIStatusMessage string `json:"ocpiStatusMessage,omitempty"`

	// EchoOK reports whether the cdr_token echoed back by our platform matches
	// the token we sent in the originating command. Nil when there is nothing
	// to compare against.
	EchoOK *bool `json:"echoOk,omitempty"`
	// EchoDiff lists the cdr_token fields that did not match.
	EchoDiff []string `json:"echoDiff,omitempty"`
	// EchoAgainstCommandUID identifies the command_sent event used for the
	// comparison.
	EchoAgainstCommandUID string `json:"echoAgainstCommandUid,omitempty"`

	// Error holds a transport error message for outbound commands.
	Error string `json:"error,omitempty"`
}

// CDRToken is the OCPI token snapshot embedded in Sessions and CDRs.
type CDRToken struct {
	UID        string `json:"uid"`
	Type       string `json:"type"`
	ContractID string `json:"contract_id"`
}
