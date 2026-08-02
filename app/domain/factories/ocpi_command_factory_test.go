package factories

import (
	"encoding/json"
	"testing"
	"time"

	"EV-Client-Simulator/app/domain/entities"
)

func mockNayaxPartner() *entities.OCPIPartner {
	return &entities.OCPIPartner{
		Slug:          "nayax-sim",
		Name:          "Nayax Simulator",
		PartyID:       "NYX",
		CountryCode:   "BR",
		TokenToCallUs: "token-b",
		TokenExpected: "token-a",
		OCPIBaseURL:   "https://ocpi-dev.nucharge.com.br",
		PublicBaseURL: "https://sim-dev.nucharge.com.br",
	}
}

func TestBuildResponseURL(t *testing.T) {
	tests := []struct {
		name        string
		inputPublic string
		inputSlug   string
		inputType   string
		inputUID    string
		expectedURL string
	}{
		{
			name:        "start session callback",
			inputPublic: "https://sim-dev.nucharge.com.br",
			inputSlug:   "nayax-sim",
			inputType:   entities.CommandStartSession,
			inputUID:    "abc-123",
			expectedURL: "https://sim-dev.nucharge.com.br/ocpi/p/nayax-sim/commands/START_SESSION/abc-123",
		},
		{
			name:        "stop session callback",
			inputPublic: "https://sim-dev.nucharge.com.br",
			inputSlug:   "nayax-sim",
			inputType:   entities.CommandStopSession,
			inputUID:    "abc-124",
			expectedURL: "https://sim-dev.nucharge.com.br/ocpi/p/nayax-sim/commands/STOP_SESSION/abc-124",
		},
		{
			name:        "localhost with port",
			inputPublic: "http://localhost:8080",
			inputSlug:   "other",
			inputType:   entities.CommandStartSession,
			inputUID:    "uid-1",
			expectedURL: "http://localhost:8080/ocpi/p/other/commands/START_SESSION/uid-1",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockPartner := &entities.OCPIPartner{Slug: test.inputSlug, PublicBaseURL: test.inputPublic}

			actualURL := BuildResponseURL(mockPartner, test.inputType, test.inputUID)

			if actualURL != test.expectedURL {
				t.Errorf("BuildResponseURL() = %q, expected %q", actualURL, test.expectedURL)
			}
		})
	}
}

func TestBuildCommandURL(t *testing.T) {
	mockPartner := mockNayaxPartner()

	tests := []struct {
		name        string
		inputType   string
		expectedURL string
	}{
		{
			name:        "start session target",
			inputType:   entities.CommandStartSession,
			expectedURL: "https://ocpi-dev.nucharge.com.br/ocpi/cpo/2.2.1/commands/START_SESSION",
		},
		{
			name:        "stop session target",
			inputType:   entities.CommandStopSession,
			expectedURL: "https://ocpi-dev.nucharge.com.br/ocpi/cpo/2.2.1/commands/STOP_SESSION",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualURL := BuildCommandURL(mockPartner, test.inputType)

			if actualURL != test.expectedURL {
				t.Errorf("BuildCommandURL() = %q, expected %q", actualURL, test.expectedURL)
			}
		})
	}
}

func TestCreateStartSessionCommand(t *testing.T) {
	inputNow := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)

	actualCommand := CreateStartSessionCommand(StartSessionParams{
		Partner:     mockNayaxPartner(),
		CommandUID:  "cmd-uid-1",
		TokenUID:    "NYX_deadbeef",
		ContractID:  "PAY0123456789AB",
		LocationID:  "LOC-1",
		EvseUID:     "EVSE-1",
		ConnectorID: "1",
		Now:         inputNow,
	})

	expectedCommand := entities.StartSessionCommand{
		ResponseURL: "https://sim-dev.nucharge.com.br/ocpi/p/nayax-sim/commands/START_SESSION/cmd-uid-1",
		Token: entities.OCPIToken{
			CountryCode: "BR",
			PartyID:     "NYX",
			UID:         "NYX_deadbeef",
			Type:        "AD_HOC_USER",
			ContractID:  "PAY0123456789AB",
			Issuer:      "Nayax Simulator",
			Valid:       true,
			Whitelist:   "ALLOWED_OFFLINE",
			LastUpdated: inputNow,
		},
		LocationID:  "LOC-1",
		EvseUID:     "EVSE-1",
		ConnectorID: "1",
	}

	if actualCommand != expectedCommand {
		t.Errorf("CreateStartSessionCommand() = %+v, expected %+v", actualCommand, expectedCommand)
	}
}

// TestCreateStartSessionCommandJSON pins the wire format, since our platform
// parses these field names verbatim.
func TestCreateStartSessionCommandJSON(t *testing.T) {
	inputNow := time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC)

	command := CreateStartSessionCommand(StartSessionParams{
		Partner:     mockNayaxPartner(),
		CommandUID:  "cmd-uid-1",
		TokenUID:    "NYX_deadbeef",
		ContractID:  "PAY0123456789AB",
		LocationID:  "LOC-1",
		EvseUID:     "EVSE-1",
		ConnectorID: "1",
		Now:         inputNow,
	})

	encoded, err := json.Marshal(command)
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}

	expectedJSON := `{"response_url":"https://sim-dev.nucharge.com.br/ocpi/p/nayax-sim/commands/START_SESSION/cmd-uid-1",` +
		`"token":{"country_code":"BR","party_id":"NYX","uid":"NYX_deadbeef","type":"AD_HOC_USER","contract_id":"PAY0123456789AB",` +
		`"issuer":"Nayax Simulator","valid":true,"whitelist":"ALLOWED_OFFLINE","last_updated":"2026-08-01T12:30:00Z"},` +
		`"location_id":"LOC-1","evse_uid":"EVSE-1","connector_id":"1"}`

	if string(encoded) != expectedJSON {
		t.Errorf("START_SESSION JSON =\n%s\nexpected\n%s", encoded, expectedJSON)
	}
}

func TestCreateStartSessionCommandOmitsEmptyEvseAndConnector(t *testing.T) {
	command := CreateStartSessionCommand(StartSessionParams{
		Partner:    mockNayaxPartner(),
		CommandUID: "cmd-uid-1",
		TokenUID:   "NYX_deadbeef",
		ContractID: "PAY1",
		LocationID: "LOC-1",
		Now:        time.Date(2026, 8, 1, 12, 30, 0, 0, time.UTC),
	})

	encoded, err := json.Marshal(command)
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() returned an error: %v", err)
	}

	if _, present := decoded["evse_uid"]; present {
		t.Errorf("evse_uid should be omitted when empty")
	}
	if _, present := decoded["connector_id"]; present {
		t.Errorf("connector_id should be omitted when empty")
	}
}

func TestCreateStopSessionCommand(t *testing.T) {
	actualCommand := CreateStopSessionCommand(mockNayaxPartner(), "cmd-uid-2", "sess-1")

	expectedCommand := entities.StopSessionCommand{
		ResponseURL: "https://sim-dev.nucharge.com.br/ocpi/p/nayax-sim/commands/STOP_SESSION/cmd-uid-2",
		SessionID:   "sess-1",
	}
	if actualCommand != expectedCommand {
		t.Errorf("CreateStopSessionCommand() = %+v, expected %+v", actualCommand, expectedCommand)
	}
}

func TestNewTokenUID(t *testing.T) {
	tests := []struct {
		name        string
		inputParty  string
		inputRandom string
		expectedUID string
	}{
		{name: "nayax token", inputParty: "NYX", inputRandom: "deadbeef", expectedUID: "NYX_deadbeef"},
		{name: "other party", inputParty: "ABC", inputRandom: "0011aabb", expectedUID: "ABC_0011aabb"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualUID := NewTokenUID(test.inputParty, test.inputRandom)

			if actualUID != test.expectedUID {
				t.Errorf("NewTokenUID() = %q, expected %q", actualUID, test.expectedUID)
			}
		})
	}
}
