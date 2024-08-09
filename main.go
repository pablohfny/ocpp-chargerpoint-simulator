package main

import (
	"EV-Client-Simulator/core/aggregates"
	"EV-Client-Simulator/infrastructure/messaging"
	"flag"
	"fmt"
)

func main() {
	var serverAddr string
	// var numClients int

	// Setup command-line flags
	flag.StringVar(&serverAddr, "serverAddr", "localhost:8080", "WebSocket server address")
	// flag.IntVar(&numClients, "clients", 1, "Number of clients to simulate")
	flag.Parse()

	client, err := messaging.NewWebsocketClient(serverAddr, "virtual")

	if err != nil {
		fmt.Printf("Error creating client %v", err)
		panic(0)
	}

	stationClient := aggregates.NewChargerStationClient(client)
	stationClient.Init()
}
