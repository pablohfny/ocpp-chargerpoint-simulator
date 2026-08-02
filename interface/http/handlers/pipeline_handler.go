package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"
)

// PipelineHandler exposes the end-to-end cockpit orchestrator.
type PipelineHandler struct {
	pipeline *services.PipelineService
}

// NewPipelineHandler creates the pipeline handler.
func NewPipelineHandler(pipeline *services.PipelineService) *PipelineHandler {
	return &PipelineHandler{pipeline: pipeline}
}

// StartContext arms a run against a partner, location and EVSE.
func (h *PipelineHandler) StartContext(w http.ResponseWriter, r *http.Request) {
	var req dto.PipelineContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	state, err := h.pipeline.StartContext(entities.PipelineContext{
		PartnerSlug:     req.PartnerSlug,
		LocationID:      req.LocationID,
		EvseUID:         req.EvseUID,
		ConnectorID:     req.ConnectorID,
		OCPPConnectorID: req.OCPPConnectorID,
	})
	if err != nil {
		writePipelineError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, state)
}

// GetState returns the current stage, hops and valid actions.
func (h *PipelineHandler) GetState(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.pipeline.State())
}

// RunAction executes one action of the current stage.
func (h *PipelineHandler) RunAction(w http.ResponseWriter, r *http.Request) {
	var req dto.PipelineActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Action == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Request body must include 'action'")
		return
	}

	state, err := h.pipeline.Execute(entities.PipelineActionID(req.Action))
	if err != nil {
		writePipelineError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, state)
}

// writePipelineError maps orchestrator errors onto HTTP responses.
func writePipelineError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrPipelineNoContext):
		writeError(w, http.StatusConflict, "PIPELINE_NO_CONTEXT", err.Error())
	case errors.Is(err, services.ErrActionNotAvailable):
		writeError(w, http.StatusConflict, "ACTION_NOT_AVAILABLE", err.Error())
	case errors.Is(err, services.ErrPartnerNotFound):
		writeError(w, http.StatusNotFound, "PARTNER_NOT_FOUND", err.Error())
	default:
		writeError(w, http.StatusBadRequest, "PIPELINE_ACTION_FAILED", err.Error())
	}
}
