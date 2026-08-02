package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"

	"github.com/go-chi/chi/v5"
)

// maxReceiverBodyBytes bounds an inbound OCPI push body.
const maxReceiverBodyBytes = 8 * 1024 * 1024

// OCPIReceiverHandler serves the endpoints our platform pushes data to. These
// routes are NOT behind basic auth: they are authenticated with the partner's
// own OCPI token.
type OCPIReceiverHandler struct {
	partners *services.OCPIPartnerService
	events   *services.OCPIEventService
}

// NewOCPIReceiverHandler creates the receiver handler.
func NewOCPIReceiverHandler(partners *services.OCPIPartnerService, events *services.OCPIEventService) *OCPIReceiverHandler {
	return &OCPIReceiverHandler{partners: partners, events: events}
}

// HandlePush returns a handler that validates the partner token and records the
// push as an event of the given kind.
func (h *OCPIReceiverHandler) HandlePush(kind entities.OCPIEventKind) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.handle(w, r, kind, "")
	}
}

// HandleCommandResult records the async CommandResult callback for a command.
func (h *OCPIReceiverHandler) HandleCommandResult(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, entities.OCPIEventCommandResult, chi.URLParam(r, "uid"))
}

// handle is the shared receiver pipeline: resolve partner, validate token,
// record event, reply with an OCPI envelope.
func (h *OCPIReceiverHandler) handle(w http.ResponseWriter, r *http.Request, kind entities.OCPIEventKind, commandUID string) {
	slug := chi.URLParam(r, "slug")

	partner, err := h.partners.Get(slug)
	if err != nil {
		writeOCPI(w, http.StatusNotFound, nil, entities.OCPIStatusInvalidParameters, "Unknown partner")
		return
	}

	body := readBody(r)

	event := entities.OCPIEvent{
		PartnerSlug: partner.Slug,
		Direction:   entities.OCPIDirectionIn,
		Kind:        kind,
		Method:      r.Method,
		Path:        r.URL.Path,
		Body:        body,
		CommandUID:  commandUID,
	}

	if !partner.MatchesAuthorization(r.Header.Get("Authorization")) {
		event.Kind = entities.OCPIEventAuthFailed
		event.AuthFailed = true
		event.StatusCode = http.StatusUnauthorized
		h.events.Record(event)
		writeOCPI(w, http.StatusUnauthorized, nil, entities.OCPIStatusInvalidParameters, "Invalid or missing authorization token")
		return
	}

	sessionID, token := services.ExtractCorrelation(kind, body)
	event.SessionID = sessionID
	event.TokenUID = token.UID
	event.TokenType = token.Type
	event.ContractID = token.ContractID
	event.StatusCode = http.StatusOK

	h.events.Record(event)
	writeOCPI(w, http.StatusOK, nil, entities.OCPIStatusSuccess, "Success")
}

// readBody reads and normalizes the request body into valid JSON, or nil.
func readBody(r *http.Request) json.RawMessage {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxReceiverBodyBytes))
	if err != nil || len(raw) == 0 {
		return nil
	}
	if json.Valid(raw) {
		return raw
	}
	// Keep unparseable payloads visible in the feed as a JSON string.
	encoded, err := json.Marshal(string(raw))
	if err != nil {
		return nil
	}
	return encoded
}

// writeOCPI writes an OCPI response envelope.
func writeOCPI(w http.ResponseWriter, httpStatus int, data interface{}, statusCode int, statusMessage string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(entities.NewOCPIEnvelope(data, statusCode, statusMessage, time.Now()))
}
