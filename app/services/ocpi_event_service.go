package services

import (
	"sync"
	"time"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/infrastructure/persistence"
)

// OCPIEventBufferSize is the number of events retained per partner.
const OCPIEventBufferSize = 500

// ocpiEventLoadCap bounds how many lines are replayed from the JSONL log on
// boot before they are split per partner and trimmed to OCPIEventBufferSize.
const ocpiEventLoadCap = 5000

// OCPIEventService keeps a per-partner ring buffer of OCPI events, appends them
// to a durable log, and verifies that the cdr_token our platform echoes back
// matches the one we sent in the originating command.
type OCPIEventService struct {
	mu      sync.RWMutex
	buffers map[string][]entities.OCPIEvent
	nextID  int64
	log     *persistence.OCPIEventLog
}

// NewOCPIEventService creates an event service. The log may be nil, in which
// case events are kept in memory only.
func NewOCPIEventService(log *persistence.OCPIEventLog) *OCPIEventService {
	return &OCPIEventService{
		buffers: make(map[string][]entities.OCPIEvent),
		nextID:  1,
		log:     log,
	}
}

// Load replays persisted events into the in-memory buffers.
func (s *OCPIEventService) Load() error {
	if s.log == nil {
		return nil
	}

	events, err := s.log.LoadTail(ocpiEventLoadCap)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, event := range events {
		s.buffers[event.PartnerSlug] = appendCapped(s.buffers[event.PartnerSlug], event)
		if event.ID >= s.nextID {
			s.nextID = event.ID + 1
		}
	}
	return nil
}

// Record stamps, verifies and stores an event, returning the stored copy.
func (s *OCPIEventService) Record(event entities.OCPIEvent) entities.OCPIEvent {
	s.mu.Lock()

	event.ID = s.nextID
	s.nextID++
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if shouldVerifyEcho(event) {
		verifyEcho(&event, s.buffers[event.PartnerSlug])
	}

	s.buffers[event.PartnerSlug] = appendCapped(s.buffers[event.PartnerSlug], event)
	s.mu.Unlock()

	if s.log != nil {
		// A logging failure must never break the receiver response.
		_ = s.log.Append(event)
	}
	return event
}

// List returns the buffered events for a partner with an ID greater than
// `after`, oldest first. A limit of zero returns everything available.
func (s *OCPIEventService) List(slug string, after int64, limit int) []entities.OCPIEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buffer := s.buffers[slug]
	result := make([]entities.OCPIEvent, 0, len(buffer))
	for _, event := range buffer {
		if event.ID > after {
			result = append(result, event)
		}
	}

	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}
	return result
}

// LastID returns the id of the newest buffered event for a partner, or zero
// when the partner has none. It is the cursor a pipeline run opens with, so
// events from earlier runs never bleed into it.
func (s *OCPIEventService) LastID(slug string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	buffer := s.buffers[slug]
	if len(buffer) == 0 {
		return 0
	}
	return buffer[len(buffer)-1].ID
}

// Clear drops every buffered event for a partner. The durable log is left
// untouched, so cleared events reappear only if the process is restarted.
func (s *OCPIEventService) Clear(slug string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.buffers, slug)
}

// shouldVerifyEcho reports whether an event carries a cdr_token worth checking.
func shouldVerifyEcho(event entities.OCPIEvent) bool {
	if event.AuthFailed {
		return false
	}
	if event.Kind != entities.OCPIEventSession && event.Kind != entities.OCPIEventCDR {
		return false
	}
	return event.TokenUID != "" || event.ContractID != ""
}

// verifyEcho compares the token echoed back on a Session/CDR against the most
// recent command we sent for the same token uid, falling back to the most
// recent command sent when no uid matches. It sets EchoOK and EchoDiff in
// place. When the partner has never sent a command, EchoOK stays nil.
func verifyEcho(event *entities.OCPIEvent, history []entities.OCPIEvent) {
	origin := findCommandSent(history, event.TokenUID)
	if origin == nil {
		return
	}

	diff := make([]string, 0, 3)
	if origin.TokenUID != event.TokenUID {
		diff = append(diff, "uid")
	}
	if origin.TokenType != event.TokenType {
		diff = append(diff, "type")
	}
	if origin.ContractID != event.ContractID {
		diff = append(diff, "contract_id")
	}

	ok := len(diff) == 0
	event.EchoOK = &ok
	event.EchoAgainstCommandUID = origin.CommandUID
	if !ok {
		event.EchoDiff = diff
	}
}

// findCommandSent returns the most recent command_sent event, preferring one
// whose token uid matches the given uid.
func findCommandSent(history []entities.OCPIEvent, tokenUID string) *entities.OCPIEvent {
	var fallback *entities.OCPIEvent

	for i := len(history) - 1; i >= 0; i-- {
		candidate := history[i]
		if candidate.Kind != entities.OCPIEventCommandSent {
			continue
		}
		if tokenUID != "" && candidate.TokenUID == tokenUID {
			return &candidate
		}
		if fallback == nil {
			fallback = &candidate
		}
	}
	return fallback
}

// appendCapped appends an event, dropping the oldest once the cap is reached.
func appendCapped(buffer []entities.OCPIEvent, event entities.OCPIEvent) []entities.OCPIEvent {
	buffer = append(buffer, event)
	if len(buffer) > OCPIEventBufferSize {
		buffer = buffer[len(buffer)-OCPIEventBufferSize:]
	}
	return buffer
}
