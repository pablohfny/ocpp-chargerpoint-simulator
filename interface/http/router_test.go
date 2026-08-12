package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/infrastructure/persistence"
	"EV-Client-Simulator/interface/http/dto"
	"EV-Client-Simulator/interface/http/middleware"
	interface_messaging "EV-Client-Simulator/interface/messaging"

	"github.com/go-chi/chi/v5"
)

// mockMessagingClient satisfies abstracts.MessagingClient without a socket.
type mockMessagingClient struct{}

func (c *mockMessagingClient) GetId() string                                    { return "test-station" }
func (c *mockMessagingClient) GetConn() any                                     { return nil }
func (c *mockMessagingClient) Listen(messages chan entities.Message)            {}
func (c *mockMessagingClient) Send(message entities.Message, expect bool) error { return nil }
func (c *mockMessagingClient) SendPeriodically(message entities.Message, expect bool, interval time.Duration) error {
	return nil
}
func (c *mockMessagingClient) Disconnect() error { return nil }
func (c *mockMessagingClient) Reconnect() error  { return nil }

// newFullRouter wires the OCPP routes and the OCPI routes onto one mux, exactly
// as main() does, so route conflicts surface in tests rather than at boot.
func newFullRouter(t *testing.T, simUser, simPass string) *chi.Mux {
	t.Helper()

	station := entities.NewChargerStation("test-station")
	controller := interface_messaging.NewChargerStationMessagingController(&station, &mockMessagingClient{})

	dir := t.TempDir()
	partnerService := services.NewOCPIPartnerService(persistence.NewOCPIPartnerStore(filepath.Join(dir, "partners.json")))
	eventService := services.NewOCPIEventService(persistence.NewOCPIEventLog(filepath.Join(dir, "events.jsonl")))
	commandService := services.NewOCPICommandService(partnerService, eventService, &mockCommandClient{})
	settingsService := services.NewAppSettingsService(entities.AppSettings{
		OCPIBaseURL:        "https://ocpi-dev.nucharge.com.br",
		PublicBaseURL:      "https://sim-dev.nucharge.com.br",
		BatteryCapacityKWh: 60,
	}, persistence.NewAppSettingsStore(filepath.Join(dir, "settings.json")))
	basicAuth := middleware.BasicAuth("test", simUser, simPass)

	router := chi.NewRouter()
	RegisterRoutes(router, RouteDependencies{
		StationService: controller.GetService(),
		LogService:     controller.GetLogService(),
		ClientID:       "test-station",
		ServerAddr:     "localhost:3001",
		Connected:      controller.IsConnected(),
		ReconnectFunc:  controller.Reconnect,
		DisconnectFunc: controller.Disconnect,
		Settings:       settingsService,
		Pipeline: services.NewPipelineService(
			controller.GetService(), partnerService, eventService, commandService, settingsService,
		),
		Runtime:   dto.RuntimeInfo{ServerAddr: "localhost:3001", ClientID: "test-station", HTTPPort: "8080"},
		BasicAuth: basicAuth,
	})
	RegisterOCPIRoutes(router, OCPIDependencies{
		Partners:  partnerService,
		Events:    eventService,
		Commands:  commandService,
		Settings:  settingsService,
		BasicAuth: basicAuth,
	})

	if _, err := partnerService.Create(entities.OCPIPartner{
		Slug: "nayax-sim", Name: "Nayax", PartyID: "NYX", CountryCode: "BR",
		TokenExpected: "token-a", OCPIBaseURL: "https://x", PublicBaseURL: "https://y",
	}); err != nil {
		t.Fatalf("could not seed partner: %v", err)
	}

	return router
}

// TestFullRouterAuthBoundaries locks in which routes basic auth protects and
// which stay open, across the OCPP and OCPI route trees on a single mux.
func TestFullRouterAuthBoundaries(t *testing.T) {
	tests := []struct {
		name           string
		inputMethod    string
		inputPath      string
		inputBasicAuth bool
		inputOCPIToken string
		expectedStatus int
	}{
		{name: "health probe is open", inputMethod: http.MethodGet, inputPath: "/health", expectedStatus: http.StatusOK},
		{name: "ocpp status needs auth", inputMethod: http.MethodGet, inputPath: "/api/v1/status", expectedStatus: http.StatusUnauthorized},
		{name: "ocpp status with auth", inputMethod: http.MethodGet, inputPath: "/api/v1/status", inputBasicAuth: true, expectedStatus: http.StatusOK},
		{name: "connector status with auth", inputMethod: http.MethodGet, inputPath: "/api/v1/status/connectors/1", inputBasicAuth: true, expectedStatus: http.StatusOK},
		{name: "ocpi control api needs auth", inputMethod: http.MethodGet, inputPath: "/ocpi/api/partners", expectedStatus: http.StatusUnauthorized},
		{name: "ocpi control api with auth", inputMethod: http.MethodGet, inputPath: "/ocpi/api/partners", inputBasicAuth: true, expectedStatus: http.StatusOK},
		{
			name:           "receiver route is exempt from basic auth",
			inputMethod:    http.MethodPost,
			inputPath:      "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs",
			inputOCPIToken: "Token token-a",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "receiver route still enforces its own token",
			inputMethod:    http.MethodPost,
			inputPath:      "/ocpi/p/nayax-sim/receiver/2.2.1/cdrs",
			inputOCPIToken: "Token wrong",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockRouter := newFullRouter(t, "admin", "secret")

			req := httptest.NewRequest(test.inputMethod, test.inputPath, nil)
			if test.inputBasicAuth {
				req.SetBasicAuth("admin", "secret")
			}
			if test.inputOCPIToken != "" {
				req.Header.Set("Authorization", test.inputOCPIToken)
			}

			recorder := httptest.NewRecorder()
			mockRouter.ServeHTTP(recorder, req)

			if recorder.Code != test.expectedStatus {
				t.Errorf("status = %d, expected %d (body %s)", recorder.Code, test.expectedStatus, recorder.Body)
			}
		})
	}
}

// TestCockpitPageRouting locks in where the browser lands: the cockpit at /,
// the old pages redirected onto their tab and the legacy panel still served.
func TestCockpitPageRouting(t *testing.T) {
	tests := []struct {
		name             string
		inputPath        string
		expectedStatus   int
		expectedLocation string
	}{
		{name: "cockpit at the root", inputPath: "/", expectedStatus: http.StatusOK},
		{name: "shared stylesheet", inputPath: "/app.css", expectedStatus: http.StatusOK},
		{name: "pipeline script", inputPath: "/pipeline.js", expectedStatus: http.StatusOK},
		{name: "legacy panel", inputPath: "/legacy", expectedStatus: http.StatusOK},
		{
			name:             "simple redirects to the charger tab",
			inputPath:        "/simple",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/#carregador",
		},
		{
			name:             "ocpi redirects to the partners tab",
			inputPath:        "/ocpi",
			expectedStatus:   http.StatusFound,
			expectedLocation: "/#parceiros",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockRouter := newFullRouter(t, "", "")

			req := httptest.NewRequest(http.MethodGet, test.inputPath, nil)
			recorder := httptest.NewRecorder()
			mockRouter.ServeHTTP(recorder, req)

			if recorder.Code != test.expectedStatus {
				t.Fatalf("status = %d, expected %d", recorder.Code, test.expectedStatus)
			}
			if test.expectedLocation != "" && recorder.Header().Get("Location") != test.expectedLocation {
				t.Errorf("Location = %q, expected %q", recorder.Header().Get("Location"), test.expectedLocation)
			}
		})
	}
}

// TestCockpitAssetsMustBeRevalidated guards the trap that made a deploy look
// like a no-op: a browser holding yesterday's app.js next to today's HTML.
func TestCockpitAssetsMustBeRevalidated(t *testing.T) {
	paths := []string{"/", "/app.js", "/app.css", "/legacy"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			mockRouter := newFullRouter(t, "", "")

			req := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			mockRouter.ServeHTTP(recorder, req)

			actualCacheControl := recorder.Header().Get("Cache-Control")
			if actualCacheControl != "no-cache" {
				t.Errorf("Cache-Control = %q, expected %q", actualCacheControl, "no-cache")
			}
		})
	}
}

// TestPipelineRoutesAreServed proves the cockpit's own API is wired and behind
// the same basic auth as the rest of the control plane.
func TestPipelineRoutesAreServed(t *testing.T) {
	mockRouter := newFullRouter(t, "", "")

	stateRequest := httptest.NewRequest(http.MethodGet, "/api/v1/pipeline/state", nil)
	stateRecorder := httptest.NewRecorder()
	mockRouter.ServeHTTP(stateRecorder, stateRequest)

	if stateRecorder.Code != http.StatusOK {
		t.Fatalf("state status = %d, expected 200 (body %s)", stateRecorder.Code, stateRecorder.Body)
	}

	var actualState struct {
		Stage string `json:"stage"`
	}
	if err := json.Unmarshal(stateRecorder.Body.Bytes(), &actualState); err != nil {
		t.Fatalf("could not decode state: %v", err)
	}
	if actualState.Stage != "idle" {
		t.Errorf("stage = %q, expected %q before a context is armed", actualState.Stage, "idle")
	}

	actionRequest := httptest.NewRequest(http.MethodPost, "/api/v1/pipeline/action", strings.NewReader(`{"action":"plug"}`))
	actionRecorder := httptest.NewRecorder()
	mockRouter.ServeHTTP(actionRecorder, actionRequest)

	if actionRecorder.Code != http.StatusConflict {
		t.Errorf("action status = %d, expected 409 without a context", actionRecorder.Code)
	}
}

// TestSettingsRoundTrip covers the Config tab's read and write path.
func TestSettingsRoundTrip(t *testing.T) {
	mockRouter := newFullRouter(t, "", "")

	updateRequest := httptest.NewRequest(http.MethodPut, "/api/v1/settings", strings.NewReader(
		`{"ocpiBaseUrl":"https://ocpi-vps.nucharge.com.br","publicBaseUrl":"https://sim.nucharge.com.br","defaultLocationId":"LOC-9","batteryCapacityKwh":82}`))
	updateRecorder := httptest.NewRecorder()
	mockRouter.ServeHTTP(updateRecorder, updateRequest)

	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update status = %d, expected 200 (body %s)", updateRecorder.Code, updateRecorder.Body)
	}

	readRequest := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	readRecorder := httptest.NewRecorder()
	mockRouter.ServeHTTP(readRecorder, readRequest)

	var actualPayload dto.SettingsResponse
	if err := json.Unmarshal(readRecorder.Body.Bytes(), &actualPayload); err != nil {
		t.Fatalf("could not decode settings: %v", err)
	}
	if actualPayload.Settings.DefaultLocationID != "LOC-9" || actualPayload.Settings.BatteryCapacityKWh != 82 {
		t.Errorf("settings = %+v, expected the update to stick", actualPayload.Settings)
	}
	if actualPayload.Runtime.ClientID != "test-station" {
		t.Errorf("runtime = %+v, expected the boot-fixed values to be reported", actualPayload.Runtime)
	}

	// The battery capacity feeds the connector view, so the change must show up
	// there too rather than being frozen at boot.
	connectorRequest := httptest.NewRequest(http.MethodGet, "/api/v1/status/connectors/1", nil)
	connectorRecorder := httptest.NewRecorder()
	mockRouter.ServeHTTP(connectorRecorder, connectorRequest)

	var actualConnector dto.ConnectorResponse
	if err := json.Unmarshal(connectorRecorder.Body.Bytes(), &actualConnector); err != nil {
		t.Fatalf("could not decode connector: %v", err)
	}
	if actualConnector.BatteryCapacityKWh != 82 {
		t.Errorf("BatteryCapacityKWh = %v, expected the saved 82", actualConnector.BatteryCapacityKWh)
	}
}

func TestConnectorStatusReportsBattery(t *testing.T) {
	mockRouter := newFullRouter(t, "", "")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status/connectors/1", nil)
	recorder := httptest.NewRecorder()
	mockRouter.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, expected 200 (body %s)", recorder.Code, recorder.Body)
	}

	var actualConnector dto.ConnectorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &actualConnector); err != nil {
		t.Fatalf("could not decode connector: %v", err)
	}
	if actualConnector.BatteryCapacityKWh != 60 {
		t.Errorf("BatteryCapacityKWh = %v, expected 60", actualConnector.BatteryCapacityKWh)
	}
	if actualConnector.BatteryPercent != 0 {
		t.Errorf("BatteryPercent = %d, expected 0 for a fresh connector", actualConnector.BatteryPercent)
	}
}
