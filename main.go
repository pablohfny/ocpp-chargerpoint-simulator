package main

import (
	"EV-Client-Simulator/domain/entities"
	infrastructure_messaging "EV-Client-Simulator/infrastructure/messaging"
	interface_messaging "EV-Client-Simulator/interface/messaging"
	"flag"
	"fmt"
)

func main() {
	var serverAddr string

	flag.StringVar(&serverAddr, "serverAddr", "localhost:8080", "WebSocket server address")
	flag.Parse()

	client, err := infrastructure_messaging.NewWebsocketClient(serverAddr, "virtual")

	if err != nil {
		fmt.Printf("Error creating client %v", err)
		panic(0)
	}

	station := entities.NewChargerStation("virtual")
	stationController := interface_messaging.NewChargerStationMessagingController(&station, client)
	stationController.Init()
}
