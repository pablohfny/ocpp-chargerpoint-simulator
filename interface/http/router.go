package http

import (
	"net/http"
	"os"
	"path/filepath"

	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/handlers"

	"github.com/go-chi/chi/v5"
)

// RouteDependencies bundles everything the HTTP routes need.
type RouteDependencies struct {
	StationService *services.ChargerStationService
	LogService     *services.MessageLogService
	ClientID       string
	ServerAddr     string
	Connected      *bool
	ReconnectFunc  func() error
	DisconnectFunc func() error

	// BatteryCapacityKWh sizes the virtual EV battery behind the reported
	// battery percentage.
	BatteryCapacityKWh float64

	// BasicAuth wraps the web UI and the control APIs. Nil disables it.
	BasicAuth func(http.Handler) http.Handler
}

// RegisterRoutes registers all API routes
func RegisterRoutes(router *chi.Mux, deps RouteDependencies) {
	// Initialize handlers
	statusHandler := handlers.NewStatusHandler(deps.StationService, deps.ClientID, deps.ServerAddr, deps.Connected, deps.BatteryCapacityKWh)
	actionHandler := handlers.NewActionHandler(deps.StationService, deps.ReconnectFunc, deps.DisconnectFunc)
	configHandler := handlers.NewConfigHandler(deps.StationService.GetConfigService())
	simulationHandler := handlers.NewSimulationHandler(deps.StationService.GetSimulationService())
	logsHandler := handlers.NewLogsHandler(deps.LogService)
	ocppTriggerHandler := handlers.NewOCPPTriggerHandler(deps.StationService)

	// API v1 routes
	router.Route("/api/v1", func(r chi.Router) {
		if deps.BasicAuth != nil {
			r.Use(deps.BasicAuth)
		}

		// Health check
		r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(`{"status":"ok"}`))
		})

		// Status endpoints
		r.Route("/status", func(r chi.Router) {
			r.Get("/", statusHandler.GetStatus)
			r.Get("/station", statusHandler.GetStation)
			r.Get("/connectors", statusHandler.GetConnectors)
			r.Get("/connectors/{id}", statusHandler.GetConnector)
			r.Get("/transactions", statusHandler.GetTransactions)
			r.Get("/connection", statusHandler.GetConnection)
		})

		// Action endpoints
		r.Route("/actions", func(r chi.Router) {
			// Connector actions
			r.Route("/connectors/{id}", func(r chi.Router) {
				r.Post("/plug", actionHandler.PlugCable)
				r.Post("/unplug", actionHandler.UnplugCable)
				r.Post("/preparing", actionHandler.SetPreparing)
				r.Post("/start", actionHandler.StartCharging)
				r.Post("/stop", actionHandler.StopCharging)
				r.Post("/authorize", actionHandler.Authorize)
				r.Post("/fault", actionHandler.SetFault)
				r.Post("/clear-fault", actionHandler.ClearFault)
				r.Post("/unavailable", actionHandler.SetUnavailable)
				r.Post("/available", actionHandler.SetAvailable)
				r.Post("/reserve", actionHandler.SetReservation)
				r.Post("/cancel-reservation", actionHandler.CancelReservation)
				r.Post("/suspend-ev", actionHandler.SuspendEV)
				r.Post("/suspend-evse", actionHandler.SuspendEVSE)
				r.Post("/resume", actionHandler.ResumeCharging)
				// Manual mode actions
				r.Post("/set-status", actionHandler.ManualSetStatus)
				r.Post("/send-meter", actionHandler.ManualSendNextMeter)
				r.Get("/meter-queue", actionHandler.ManualGetMeterQueue)
				r.Delete("/meter-queue", actionHandler.ManualFlushMeterQueue)
				r.Post("/stop-transaction", actionHandler.ManualSendStopTransaction)
			})

			// Station actions
			r.Route("/station", func(r chi.Router) {
				r.Post("/boot", actionHandler.SendBootNotification)
				r.Post("/heartbeat", actionHandler.SendHeartbeat)
				r.Post("/reset", actionHandler.Reset)
				r.Post("/reconnect", actionHandler.Reconnect)
				r.Post("/disconnect", actionHandler.Disconnect)
			})
		})

		// Configuration endpoints
		r.Route("/config", func(r chi.Router) {
			// OCPP configuration
			r.Route("/ocpp", func(r chi.Router) {
				r.Get("/", configHandler.GetAllConfig)
				r.Get("/{key}", configHandler.GetConfig)
				r.Put("/{key}", configHandler.SetConfig)
			})

			// Simulation configuration
			r.Route("/simulation", func(r chi.Router) {
				r.Get("/", simulationHandler.GetSimulationSettings)
				r.Put("/", simulationHandler.SetSimulationSettings)
				r.Get("/delays", simulationHandler.GetDelays)
				r.Put("/delays", simulationHandler.SetDelays)
				r.Get("/failures", simulationHandler.GetFailureRates)
				r.Put("/failures", simulationHandler.SetFailureRates)
				r.Get("/errors", simulationHandler.GetErrorInjection)
				r.Put("/errors", simulationHandler.SetErrorInjection)
				r.Get("/transaction", simulationHandler.GetTransactionSettings)
				r.Put("/transaction", simulationHandler.SetTransactionSettings)
				r.Post("/reset", simulationHandler.ResetSimulation)
				r.Get("/manual-mode", simulationHandler.GetManualMode)
				r.Put("/manual-mode", simulationHandler.SetManualMode)
			})
		})

		// Log endpoints
		r.Route("/logs", func(r chi.Router) {
			r.Get("/messages", logsHandler.GetLogs)
			r.Delete("/messages", logsHandler.ClearLogs)
		})

		// OCPP trigger endpoints
		r.Route("/ocpp/trigger", func(r chi.Router) {
			r.Post("/boot-notification", ocppTriggerHandler.TriggerBootNotification)
			r.Post("/status-notification", ocppTriggerHandler.TriggerStatusNotification)
			r.Post("/meter-values", ocppTriggerHandler.TriggerMeterValues)
			r.Post("/heartbeat", ocppTriggerHandler.TriggerHeartbeat)
			r.Post("/authorize", ocppTriggerHandler.TriggerAuthorize)
			r.Post("/start-transaction", ocppTriggerHandler.TriggerStartTransaction)
		})
	})

	// Unauthenticated health check for container/reverse-proxy probes
	router.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Serve static files for web UI
	staticPath := findStaticPath()
	if staticPath == "" {
		return
	}

	protect := deps.BasicAuth
	if protect == nil {
		protect = func(next http.Handler) http.Handler { return next }
	}

	// Named pages, so the URLs stay clean and extension free
	router.Method(http.MethodGet, "/simple", protect(servePage(staticPath, "simple.html")))
	router.Method(http.MethodGet, "/ocpi", protect(servePage(staticPath, "ocpi.html")))

	router.Handle("/*", protect(http.FileServer(http.Dir(staticPath))))
}

// servePage serves a single HTML file from the static directory.
func servePage(staticPath, file string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticPath, file))
	})
}

// findStaticPath finds the web/static directory
func findStaticPath() string {
	// Try common locations
	paths := []string{
		"web/static",
		"./web/static",
		"../web/static",
	}

	// Also try relative to executable
	if execPath, err := os.Executable(); err == nil {
		execDir := filepath.Dir(execPath)
		paths = append(paths, filepath.Join(execDir, "web/static"))
	}

	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			return p
		}
	}

	return ""
}
