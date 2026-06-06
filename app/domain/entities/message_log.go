package entities

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// MessageDirection indicates if a message was incoming or outgoing
type MessageDirection string

const (
	DirectionIncoming MessageDirection = "incoming"
	DirectionOutgoing MessageDirection = "outgoing"
)

// MessageLogEntry represents a logged OCPP message
type MessageLogEntry struct {
	ID          string                 `json:"id"`
	Timestamp   time.Time              `json:"timestamp"`
	Direction   MessageDirection       `json:"direction"`
	MessageType int8                   `json:"messageType"`
	MessageID   string                 `json:"messageId"`
	Action      string                 `json:"action"`
	Payload     map[string]interface{} `json:"payload"`
	ConnectorID int                    `json:"connectorId,omitempty"`
	RawMessage  string                 `json:"rawMessage"`
}

// MessageLog maintains a circular buffer of OCPP messages
type MessageLog struct {
	entries    []MessageLogEntry
	maxEntries int
	mu         sync.RWMutex
}

// NewMessageLog creates a new message log with specified capacity
func NewMessageLog(maxEntries int) *MessageLog {
	if maxEntries <= 0 {
		maxEntries = 1000
	}
	return &MessageLog{
		entries:    make([]MessageLogEntry, 0, maxEntries),
		maxEntries: maxEntries,
	}
}

// LogMessage adds a message to the log
func (ml *MessageLog) LogMessage(direction MessageDirection, msg Message, rawMessage string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	entry := MessageLogEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		Direction:   direction,
		MessageType: msg.Type,
		MessageID:   msg.ID,
		Action:      msg.Action,
		Payload:     msg.Payload,
		ConnectorID: msg.ConnectorId,
		RawMessage:  rawMessage,
	}

	// Circular buffer: remove oldest if at capacity
	if len(ml.entries) >= ml.maxEntries {
		ml.entries = ml.entries[1:]
	}

	ml.entries = append(ml.entries, entry)
}

// LogRawIncoming logs a raw incoming message before parsing
func (ml *MessageLog) LogRawIncoming(messageType int8, messageID, action string, payload map[string]interface{}, connectorID int, rawMessage string) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	entry := MessageLogEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		Direction:   DirectionIncoming,
		MessageType: messageType,
		MessageID:   messageID,
		Action:      action,
		Payload:     payload,
		ConnectorID: connectorID,
		RawMessage:  rawMessage,
	}

	if len(ml.entries) >= ml.maxEntries {
		ml.entries = ml.entries[1:]
	}

	ml.entries = append(ml.entries, entry)
}

// GetEntries returns log entries with optional filtering
func (ml *MessageLog) GetEntries(filter MessageLogFilter) []MessageLogEntry {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	result := make([]MessageLogEntry, 0)

	for _, entry := range ml.entries {
		if filter.matches(entry) {
			result = append(result, entry)
		}
	}

	// Apply pagination
	if filter.Offset > 0 {
		if filter.Offset >= len(result) {
			return []MessageLogEntry{}
		}
		result = result[filter.Offset:]
	}

	if filter.Limit > 0 && filter.Limit < len(result) {
		result = result[:filter.Limit]
	}

	return result
}

// GetAllEntries returns all log entries
func (ml *MessageLog) GetAllEntries() []MessageLogEntry {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	result := make([]MessageLogEntry, len(ml.entries))
	copy(result, ml.entries)
	return result
}

// Clear removes all entries from the log
func (ml *MessageLog) Clear() {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	ml.entries = make([]MessageLogEntry, 0, ml.maxEntries)
}

// Count returns the current number of entries
func (ml *MessageLog) Count() int {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	return len(ml.entries)
}

// MessageLogFilter defines filtering criteria for log queries
type MessageLogFilter struct {
	Direction   *MessageDirection
	Action      string
	ConnectorID *int
	Since       *time.Time
	Until       *time.Time
	Offset      int
	Limit       int
}

// matches checks if a log entry matches the filter criteria
func (f *MessageLogFilter) matches(entry MessageLogEntry) bool {
	if f.Direction != nil && entry.Direction != *f.Direction {
		return false
	}

	if f.Action != "" && entry.Action != f.Action {
		return false
	}

	if f.ConnectorID != nil && entry.ConnectorID != *f.ConnectorID {
		return false
	}

	if f.Since != nil && entry.Timestamp.Before(*f.Since) {
		return false
	}

	if f.Until != nil && entry.Timestamp.After(*f.Until) {
		return false
	}

	return true
}
