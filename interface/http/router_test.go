package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
	basicAuth := middleware.BasicAuth("test", simUser, simPass)

	router := chi.NewRouter()
	RegisterRoutes(router, RouteDependencies{
		StationService:     controller.GetService(),
		LogService:         controller.GetLogService(),
		ClientID:           "test-station",
		ServerAddr:         "localhost:3001",
		Connected:          controller.IsConnected(),
		ReconnectFunc:      controller.Reconnect,
		DisconnectFunc:     controller.Disconnect,
		BatteryCapacityKWh: 60,
		BasicAuth:          basicAuth,
	})
	RegisterOCPIRoutes(router, OCPIDependencies{
		Partners:  partnerService,
		Events:    eventService,
		Commands:  services.NewOCPICommandService(partnerService, eventService, &mockCommandClient{}),
		Defaults:  dto.OCPIDefaultsResponse{},
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
