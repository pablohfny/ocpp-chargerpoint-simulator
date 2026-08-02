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
