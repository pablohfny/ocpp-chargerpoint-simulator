package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"

	"github.com/go-chi/chi/v5"
)

// OCPIPartnerHandler serves the partner control API used by the Parceiros tab.
type OCPIPartnerHandler struct {
	partners *services.OCPIPartnerService
	events   *services.OCPIEventService
	commands *services.OCPICommandService
	settings *services.AppSettingsService
}

// NewOCPIPartnerHandler creates the partner control handler.
func NewOCPIPartnerHandler(
	partners *services.OCPIPartnerService,
	events *services.OCPIEventService,
	commands *services.OCPICommandService,
	settings *services.AppSettingsService,
) *OCPIPartnerHandler {
	return &OCPIPartnerHandler{partners: partners, events: events, commands: commands, settings: settings}
}

// GetDefaults returns the configured defaults used to prefill the UI forms.
func (h *OCPIPartnerHandler) GetDefaults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.defaults())
}

// defaults reads the current form defaults from the settings service, so a
// change saved in the Config tab is picked up without a restart.
func (h *OCPIPartnerHandler) defaults() dto.OCPIDefaultsResponse {
	settings := h.settings.Get()
	return dto.OCPIDefaultsResponse{
		LocationID:    settings.DefaultLocationID,
		EvseUID:       settings.DefaultEvseUID,
		OCPIBaseURL:   settings.OCPIBaseURL,
		PublicBaseURL: settings.PublicBaseURL,
	}
}

// ListPartners returns every partner profile.
func (h *OCPIPartnerHandler) ListPartners(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.partners.List())
}

// GetPartner returns a single partner profile.
func (h *OCPIPartnerHandler) GetPartner(w http.ResponseWriter, r *http.Request) {
	partner, err := h.partners.Get(chi.URLParam(r, "slug"))
	if err != nil {
		writePartnerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, partner)
}

// CreatePartner stores a new partner profile.
func (h *OCPIPartnerHandler) CreatePartner(w http.ResponseWriter, r *http.Request) {
	var partner entities.OCPIPartner
	if err := json.NewDecoder(r.Body).Decode(&partner); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.applyDefaults(&partner)

	created, err := h.partners.Create(partner)
	if err != nil {
		writePartnerError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// UpdatePartner replaces an existing partner profile.
func (h *OCPIPartnerHandler) UpdatePartner(w http.ResponseWriter, r *http.Request) {
	var partner entities.OCPIPartner
	if err := json.NewDecoder(r.Body).Decode(&partner); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	h.applyDefaults(&partner)

	updated, err := h.partners.Update(chi.URLParam(r, "slug"), partner)
	if err != nil {
		writePartnerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// DeletePartner removes a partner profile.
func (h *OCPIPartnerHandler) DeletePartner(w http.ResponseWriter, r *http.Request) {
	if err := h.partners.Delete(chi.URLParam(r, "slug")); err != nil {
		writePartnerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.ActionResponse{Success: true, Message: "Partner deleted"})
}

// StartSession dispatches a START_SESSION command for the partner.
func (h *OCPIPartnerHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	var req dto.StartSessionRequest
	json.NewDecoder(r.Body).Decode(&req) // optional body, defaults applied below

	defaults := h.defaults()
	if req.LocationID == "" {
		req.LocationID = defaults.LocationID
	}
	if req.EvseUID == "" {
		req.EvseUID = defaults.EvseUID
	}

	result, err := h.commands.StartSession(chi.URLParam(r, "slug"), req.LocationID, req.EvseUID, req.ConnectorID)
	if err != nil {
		writePartnerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// StopSession dispatches a STOP_SESSION command for the partner.
func (h *OCPIPartnerHandler) StopSession(w http.ResponseWriter, r *http.Request) {
	var req dto.StopSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	result, err := h.commands.StopSession(chi.URLParam(r, "slug"), req.SessionID)
	if err != nil {
		writePartnerError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// GetEvents returns the partner's event feed newer than the `after` cursor.
func (h *OCPIPartnerHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if _, err := h.partners.Get(slug); err != nil {
		writePartnerError(w, err)
		return
	}

	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	events := h.events.List(slug, after, limit)
	lastID := after
	if len(events) > 0 {
		lastID = events[len(events)-1].ID
	}

	writeJSON(w, http.StatusOK, dto.OCPIEventsResponse{
		PartnerSlug: slug,
		LastID:      lastID,
		Events:      events,
	})
}

// ClearEvents drops the partner's in-memory event feed.
func (h *OCPIPartnerHandler) ClearEvents(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	if _, err := h.partners.Get(slug); err != nil {
		writePartnerError(w, err)
		return
	}

	h.events.Clear(slug)
	writeJSON(w, http.StatusOK, dto.ActionResponse{Success: true, Message: "Events cleared"})
}

// applyDefaults fills the URL fields a UI form may leave blank.
func (h *OCPIPartnerHandler) applyDefaults(partner *entities.OCPIPartner) {
	defaults := h.defaults()
	if partner.OCPIBaseURL == "" {
		partner.OCPIBaseURL = defaults.OCPIBaseURL
	}
	if partner.PublicBaseURL == "" {
		partner.PublicBaseURL = defaults.PublicBaseURL
	}
}

// writePartnerError maps service errors onto HTTP responses.
func writePartnerError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrPartnerNotFound):
		writeError(w, http.StatusNotFound, "PARTNER_NOT_FOUND", err.Error())
	case errors.Is(err, services.ErrPartnerExists):
		writeError(w, http.StatusConflict, "PARTNER_EXISTS", err.Error())
	default:
		writeError(w, http.StatusBadRequest, "INVALID_PARTNER", err.Error())
	}
}
