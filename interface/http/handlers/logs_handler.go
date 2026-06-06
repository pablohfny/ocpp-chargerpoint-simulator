package handlers

import (
	"net/http"
	"strconv"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"
)

// LogsHandler handles message log endpoints
type LogsHandler struct {
	logService *services.MessageLogService
}

// NewLogsHandler creates a new logs handler
func NewLogsHandler(logService *services.MessageLogService) *LogsHandler {
	return &LogsHandler{
		logService: logService,
	}
}

// GetLogs returns message logs with optional filtering
func (h *LogsHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	filter := entities.MessageLogFilter{}

	// Parse query parameters
	if direction := r.URL.Query().Get("direction"); direction != "" {
		d := entities.MessageDirection(direction)
		filter.Direction = &d
	}

	if action := r.URL.Query().Get("action"); action != "" {
		filter.Action = action
	}

	if connectorStr := r.URL.Query().Get("connector"); connectorStr != "" {
		if connectorID, err := strconv.Atoi(connectorStr); err == nil {
			filter.ConnectorID = &connectorID
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	} else {
		filter.Limit = 100 // Default limit
	}

	entries := h.logService.GetEntries(filter)
	total := h.logService.Count()

	response := dto.MessageLogResponse{
		Total:   total,
		Offset:  filter.Offset,
		Limit:   filter.Limit,
		Entries: entries,
	}

	writeJSON(w, http.StatusOK, response)
}

// ClearLogs clears all message logs
func (h *LogsHandler) ClearLogs(w http.ResponseWriter, r *http.Request) {
	h.logService.Clear()
	writeJSON(w, http.StatusOK, dto.ActionResponse{
		Success: true,
		Message: "Message logs cleared",
	})
}
