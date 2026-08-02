package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/infrastructure/persistence"
)

// appendRawLine writes a literal line to the JSONL log, bypassing encoding, to
// simulate a truncated or corrupted write.
func appendRawLine(t *testing.T, path, line string) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("could not open %s: %v", path, err)
	}
	defer file.Close()

	if _, err := file.WriteString(line + "\n"); err != nil {
		t.Fatalf("could not write to %s: %v", path, err)
	}
}

func TestOCPIEventServiceRingBuffer(t *testing.T) {
	tests := []struct {
		name          string
		inputCount    int
		expectedLen   int
		expectedFirst int64
		expectedLast  int64
	}{
		{name: "below the cap keeps everything", inputCount: 10, expectedLen: 10, expectedFirst: 1, expectedLast: 10},
		{name: "at the cap keeps everything", inputCount: OCPIEventBufferSize, expectedLen: OCPIEventBufferSize, expectedFirst: 1, expectedLast: OCPIEventBufferSize},
		{
			name:          "above the cap drops the oldest",
			inputCount:    OCPIEventBufferSize + 25,
			expectedLen:   OCPIEventBufferSize,
			expectedFirst: 26,
			expectedLast:  OCPIEventBufferSize + 25,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewOCPIEventService(nil)

			for i := 0; i < test.inputCount; i++ {
				mockService.Record(entities.OCPIEvent{PartnerSlug: "nayax", Kind: entities.OCPIEventLocation})
			}

			actualEvents := mockService.List("nayax", 0, 0)

			if len(actualEvents) != test.expectedLen {
				t.Fatalf("List() returned %d events, expected %d", len(actualEvents), test.expectedLen)
			}
			if actualEvents[0].ID != test.expectedFirst {
				t.Errorf("first event ID = %d, expected %d", actualEvents[0].ID, test.expectedFirst)
			}
			if actualEvents[len(actualEvents)-1].ID != test.expectedLast {
				t.Errorf("last event ID = %d, expected %d", actualEvents[len(actualEvents)-1].ID, test.expectedLast)
			}
		})
	}
}

func TestOCPIEventServiceIsolatesPartners(t *testing.T) {
	mockService := NewOCPIEventService(nil)

	mockService.Record(entities.OCPIEvent{PartnerSlug: "nayax", Kind: entities.OCPIEventSession})
	mockService.Record(entities.OCPIEvent{PartnerSlug: "other", Kind: entities.OCPIEventSession})
	mockService.Record(entities.OCPIEvent{PartnerSlug: "nayax", Kind: entities.OCPIEventCDR})

	actualNayax := mockService.List("nayax", 0, 0)
	actualOther := mockService.List("other", 0, 0)

	if len(actualNayax) != 2 {
		t.Errorf("nayax buffer has %d events, expected 2", len(actualNayax))
	}
	if len(actualOther) != 1 {
		t.Errorf("other buffer has %d events, expected 1", len(actualOther))
	}
	if actualNayax[0].ID != 1 || actualNayax[1].ID != 3 {
		t.Errorf("nayax event IDs = %d,%d, expected 1,3", actualNayax[0].ID, actualNayax[1].ID)
	}
}

func TestOCPIEventServiceListAfterCursor(t *testing.T) {
	mockService := NewOCPIEventService(nil)
	for i := 0; i < 5; i++ {
		mockService.Record(entities.OCPIEvent{PartnerSlug: "nayax", Kind: entities.OCPIEventLocation})
	}

	tests := []struct {
		name        string
		inputAfter  int64
		inputLimit  int
		expectedIDs []int64
	}{
		{name: "no cursor returns all", inputAfter: 0, inputLimit: 0, expectedIDs: []int64{1, 2, 3, 4, 5}},
		{name: "cursor skips consumed events", inputAfter: 3, inputLimit: 0, expectedIDs: []int64{4, 5}},
		{name: "cursor at the end returns nothing", inputAfter: 5, inputLimit: 0, expectedIDs: []int64{}},
		{name: "future cursor returns nothing", inputAfter: 99, inputLimit: 0, expectedIDs: []int64{}},
		{name: "limit keeps the newest", inputAfter: 0, inputLimit: 2, expectedIDs: []int64{4, 5}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualEvents := mockService.List("nayax", test.inputAfter, test.inputLimit)

			if len(actualEvents) != len(test.expectedIDs) {
				t.Fatalf("List() returned %d events, expected %d", len(actualEvents), len(test.expectedIDs))
			}
			for i, expectedID := range test.expectedIDs {
				if actualEvents[i].ID != expectedID {
					t.Errorf("event[%d].ID = %d, expected %d", i, actualEvents[i].ID, expectedID)
				}
			}
		})
	}
}

func TestOCPIEventServicePersistsAcrossRestart(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "ocpi-events.jsonl")

	originalService := NewOCPIEventService(persistence.NewOCPIEventLog(inputPath))
	originalService.Record(entities.OCPIEvent{PartnerSlug: "nayax", Kind: entities.OCPIEventCommandSent, TokenUID: "NYX_a1"})
	originalService.Record(entities.OCPIEvent{PartnerSlug: "nayax", Kind: entities.OCPIEventSession, SessionID: "sess-1"})
	originalService.Record(entities.OCPIEvent{PartnerSlug: "other", Kind: entities.OCPIEventCDR})

	restartedService := NewOCPIEventService(persistence.NewOCPIEventLog(inputPath))
	if err := restartedService.Load(); err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	actualNayax := restartedService.List("nayax", 0, 0)
	if len(actualNayax) != 2 {
		t.Fatalf("restored nayax buffer has %d events, expected 2", len(actualNayax))
	}
	if actualNayax[1].SessionID != "sess-1" {
		t.Errorf("restored session ID = %q, expected %q", actualNayax[1].SessionID, "sess-1")
	}
	if len(restartedService.List("other", 0, 0)) != 1 {
		t.Errorf("restored other buffer should hold 1 event")
	}

	// IDs must keep increasing so the UI cursor never replays an old event.
	actualNext := restartedService.Record(entities.OCPIEvent{PartnerSlug: "nayax", Kind: entities.OCPIEventCDR})
	if actualNext.ID != 4 {
		t.Errorf("next event ID = %d, expected 4", actualNext.ID)
	}
}

func TestOCPIEventServiceLoadTrimsToBufferSize(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "ocpi-events.jsonl")
	inputLog := persistence.NewOCPIEventLog(inputPath)

	for i := 1; i <= OCPIEventBufferSize+50; i++ {
		if err := inputLog.Append(entities.OCPIEvent{ID: int64(i), PartnerSlug: "nayax", Kind: entities.OCPIEventLocation}); err != nil {
			t.Fatalf("Append() returned an error: %v", err)
		}
	}

	actualService := NewOCPIEventService(inputLog)
	if err := actualService.Load(); err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	actualEvents := actualService.List("nayax", 0, 0)
	if len(actualEvents) != OCPIEventBufferSize {
		t.Fatalf("restored buffer has %d events, expected %d", len(actualEvents), OCPIEventBufferSize)
	}
	if actualEvents[0].ID != 51 {
		t.Errorf("oldest restored event ID = %d, expected 51", actualEvents[0].ID)
	}
}

func TestOCPIEventLogSkipsMalformedLines(t *testing.T) {
	inputPath := filepath.Join(t.TempDir(), "ocpi-events.jsonl")
	inputLog := persistence.NewOCPIEventLog(inputPath)

	if err := inputLog.Append(entities.OCPIEvent{ID: 1, PartnerSlug: "nayax"}); err != nil {
		t.Fatalf("Append() returned an error: %v", err)
	}
	appendRawLine(t, inputPath, "{not json")
	if err := inputLog.Append(entities.OCPIEvent{ID: 2, PartnerSlug: "nayax"}); err != nil {
		t.Fatalf("Append() returned an error: %v", err)
	}

	actualEvents, err := inputLog.LoadTail(100)
	if err != nil {
		t.Fatalf("LoadTail() returned an error: %v", err)
	}
	if len(actualEvents) != 2 {
		t.Errorf("LoadTail() returned %d events, expected 2 (malformed line skipped)", len(actualEvents))
	}
}

func TestOCPIEventLogMissingFileIsEmpty(t *testing.T) {
	actualEvents, err := persistence.NewOCPIEventLog(filepath.Join(t.TempDir(), "missing.jsonl")).LoadTail(100)

	if err != nil {
		t.Fatalf("LoadTail() returned an error: %v", err)
	}
	if len(actualEvents) != 0 {
		t.Errorf("LoadTail() returned %d events, expected 0", len(actualEvents))
	}
}

func TestOCPIEventServiceEchoVerification(t *testing.T) {
	type commandSent struct {
		tokenUID   string
		tokenType  string
		contractID string
		commandUID string
	}

	tests := []struct {
		name            string
		inputCommands   []commandSent
		inputKind       entities.OCPIEventKind
		inputTokenUID   string
		inputTokenType  string
		inputContractID string
		expectedEchoSet bool
		expectedEchoOK  bool
		expectedDiff    []string
		expectedAgainst string
	}{
		{
			name:            "matching echo passes",
			inputCommands:   []commandSent{{tokenUID: "NYX_a1", tokenType: "AD_HOC_USER", contractID: "PAY1", commandUID: "cmd-1"}},
			inputKind:       entities.OCPIEventSession,
			inputTokenUID:   "NYX_a1",
			inputTokenType:  "AD_HOC_USER",
			inputContractID: "PAY1",
			expectedEchoSet: true,
			expectedEchoOK:  true,
			expectedAgainst: "cmd-1",
		},
		{
			name:            "mangled contract id fails",
			inputCommands:   []commandSent{{tokenUID: "NYX_a1", tokenType: "AD_HOC_USER", contractID: "PAY1", commandUID: "cmd-1"}},
			inputKind:       entities.OCPIEventCDR,
			inputTokenUID:   "NYX_a1",
			inputTokenType:  "AD_HOC_USER",
			inputContractID: "SOMETHING_ELSE",
			expectedEchoSet: true,
			expectedEchoOK:  false,
			expectedDiff:    []string{"contract_id"},
			expectedAgainst: "cmd-1",
		},
		{
			name:            "changed token type fails",
			inputCommands:   []commandSent{{tokenUID: "NYX_a1", tokenType: "AD_HOC_USER", contractID: "PAY1", commandUID: "cmd-1"}},
			inputKind:       entities.OCPIEventSession,
			inputTokenUID:   "NYX_a1",
			inputTokenType:  "APP_USER",
			inputContractID: "PAY1",
			expectedEchoSet: true,
			expectedEchoOK:  false,
			expectedDiff:    []string{"type"},
			expectedAgainst: "cmd-1",
		},
		{
			name: "matches the command with the same token uid, not the newest",
			inputCommands: []commandSent{
				{tokenUID: "NYX_a1", tokenType: "AD_HOC_USER", contractID: "PAY1", commandUID: "cmd-1"},
				{tokenUID: "NYX_b2", tokenType: "AD_HOC_USER", contractID: "PAY2", commandUID: "cmd-2"},
			},
			inputKind:       entities.OCPIEventSession,
			inputTokenUID:   "NYX_a1",
			inputTokenType:  "AD_HOC_USER",
			inputContractID: "PAY1",
			expectedEchoSet: true,
			expectedEchoOK:  true,
			expectedAgainst: "cmd-1",
		},
		{
			name: "unknown token uid falls back to the newest command and fails",
			inputCommands: []commandSent{
				{tokenUID: "NYX_a1", tokenType: "AD_HOC_USER", contractID: "PAY1", commandUID: "cmd-1"},
				{tokenUID: "NYX_b2", tokenType: "AD_HOC_USER", contractID: "PAY2", commandUID: "cmd-2"},
			},
			inputKind:       entities.OCPIEventSession,
			inputTokenUID:   "SOMETHING_ELSE",
			inputTokenType:  "AD_HOC_USER",
			inputContractID: "PAY2",
			expectedEchoSet: true,
			expectedEchoOK:  false,
			expectedDiff:    []string{"uid"},
			expectedAgainst: "cmd-2",
		},
		{
			name:            "no command history leaves the verdict unset",
			inputCommands:   nil,
			inputKind:       entities.OCPIEventSession,
			inputTokenUID:   "NYX_a1",
			inputTokenType:  "AD_HOC_USER",
			inputContractID: "PAY1",
			expectedEchoSet: false,
		},
		{
			name:            "locations are not echo checked",
			inputCommands:   []commandSent{{tokenUID: "NYX_a1", tokenType: "AD_HOC_USER", contractID: "PAY1", commandUID: "cmd-1"}},
			inputKind:       entities.OCPIEventLocation,
			inputTokenUID:   "NYX_a1",
			inputTokenType:  "AD_HOC_USER",
			inputContractID: "PAY1",
			expectedEchoSet: false,
		},
		{
			name:            "session without a token is not echo checked",
			inputCommands:   []commandSent{{tokenUID: "NYX_a1", tokenType: "AD_HOC_USER", contractID: "PAY1", commandUID: "cmd-1"}},
			inputKind:       entities.OCPIEventSession,
			expectedEchoSet: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockService := NewOCPIEventService(nil)
			for _, command := range test.inputCommands {
				mockService.Record(entities.OCPIEvent{
					PartnerSlug: "nayax",
					Kind:        entities.OCPIEventCommandSent,
					CommandUID:  command.commandUID,
					TokenUID:    command.tokenUID,
					TokenType:   command.tokenType,
					ContractID:  command.contractID,
				})
			}

			actualEvent := mockService.Record(entities.OCPIEvent{
				PartnerSlug: "nayax",
				Kind:        test.inputKind,
				TokenUID:    test.inputTokenUID,
				TokenType:   test.inputTokenType,
				ContractID:  test.inputContractID,
			})

			if !test.expectedEchoSet {
				if actualEvent.EchoOK != nil {
					t.Fatalf("EchoOK = %v, expected nil", *actualEvent.EchoOK)
				}
				return
			}

			if actualEvent.EchoOK == nil {
				t.Fatalf("EchoOK = nil, expected %v", test.expectedEchoOK)
			}
			if *actualEvent.EchoOK != test.expectedEchoOK {
				t.Errorf("EchoOK = %v, expected %v", *actualEvent.EchoOK, test.expectedEchoOK)
			}
			if actualEvent.EchoAgainstCommandUID != test.expectedAgainst {
				t.Errorf("EchoAgainstCommandUID = %q, expected %q", actualEvent.EchoAgainstCommandUID, test.expectedAgainst)
			}
			if fmt.Sprint(actualEvent.EchoDiff) != fmt.Sprint(test.expectedDiff) {
				t.Errorf("EchoDiff = %v, expected %v", actualEvent.EchoDiff, test.expectedDiff)
			}
		})
	}
}

func TestOCPIEventServiceSkipsEchoOnAuthFailure(t *testing.T) {
	mockService := NewOCPIEventService(nil)
	mockService.Record(entities.OCPIEvent{
		PartnerSlug: "nayax",
		Kind:        entities.OCPIEventCommandSent,
		CommandUID:  "cmd-1",
		TokenUID:    "NYX_a1",
		ContractID:  "PAY1",
	})

	actualEvent := mockService.Record(entities.OCPIEvent{
		PartnerSlug: "nayax",
		Kind:        entities.OCPIEventAuthFailed,
		AuthFailed:  true,
		TokenUID:    "NYX_a1",
		ContractID:  "PAY1",
	})

	if actualEvent.EchoOK != nil {
		t.Errorf("EchoOK = %v, expected nil for an auth failure", *actualEvent.EchoOK)
	}
}

func TestOCPIEventServiceClear(t *testing.T) {
	mockService := NewOCPIEventService(nil)
	mockService.Record(entities.OCPIEvent{PartnerSlug: "nayax", Kind: entities.OCPIEventSession})
	mockService.Record(entities.OCPIEvent{PartnerSlug: "other", Kind: entities.OCPIEventSession})

	mockService.Clear("nayax")

	if len(mockService.List("nayax", 0, 0)) != 0 {
		t.Errorf("cleared partner should have no events")
	}
	if len(mockService.List("other", 0, 0)) != 1 {
		t.Errorf("clearing one partner must not touch another")
	}
}

func TestExtractCorrelation(t *testing.T) {
	tests := []struct {
		name               string
		inputKind          entities.OCPIEventKind
		inputBody          string
		expectedSessionID  string
		expectedTokenUID   string
		expectedTokenType  string
		expectedContractID string
	}{
		{
			name:               "session push",
			inputKind:          entities.OCPIEventSession,
			inputBody:          `{"id":"sess-1","cdr_token":{"uid":"NYX_a1","type":"AD_HOC_USER","contract_id":"PAY1"}}`,
			expectedSessionID:  "sess-1",
			expectedTokenUID:   "NYX_a1",
			expectedTokenType:  "AD_HOC_USER",
			expectedContractID: "PAY1",
		},
		{
			name:               "cdr prefers session_id",
			inputKind:          entities.OCPIEventCDR,
			inputBody:          `{"id":"cdr-9","session_id":"sess-1","cdr_token":{"uid":"NYX_a1","type":"AD_HOC_USER","contract_id":"PAY1"}}`,
			expectedSessionID:  "sess-1",
			expectedTokenUID:   "NYX_a1",
			expectedTokenType:  "AD_HOC_USER",
			expectedContractID: "PAY1",
		},
		{
			name:              "cdr without session_id falls back to id",
			inputKind:         entities.OCPIEventCDR,
			inputBody:         `{"id":"cdr-9"}`,
			expectedSessionID: "cdr-9",
		},
		{
			name:              "session patch without a token",
			inputKind:         entities.OCPIEventSession,
			inputBody:         `{"id":"sess-1","kwh":12.5}`,
			expectedSessionID: "sess-1",
		},
		{
			name:      "location push carries no session",
			inputKind: entities.OCPIEventLocation,
			inputBody: `{"id":"LOC1","name":"Praca"}`,
		},
		{
			name:      "malformed body yields empty values",
			inputKind: entities.OCPIEventSession,
			inputBody: `{not json`,
		},
		{
			name:      "empty body yields empty values",
			inputKind: entities.OCPIEventSession,
			inputBody: ``,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualSessionID, actualToken := ExtractCorrelation(test.inputKind, []byte(test.inputBody))

			if actualSessionID != test.expectedSessionID {
				t.Errorf("sessionID = %q, expected %q", actualSessionID, test.expectedSessionID)
			}
			if actualToken.UID != test.expectedTokenUID {
				t.Errorf("token.UID = %q, expected %q", actualToken.UID, test.expectedTokenUID)
			}
			if actualToken.Type != test.expectedTokenType {
				t.Errorf("token.Type = %q, expected %q", actualToken.Type, test.expectedTokenType)
			}
			if actualToken.ContractID != test.expectedContractID {
				t.Errorf("token.ContractID = %q, expected %q", actualToken.ContractID, test.expectedContractID)
			}
		})
	}
}

// TestOCPIEventServiceEndToEndEcho walks the exact flow the pilot cares about:
// a START_SESSION is sent, then our platform pushes back a Session and a CDR
// carrying the cdr_token it received.
func TestOCPIEventServiceEndToEndEcho(t *testing.T) {
	mockService := NewOCPIEventService(nil)
	mockService.Record(entities.OCPIEvent{
		PartnerSlug: "nayax",
		Kind:        entities.OCPIEventCommandSent,
		CommandUID:  "cmd-1",
		TokenUID:    "NYX_deadbeef",
		TokenType:   "AD_HOC_USER",
		ContractID:  "PAY0123456789AB",
	})

	inputSessionBody := `{"id":"sess-1","cdr_token":{"uid":"NYX_deadbeef","type":"AD_HOC_USER","contract_id":"PAY0123456789AB"}}`
	sessionID, token := ExtractCorrelation(entities.OCPIEventSession, []byte(inputSessionBody))

	actualEvent := mockService.Record(entities.OCPIEvent{
		PartnerSlug: "nayax",
		Kind:        entities.OCPIEventSession,
		Body:        json.RawMessage(inputSessionBody),
		SessionID:   sessionID,
		TokenUID:    token.UID,
		TokenType:   token.Type,
		ContractID:  token.ContractID,
	})

	if actualEvent.EchoOK == nil || !*actualEvent.EchoOK {
		t.Fatalf("expected the echoed cdr_token to verify, got %+v (diff %v)", actualEvent.EchoOK, actualEvent.EchoDiff)
	}
	if actualEvent.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, expected %q", actualEvent.SessionID, "sess-1")
	}
}
