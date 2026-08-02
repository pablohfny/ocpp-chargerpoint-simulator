package services

import (
	"encoding/json"

	"EV-Client-Simulator/app/domain/entities"
)

// ocpiPayload is the loose shape of the OCPI objects our platform pushes to a
// partner. Every field is optional: PATCH bodies are partial by definition.
type ocpiPayload struct {
	ID        string             `json:"id"`
	SessionID string             `json:"session_id"`
	CDRToken  *entities.CDRToken `json:"cdr_token"`
	Result    string             `json:"result"`
}

// ocpiEnvelopeStatus is the verdict wrapper every OCPI response carries.
type ocpiEnvelopeStatus struct {
	StatusCode    *int   `json:"status_code"`
	StatusMessage string `json:"status_message"`
}

// ExtractEnvelopeStatus pulls the OCPI verdict out of a response body. A
// missing or unparseable envelope yields zero, which callers must read as "no
// verdict" rather than as a failure: not every error path returns an envelope.
func ExtractEnvelopeStatus(body []byte) (statusCode int, statusMessage string) {
	if len(body) == 0 {
		return 0, ""
	}

	var envelope ocpiEnvelopeStatus
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.StatusCode == nil {
		return 0, ""
	}
	return *envelope.StatusCode, envelope.StatusMessage
}

// ExtractCommandResult pulls the `result` of an async CommandResult callback
// (ACCEPTED, REJECTED, TIMEOUT, UNKNOWN_SESSION). An unparseable body yields an
// empty string, which the pipeline reads as "no verdict yet".
func ExtractCommandResult(body []byte) string {
	if len(body) == 0 {
		return ""
	}

	var payload ocpiPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return payload.Result
}

// ExtractCorrelation pulls the identifiers used to correlate an inbound push
// with the command that originated it. Unparseable bodies simply yield empty
// values so a malformed push is still logged.
func ExtractCorrelation(kind entities.OCPIEventKind, body []byte) (sessionID string, token entities.CDRToken) {
	if len(body) == 0 {
		return "", entities.CDRToken{}
	}

	var payload ocpiPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", entities.CDRToken{}
	}

	if payload.CDRToken != nil {
		token = *payload.CDRToken
	}

	switch kind {
	case entities.OCPIEventSession:
		sessionID = payload.ID
	case entities.OCPIEventCDR:
		sessionID = payload.SessionID
		if sessionID == "" {
			sessionID = payload.ID
		}
	}

	return sessionID, token
}
