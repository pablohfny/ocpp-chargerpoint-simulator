package main

import (
	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/config"
	infrastructure_http "EV-Client-Simulator/infrastructure/http"
	infrastructure_messaging "EV-Client-Simulator/infrastructure/messaging"
	interface_http "EV-Client-Simulator/interface/http"
	interface_messaging "EV-Client-Simulator/interface/messaging"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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

	// Register HTTP routes
	interface_http.RegisterRoutes(
		httpServer.Router(),
		stationController.GetService(),
		stationController.GetLogService(),
		cfg.ClientID,
		cfg.ServerAddr,
		stationController.IsConnected(),
		stationController.Reconnect,
		stationController.Disconnect,
	)

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
