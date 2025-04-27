package main

import (
	"EV-Client-Simulator/app/domain/entities"
	infrastructure_messaging "EV-Client-Simulator/infrastructure/messaging"
	interface_messaging "EV-Client-Simulator/interface/messaging"
	"flag"
	"fmt"
	"os"
)

func main() {
	var serverAddr string
	var clientId string

	flag.StringVar(&serverAddr, "serverAddr", os.Getenv("SERVER_ADDR"), "WebSocket server address")
	flag.StringVar(&clientId, "clientId", os.Getenv("CLIENT_ID"), "Client ID")
	flag.Parse()

	client, err := infrastructure_messaging.NewWebsocketClient(serverAddr, clientId)

	if err != nil {
		fmt.Printf("Error creating client %v", err)
		panic(0)
	}

	station := entities.NewChargerStation("virtual")
	stationController := interface_messaging.NewChargerStationMessagingController(&station, client)
	stationController.Init()
}
