package main

import (
	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/config"
	infrastructure_http "EV-Client-Simulator/infrastructure/http"
	infrastructure_messaging "EV-Client-Simulator/infrastructure/messaging"
	"EV-Client-Simulator/infrastructure/persistence"
	interface_http "EV-Client-Simulator/interface/http"
	"EV-Client-Simulator/interface/http/dto"
	"EV-Client-Simulator/interface/http/middleware"
	interface_messaging "EV-Client-Simulator/interface/messaging"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-chi/chi/v5"
)

func main() {
	// Load configuration
	cfg := config.Load()

	if cfg.ServerAddr == "" {
		fmt.Println("Error: serverAddr is required")
		os.Exit(1)
	}

	// Create WebSocket client
	client, err := infrastructure_messaging.NewWebsocketClient(cfg.ServerAddr, cfg.ClientID)
	if err != nil {
		fmt.Printf("Error creating client: %v\n", err)
		os.Exit(1)
	}

	// Create station and messaging controller
	station := entities.NewChargerStation(cfg.ClientID)
	stationController := interface_messaging.NewChargerStationMessagingController(&station, client)

	// Create HTTP server
	httpServer := infrastructure_http.NewServer(cfg.HTTPPort)

	basicAuth := middleware.BasicAuth("NuCharge Simulator", cfg.SimUser, cfg.SimPass)

	// Settings are env-seeded and then overlaid with whatever the Config tab
	// persisted, so the file wins over the environment on every boot.
	settingsService := services.NewAppSettingsService(
		cfg.BootstrapSettings(),
		persistence.NewAppSettingsStore(cfg.SettingsPath),
	)
	if err := settingsService.Load(); err != nil {
		fmt.Printf("Warning: could not load settings: %v\n", err)
	}

	ocpi := setupOCPI(cfg, httpServer.Router(), basicAuth, settingsService)

	pipelineService := services.NewPipelineService(
		stationController.GetService(),
		ocpi.partners,
		ocpi.events,
		ocpi.commands,
		settingsService,
	)

	// Register HTTP routes
	interface_http.RegisterRoutes(httpServer.Router(), interface_http.RouteDependencies{
		StationService: stationController.GetService(),
		LogService:     stationController.GetLogService(),
		ClientID:       cfg.ClientID,
		ServerAddr:     cfg.ServerAddr,
		Connected:      stationController.IsConnected(),
		ReconnectFunc:  stationController.Reconnect,
		DisconnectFunc: stationController.Disconnect,
		Settings:       settingsService,
		Pipeline:       pipelineService,
		Runtime: dto.RuntimeInfo{
			ServerAddr:   cfg.ServerAddr,
			ClientID:     cfg.ClientID,
			HTTPPort:     cfg.HTTPPort,
			AuthEnabled:  cfg.AuthEnabled(),
			PartnersPath: cfg.OCPIDataPath,
		},
		BasicAuth: basicAuth,
	})

	// Start HTTP server in background
	go func() {
		if err := httpServer.Start(); err != nil {
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	fmt.Printf("OCPP Simulator started\n")
	fmt.Printf("  Client ID: %s\n", cfg.ClientID)
	fmt.Printf("  Server: %s\n", cfg.ServerAddr)
	fmt.Printf("  HTTP API: http://localhost:%s/api/v1\n", cfg.HTTPPort)
	fmt.Printf("  Cockpit: http://localhost:%s/ (%d parceiro(s) OCPI)\n", cfg.HTTPPort, len(ocpi.partners.List()))
	fmt.Printf("  Painel legado: http://localhost:%s/legacy\n", cfg.HTTPPort)
	fmt.Printf("  OCPI data: %s\n", cfg.OCPIDataPath)
	fmt.Printf("  Settings: %s\n", cfg.SettingsPath)
	if cfg.AuthEnabled() {
		fmt.Printf("  Basic auth: enabled (user %s)\n", cfg.SimUser)
	} else {
		fmt.Printf("  Basic auth: disabled (set SIM_USER/SIM_PASS to enable)\n")
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		stationController.Disconnect()
		os.Exit(0)
	}()

	// Start the messaging controller (blocking)
	stationController.Init()
}

// ocpiServices are the OCPI building blocks the pipeline orchestrator reuses.
type ocpiServices struct {
	partners *services.OCPIPartnerService
	events   *services.OCPIEventService
	commands *services.OCPICommandService
}

// setupOCPI wires the OCPI partner simulator and registers its routes.
func setupOCPI(
	cfg *config.Config,
	router *chi.Mux,
	basicAuth func(http.Handler) http.Handler,
	settings *services.AppSettingsService,
) ocpiServices {
	partnerStore := persistence.NewOCPIPartnerStore(cfg.OCPIDataPath)
	eventLog := persistence.NewOCPIEventLog(filepath.Join(filepath.Dir(cfg.OCPIDataPath), "ocpi-events.jsonl"))

	partnerService := services.NewOCPIPartnerService(partnerStore)
	if err := partnerService.Load(); err != nil {
		fmt.Printf("Warning: could not load OCPI partners: %v\n", err)
	}

	eventService := services.NewOCPIEventService(eventLog)
	if err := eventService.Load(); err != nil {
		fmt.Printf("Warning: could not load OCPI events: %v\n", err)
	}

	commandService := services.NewOCPICommandService(partnerService, eventService, infrastructure_http.NewOCPIClient())

	interface_http.RegisterOCPIRoutes(router, interface_http.OCPIDependencies{
		Partners:  partnerService,
		Events:    eventService,
		Commands:  commandService,
		Settings:  settings,
		BasicAuth: basicAuth,
	})

	return ocpiServices{partners: partnerService, events: eventService, commands: commandService}
}
