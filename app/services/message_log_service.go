package services

import (
	"EV-Client-Simulator/app/domain/entities"
)

// MessageLogService manages OCPP message logging
type MessageLogService struct {
	log *entities.MessageLog
}

// NewMessageLogService creates a new message log service
func NewMessageLogService(maxEntries int) *MessageLogService {
	return &MessageLogService{
		log: entities.NewMessageLog(maxEntries),
	}
}

// LogOutgoing logs an outgoing message
func (s *MessageLogService) LogOutgoing(msg entities.Message, rawMessage string) {
	s.log.LogMessage(entities.DirectionOutgoing, msg, rawMessage)
}

// LogIncoming logs an incoming message
func (s *MessageLogService) LogIncoming(msg entities.Message, rawMessage string) {
	s.log.LogMessage(entities.DirectionIncoming, msg, rawMessage)
}

// LogRawIncoming logs a raw incoming message before full parsing
func (s *MessageLogService) LogRawIncoming(messageType int8, messageID, action string, payload map[string]interface{}, connectorID int, rawMessage string) {
	s.log.LogRawIncoming(messageType, messageID, action, payload, connectorID, rawMessage)
}

// GetEntries returns log entries with optional filtering
func (s *MessageLogService) GetEntries(filter entities.MessageLogFilter) []entities.MessageLogEntry {
	return s.log.GetEntries(filter)
}

// GetAllEntries returns all log entries
func (s *MessageLogService) GetAllEntries() []entities.MessageLogEntry {
	return s.log.GetAllEntries()
}

// Clear removes all entries
func (s *MessageLogService) Clear() {
	s.log.Clear()
}

// Count returns the number of entries
func (s *MessageLogService) Count() int {
	return s.log.Count()
}
