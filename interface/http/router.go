package http

import (
	"net/http"
	"os"
	"path/filepath"

	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/interface/http/dto"
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

	// Settings owns every runtime-editable value shown in the Config tab.
	Settings *services.AppSettingsService
	// Pipeline is the end-to-end cockpit orchestrator. Nil skips its routes.
	Pipeline *services.PipelineService
	// Runtime holds the boot-fixed values the Config tab shows read-only.
	Runtime dto.RuntimeInfo

	// BasicAuth wraps the web UI and the control APIs. Nil disables it.
	BasicAuth func(http.Handler) http.Handler
}

// RegisterRoutes registers all API routes
func RegisterRoutes(router *chi.Mux, deps RouteDependencies) {
	// Initialize handlers
	statusHandler := handlers.NewStatusHandler(deps.StationService, deps.Settings, deps.ClientID, deps.ServerAddr, deps.Connected)
	actionHandler := handlers.NewActionHandler(deps.StationService, deps.ReconnectFunc, deps.DisconnectFunc)
	configHandler := handlers.NewConfigHandler(deps.StationService.GetConfigService())
	simulationHandler := handlers.NewSimulationHandler(deps.StationService.GetSimulationService())
	logsHandler := handlers.NewLogsHandler(deps.LogService)
	ocppTriggerHandler := handlers.NewOCPPTriggerHandler(deps.StationService)
	settingsHandler := handlers.NewSettingsHandler(deps.Settings, deps.Runtime)

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

		// Simulator settings, editable from the Config tab
		r.Route("/settings", func(r chi.Router) {
			r.Get("/", settingsHandler.GetSettings)
			r.Put("/", settingsHandler.UpdateSettings)
		})

		// End-to-end pipeline cockpit
		if deps.Pipeline != nil {
			pipelineHandler := handlers.NewPipelineHandler(deps.Pipeline)
			r.Route("/pipeline", func(r chi.Router) {
				r.Get("/state", pipelineHandler.GetState)
				r.Post("/start-context", pipelineHandler.StartContext)
				r.Post("/action", pipelineHandler.RunAction)
			})
		}

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

	// The unified cockpit lives at /, served by the file server below. The old
	// standalone pages redirect onto their tab so bookmarks keep working, and
	// the full legacy panel stays reachable during the transition.
	router.Method(http.MethodGet, "/legacy", protect(revalidate(servePage(staticPath, "legacy.html"))))
	router.Method(http.MethodGet, "/simple", protect(redirectTo("/#carregador")))
	router.Method(http.MethodGet, "/ocpi", protect(redirectTo("/#parceiros")))

	router.Handle("/*", protect(revalidate(http.FileServer(http.Dir(staticPath)))))
}

// revalidate forbids the browser from reusing a cached asset without asking.
// The file server only sends Last-Modified, and with no Cache-Control a browser
// is free to guess a freshness window from the file's age — which after a
// redeploy means a fresh index.html running yesterday's app.js. "no-cache"
// still allows a cheap 304, it just removes the guessing.
func revalidate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		next.ServeHTTP(w, r)
	})
}

// servePage serves a single HTML file from the static directory.
func servePage(staticPath, file string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(staticPath, file))
	})
}

// redirectTo permanently points an old page at its tab in the cockpit.
func redirectTo(target string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target, http.StatusFound)
	})
}

// findStaticPath finds the web/static directory
func findStaticPath() string {
	// Try common locations
	paths := []string{
		"web/static",
		"./web/static",
		"../web/static",
		// Reached from a package directory two levels down, as the tests are.
		"../../web/static",
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
