package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"EV-Client-Simulator/app/domain/abstracts"
	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/infrastructure/persistence"
	"EV-Client-Simulator/interface/http/dto"
	"EV-Client-Simulator/interface/http/middleware"

	"github.com/go-chi/chi/v5"
)

// mockCommandClient records the last dispatched command instead of sending it.
type mockCommandClient struct {
	lastURL     string
	lastToken   string
	lastPayload interface{}
	response    *abstracts.OCPICommandResponse
	err         error
}

func (c *mockCommandClient) PostCommand(url, token string, payload interface{}) (*abstracts.OCPICommandResponse, error) {
	c.lastURL = url
	c.lastToken = token
	c.lastPayload = payload
	if c.err != nil {
		return nil, c.err
	}
	if c.response != nil {
		return c.response, nil
	}
	return &abstracts.OCPICommandResponse{StatusCode: 200, Body: []byte(`{"status_code":1000}`)}, nil
}

type ocpiTestEnv struct {
	router   *chi.Mux
	partners *services.OCPIPartnerService
	events   *services.OCPIEventService
	client   *mockCommandClient
}

func newOCPITestEnv(t *testing.T, simUser, simPass string) *ocpiTestEnv {
	t.Helper()

	dir := t.TempDir()
	partnerService := services.NewOCPIPartnerService(persistence.NewOCPIPartnerStore(filepath.Join(dir, "partners.json")))
	eventService := services.NewOCPIEventService(persistence.NewOCPIEventLog(filepath.Join(dir, "events.jsonl")))
	client := &mockCommandClient{}

	if _, err := partnerService.Create(entities.OCPIPartner{
		Slug:          "nayax-sim",
		Name:          "Nayax Simulator",
		PartyID:       "NYX",
		CountryCode:   "BR",
		TokenToCallUs: "token-b",
		TokenExpected: "token-a",
		OCPIBaseURL:   "https://ocpi-dev.nucharge.com.br",
		PublicBaseURL: "https://sim-dev.nucharge.com.br",
	}); err != nil {
		t.Fatalf("could not seed partner: %v", err)
	}

	settingsService := services.NewAppSettingsService(entities.AppSettings{
		OCPIBaseURL:        "https://ocpi-dev.nucharge.com.br",
		PublicBaseURL:      "https://sim-dev.nucharge.com.br",
		DefaultLocationID:  "LOC-DEFAULT",
		DefaultEvseUID:     "EVSE-DEFAULT",
		BatteryCapacityKWh: 60,
	}, nil)

	router := chi.NewRouter()
	RegisterOCPIRoutes(router, OCPIDependencies{
		Partners:  partnerService,
		Events:    eventService,
		Commands:  services.NewOCPICommandService(partnerService, eventService, client),
		Settings:  settingsService,
		BasicAuth: middleware.BasicAuth("test", simUser, simPass),
	})

	return &ocpiTestEnv{router: router, partners: partnerService, events: eventService, client: client}
}

func (e *ocpiTestEnv) do(t *testing.T, method, path, token string, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}

	req := httptest.NewRequest(method, path, reader)
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	recorder := httptest.NewRecorder()
	e.router.ServeHTTP(recorder, req)
	return recorder
}

func TestReceiverTokenValidation(t *testing.T) {
	tests := []struct {
		name               string
		inputMethod        string
		inputPath          string
		inputToken         string
		expectedStatus     int
		expectedOCPIStatus int
		expectedKind       entities.OCPIEventKind
	}{
		{
			name:               "session put with the right token",
			inputMethod:        http.MethodPut,
			inputPath:          "/ocpi/p/nayax-sim/receiver/2.2.1/sessions/BR/NYX/sess-1",
			inputToken:         "Token token-a",
			expectedStatus:     http.StatusOK,
			expectedOCPIStatus: entities.OCPIStatusSuccess,
			expectedKind:       entities.OCPIEventSession,
		},
		{
			name:               "session patch with the right token",
			inputMethod:        http.MethodPatch,
			inputPath:          "/ocpi/p/nayax-sim/receiver/2.2.1/sessions/BR/NYX/sess-1",
			inputToken:         "Token token-a",
			expectedStatus:     http.StatusOK,
			expectedOCPIStatus: entities.OCPIStatusSuccess,
			expectedKind:       entities.OCPIEventSession,
		},
		{
			name:               "cdr post with the right token",
			inputMethod:        http.MethodPost,
			inputPath:          "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs",
			inputToken:         "Token token-a",
			expectedStatus:     http.StatusOK,
			expectedOCPIStatus: entities.OCPIStatusSuccess,
			expectedKind:       entities.OCPIEventCDR,
		},
		{
			name:               "location put with the right token",
			inputMethod:        http.MethodPut,
			inputPath:          "/ocpi/p/nayax-sim/receiver/2.2.1/locations/BR/NYX/LOC1",
			inputToken:         "Token token-a",
			expectedStatus:     http.StatusOK,
			expectedOCPIStatus: entities.OCPIStatusSuccess,
			expectedKind:       entities.OCPIEventLocation,
		},
		{
			name:               "tariff delete with the right token",
			inputMethod:        http.MethodDelete,
			inputPath:          "/ocpi/p/nayax-sim/receiver/2.2.1/tariffs/BR/NYX/TAR1",
			inputToken:         "Token token-a",
			expectedStatus:     http.StatusOK,
			expectedOCPIStatus: entities.OCPIStatusSuccess,
			expectedKind:       entities.OCPIEventTariff,
		},
		{
			name:               "command result callback",
			inputMethod:        http.MethodPost,
			inputPath:          "/ocpi/p/nayax-sim/commands/START_SESSION/cmd-1",
			inputToken:         "Token token-a",
			expectedStatus:     http.StatusOK,
			expectedOCPIStatus: entities.OCPIStatusSuccess,
			expectedKind:       entities.OCPIEventCommandResult,
		},
		{
			name:               "wrong token is rejected but logged",
			inputMethod:        http.MethodPut,
			inputPath:          "/ocpi/p/nayax-sim/receiver/2.2.1/sessions/BR/NYX/sess-1",
			inputToken:         "Token wrong",
			expectedStatus:     http.StatusUnauthorized,
			expectedOCPIStatus: entities.OCPIStatusInvalidParameters,
			expectedKind:       entities.OCPIEventAuthFailed,
		},
		{
			name:               "missing token is rejected but logged",
			inputMethod:        http.MethodPost,
			inputPath:          "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs",
			inputToken:         "",
			expectedStatus:     http.StatusUnauthorized,
			expectedOCPIStatus: entities.OCPIStatusInvalidParameters,
			expectedKind:       entities.OCPIEventAuthFailed,
		},
		{
			name:               "basic auth on a receiver route is rejected",
			inputMethod:        http.MethodPost,
			inputPath:          "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs",
			inputToken:         "Basic dG9rZW4tYQ==",
			expectedStatus:     http.StatusUnauthorized,
			expectedOCPIStatus: entities.OCPIStatusInvalidParameters,
			expectedKind:       entities.OCPIEventAuthFailed,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockEnv := newOCPITestEnv(t, "admin", "secret")

			actualResponse := mockEnv.do(t, test.inputMethod, test.inputPath, test.inputToken, `{"id":"sess-1"}`)

			if actualResponse.Code != test.expectedStatus {
				t.Fatalf("status = %d, expected %d (body %s)", actualResponse.Code, test.expectedStatus, actualResponse.Body)
			}

			var envelope entities.OCPIEnvelope
			if err := json.Unmarshal(actualResponse.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("response is not an OCPI envelope: %v (body %s)", err, actualResponse.Body)
			}
			if envelope.StatusCode != test.expectedOCPIStatus {
				t.Errorf("status_code = %d, expected %d", envelope.StatusCode, test.expectedOCPIStatus)
			}
			if envelope.Timestamp == "" {
				t.Errorf("envelope is missing a timestamp")
			}

			actualEvents := mockEnv.events.List("nayax-sim", 0, 0)
			if len(actualEvents) != 1 {
				t.Fatalf("recorded %d events, expected 1", len(actualEvents))
			}
			if actualEvents[0].Kind != test.expectedKind {
				t.Errorf("event kind = %q, expected %q", actualEvents[0].Kind, test.expectedKind)
			}
			if actualEvents[0].Direction != entities.OCPIDirectionIn {
				t.Errorf("event direction = %q, expected %q", actualEvents[0].Direction, entities.OCPIDirectionIn)
			}
		})
	}
}

func TestReceiverIsExemptFromBasicAuth(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "admin", "secret")

	// No basic auth credentials at all, only the OCPI token.
	actualResponse := mockEnv.do(t, http.MethodPost, "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs", "Token token-a", `{"id":"cdr-1"}`)

	if actualResponse.Code != http.StatusOK {
		t.Errorf("status = %d, expected 200 (receiver routes must not require basic auth)", actualResponse.Code)
	}
}

func TestReceiverUnknownPartner(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	actualResponse := mockEnv.do(t, http.MethodPost, "/ocpi/p/ghost/receiver/2.2.1/cdrs", "Token token-a", `{}`)

	if actualResponse.Code != http.StatusNotFound {
		t.Errorf("status = %d, expected 404", actualResponse.Code)
	}
}

func TestReceiverCommandResultCapturesUID(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	mockEnv.do(t, http.MethodPost, "/ocpi/p/nayax-sim/commands/START_SESSION/cmd-abc", "Token token-a", `{"result":"ACCEPTED"}`)

	actualEvents := mockEnv.events.List("nayax-sim", 0, 0)
	if len(actualEvents) != 1 {
		t.Fatalf("recorded %d events, expected 1", len(actualEvents))
	}
	if actualEvents[0].CommandUID != "cmd-abc" {
		t.Errorf("CommandUID = %q, expected %q", actualEvents[0].CommandUID, "cmd-abc")
	}
}

func TestControlAPIRequiresBasicAuth(t *testing.T) {
	tests := []struct {
		name           string
		inputMethod    string
		inputPath      string
		inputAuth      string
		expectedStatus int
	}{
		{name: "list without credentials", inputMethod: http.MethodGet, inputPath: "/ocpi/api/partners", expectedStatus: http.StatusUnauthorized},
		{name: "events without credentials", inputMethod: http.MethodGet, inputPath: "/ocpi/api/partners/nayax-sim/events", expectedStatus: http.StatusUnauthorized},
		{name: "start without credentials", inputMethod: http.MethodPost, inputPath: "/ocpi/api/partners/nayax-sim/commands/start", expectedStatus: http.StatusUnauthorized},
		{
			name:           "list with wrong credentials",
			inputMethod:    http.MethodGet,
			inputPath:      "/ocpi/api/partners",
			inputAuth:      "Basic YWRtaW46bm90LXNlY3JldA==",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "list with correct credentials",
			inputMethod:    http.MethodGet,
			inputPath:      "/ocpi/api/partners",
			inputAuth:      "Basic YWRtaW46c2VjcmV0",
			expectedStatus: http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockEnv := newOCPITestEnv(t, "admin", "secret")

			actualResponse := mockEnv.do(t, test.inputMethod, test.inputPath, test.inputAuth, `{"locationId":"LOC-1"}`)

			if actualResponse.Code != test.expectedStatus {
				t.Errorf("status = %d, expected %d", actualResponse.Code, test.expectedStatus)
			}
		})
	}
}

func TestControlAPIOpenWhenAuthDisabled(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	actualResponse := mockEnv.do(t, http.MethodGet, "/ocpi/api/partners", "", "")

	if actualResponse.Code != http.StatusOK {
		t.Errorf("status = %d, expected 200 when SIM_USER is unset", actualResponse.Code)
	}
}

func TestStartSessionDispatchesCommand(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	actualResponse := mockEnv.do(t, http.MethodPost, "/ocpi/api/partners/nayax-sim/commands/start", "", `{"locationId":"LOC-1","evseUid":"EVSE-1","connectorId":"1"}`)

	if actualResponse.Code != http.StatusOK {
		t.Fatalf("status = %d, expected 200 (body %s)", actualResponse.Code, actualResponse.Body)
	}

	if mockEnv.client.lastURL != "https://ocpi-dev.nucharge.com.br/ocpi/cpo/2.2.1/commands/START_SESSION" {
		t.Errorf("target URL = %q", mockEnv.client.lastURL)
	}
	if mockEnv.client.lastToken != "token-b" {
		t.Errorf("outbound token = %q, expected %q", mockEnv.client.lastToken, "token-b")
	}

	command, ok := mockEnv.client.lastPayload.(entities.StartSessionCommand)
	if !ok {
		t.Fatalf("payload type = %T, expected StartSessionCommand", mockEnv.client.lastPayload)
	}
	if command.LocationID != "LOC-1" || command.EvseUID != "EVSE-1" || command.ConnectorID != "1" {
		t.Errorf("command targets = %+v", command)
	}
	if command.Token.Type != "AD_HOC_USER" || command.Token.ContractID == "" {
		t.Errorf("token = %+v, expected an AD_HOC_USER token with a contract id", command.Token)
	}

	actualEvents := mockEnv.events.List("nayax-sim", 0, 0)
	if len(actualEvents) != 1 || actualEvents[0].Kind != entities.OCPIEventCommandSent {
		t.Fatalf("expected one command_sent event, got %+v", actualEvents)
	}
	if actualEvents[0].ContractID != command.Token.ContractID {
		t.Errorf("event contract id = %q, expected %q", actualEvents[0].ContractID, command.Token.ContractID)
	}
}

func TestStartSessionUsesConfiguredDefaults(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	mockEnv.do(t, http.MethodPost, "/ocpi/api/partners/nayax-sim/commands/start", "", `{}`)

	command, ok := mockEnv.client.lastPayload.(entities.StartSessionCommand)
	if !ok {
		t.Fatalf("payload type = %T, expected StartSessionCommand", mockEnv.client.lastPayload)
	}
	if command.LocationID != "LOC-DEFAULT" || command.EvseUID != "EVSE-DEFAULT" {
		t.Errorf("command = %+v, expected the configured defaults to be applied", command)
	}
}

func TestStopSessionRequiresSessionID(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	actualResponse := mockEnv.do(t, http.MethodPost, "/ocpi/api/partners/nayax-sim/commands/stop", "", `{}`)

	if actualResponse.Code != http.StatusBadRequest {
		t.Errorf("status = %d, expected 400", actualResponse.Code)
	}
}

// TestFullEchoRoundTrip drives the pilot scenario end to end through HTTP: the
// partner starts a session, then our platform pushes the Session back with the
// cdr_token it was given.
func TestFullEchoRoundTrip(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	mockEnv.do(t, http.MethodPost, "/ocpi/api/partners/nayax-sim/commands/start", "", `{"locationId":"LOC-1","evseUid":"EVSE-1"}`)
	command := mockEnv.client.lastPayload.(entities.StartSessionCommand)

	sessionBody, err := json.Marshal(map[string]interface{}{
		"id": "sess-1",
		"cdr_token": map[string]string{
			"uid":         command.Token.UID,
			"type":        command.Token.Type,
			"contract_id": command.Token.ContractID,
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}

	mockEnv.do(t, http.MethodPut, "/ocpi/p/nayax-sim/receiver/2.2.1/sessions/BR/NYX/sess-1", "Token token-a", string(sessionBody))

	actualEvents := mockEnv.events.List("nayax-sim", 0, 0)
	if len(actualEvents) != 2 {
		t.Fatalf("recorded %d events, expected 2", len(actualEvents))
	}

	sessionEvent := actualEvents[1]
	if sessionEvent.EchoOK == nil || !*sessionEvent.EchoOK {
		t.Fatalf("expected the cdr_token echo to verify, got %v (diff %v)", sessionEvent.EchoOK, sessionEvent.EchoDiff)
	}
	if sessionEvent.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, expected %q", sessionEvent.SessionID, "sess-1")
	}
}

// TestEchoRoundTripDetectsMangledContractID is the regression this tool exists
// to catch: our platform replacing the partner's contract_id.
func TestEchoRoundTripDetectsMangledContractID(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	mockEnv.do(t, http.MethodPost, "/ocpi/api/partners/nayax-sim/commands/start", "", `{"locationId":"LOC-1"}`)
	command := mockEnv.client.lastPayload.(entities.StartSessionCommand)

	sessionBody, err := json.Marshal(map[string]interface{}{
		"id": "sess-1",
		"cdr_token": map[string]string{
			"uid":         command.Token.UID,
			"type":        command.Token.Type,
			"contract_id": "REPLACED-BY-PLATFORM",
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}

	mockEnv.do(t, http.MethodPost, "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs", "Token token-a", string(sessionBody))

	actualEvents := mockEnv.events.List("nayax-sim", 0, 0)
	cdrEvent := actualEvents[len(actualEvents)-1]

	if cdrEvent.EchoOK == nil || *cdrEvent.EchoOK {
		t.Fatalf("expected the echo check to fail, got %v", cdrEvent.EchoOK)
	}
	if len(cdrEvent.EchoDiff) != 1 || cdrEvent.EchoDiff[0] != "contract_id" {
		t.Errorf("EchoDiff = %v, expected [contract_id]", cdrEvent.EchoDiff)
	}
}

func TestPartnerCRUD(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	created := mockEnv.do(t, http.MethodPost, "/ocpi/api/partners", "", `{"slug":"second","name":"Second","partyId":"SEC","countryCode":"BR","tokenExpected":"tok","ocpiBaseUrl":"https://x","publicBaseUrl":"https://y"}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d, expected 201 (body %s)", created.Code, created.Body)
	}

	duplicate := mockEnv.do(t, http.MethodPost, "/ocpi/api/partners", "", `{"slug":"second","name":"Second","partyId":"SEC","countryCode":"BR","tokenExpected":"tok","ocpiBaseUrl":"https://x","publicBaseUrl":"https://y"}`)
	if duplicate.Code != http.StatusConflict {
		t.Errorf("duplicate create status = %d, expected 409", duplicate.Code)
	}

	invalid := mockEnv.do(t, http.MethodPost, "/ocpi/api/partners", "", `{"slug":"third","name":"Third","partyId":"TOOLONG","countryCode":"BR","tokenExpected":"tok","ocpiBaseUrl":"https://x","publicBaseUrl":"https://y"}`)
	if invalid.Code != http.StatusBadRequest {
		t.Errorf("invalid create status = %d, expected 400", invalid.Code)
	}

	updated := mockEnv.do(t, http.MethodPut, "/ocpi/api/partners/second", "", `{"name":"Second Renamed","partyId":"SEC","countryCode":"BR","tokenExpected":"tok2","ocpiBaseUrl":"https://x","publicBaseUrl":"https://y"}`)
	if updated.Code != http.StatusOK {
		t.Fatalf("update status = %d, expected 200 (body %s)", updated.Code, updated.Body)
	}

	partner, err := mockEnv.partners.Get("second")
	if err != nil {
		t.Fatalf("Get() returned an error: %v", err)
	}
	if partner.Name != "Second Renamed" || partner.TokenExpected != "tok2" {
		t.Errorf("partner = %+v, expected the update to be applied", partner)
	}

	deleted := mockEnv.do(t, http.MethodDelete, "/ocpi/api/partners/second", "", "")
	if deleted.Code != http.StatusOK {
		t.Errorf("delete status = %d, expected 200", deleted.Code)
	}
	if _, err := mockEnv.partners.Get("second"); err == nil {
		t.Errorf("partner should be gone after delete")
	}
}

func TestPartnerProfilesSurviveRestart(t *testing.T) {
	dir := t.TempDir()
	inputPath := filepath.Join(dir, "partners.json")

	originalService := services.NewOCPIPartnerService(persistence.NewOCPIPartnerStore(inputPath))
	if _, err := originalService.Create(entities.OCPIPartner{
		Slug: "nayax-sim", Name: "Nayax", PartyID: "NYX", CountryCode: "BR",
		TokenExpected: "token-a", OCPIBaseURL: "https://x", PublicBaseURL: "https://y",
	}); err != nil {
		t.Fatalf("Create() returned an error: %v", err)
	}

	restartedService := services.NewOCPIPartnerService(persistence.NewOCPIPartnerStore(inputPath))
	if err := restartedService.Load(); err != nil {
		t.Fatalf("Load() returned an error: %v", err)
	}

	actualPartner, err := restartedService.Get("nayax-sim")
	if err != nil {
		t.Fatalf("Get() returned an error after restart: %v", err)
	}
	if actualPartner.TokenExpected != "token-a" {
		t.Errorf("TokenExpected = %q, expected %q", actualPartner.TokenExpected, "token-a")
	}
}

func TestEventsEndpointCursor(t *testing.T) {
	mockEnv := newOCPITestEnv(t, "", "")

	mockEnv.do(t, http.MethodPost, "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs", "Token token-a", `{"id":"cdr-1"}`)
	mockEnv.do(t, http.MethodPost, "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs", "Token token-a", `{"id":"cdr-2"}`)

	first := mockEnv.do(t, http.MethodGet, "/ocpi/api/partners/nayax-sim/events", "", "")
	var firstPage dto.OCPIEventsResponse
	if err := json.Unmarshal(first.Body.Bytes(), &firstPage); err != nil {
		t.Fatalf("could not decode events: %v", err)
	}
	if len(firstPage.Events) != 2 || firstPage.LastID != 2 {
		t.Fatalf("first page = %+v, expected 2 events ending at id 2", firstPage)
	}

	second := mockEnv.do(t, http.MethodGet, "/ocpi/api/partners/nayax-sim/events?after=2", "", "")
	var secondPage dto.OCPIEventsResponse
	if err := json.Unmarshal(second.Body.Bytes(), &secondPage); err != nil {
		t.Fatalf("could not decode events: %v", err)
	}
	if len(secondPage.Events) != 0 {
		t.Errorf("second page returned %d events, expected 0", len(secondPage.Events))
	}
}
