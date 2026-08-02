package persistence

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"EV-Client-Simulator/app/domain/entities"
)

// maxScanTokenSize bounds a single JSONL line; OCPI CDR bodies can be large.
const maxScanTokenSize = 4 * 1024 * 1024

// OCPIEventLog appends events to a JSONL file so the feed survives restarts.
type OCPIEventLog struct {
	path string
	mu   sync.Mutex
}

// NewOCPIEventLog creates an event log backed by the given file path.
func NewOCPIEventLog(path string) *OCPIEventLog {
	return &OCPIEventLog{path: path}
}

// Path returns the backing file path.
func (l *OCPIEventLog) Path() string {
	return l.path
}

// Append writes a single event as one JSON line.
func (l *OCPIEventLog) Append(event entities.OCPIEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	line, err := json.Marshal(event)
	if err != nil {
		return err
	}

	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	return nil
}

// LoadTail reads the last `limit` events from the log. Malformed lines are
// skipped so a partially written line never blocks startup.
func (l *OCPIEventLog) LoadTail(limit int) ([]entities.OCPIEvent, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.Open(l.path)
	if os.IsNotExist(err) {
		return []entities.OCPIEvent{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()

	events := make([]entities.OCPIEvent, 0, limit)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxScanTokenSize)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event entities.OCPIEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		events = append(events, event)
		if limit > 0 && len(events) > limit {
			events = events[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		return events, err
	}
	return events, nil
}
