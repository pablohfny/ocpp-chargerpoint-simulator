package main

import (
	"EV-Client-Simulator/core/entities"
	"EV-Client-Simulator/core/factories"
	"EV-Client-Simulator/infrastructure/messaging"
	"flag"
	"fmt"
)

func main() {
	var serverAddr string
	var numClients int

	// Setup command-line flags
	flag.StringVar(&serverAddr, "serverAddr", "localhost:8080", "WebSocket server address")
	flag.IntVar(&numClients, "clients", 1, "Number of clients to simulate")
	flag.Parse()

	callsChannel := make(chan entities.Message)
	resultsChannel := make(chan entities.Message)

	client, err := messaging.NewWebsocketClient(serverAddr, "teste")

	if err != nil {
		fmt.Printf("Error creating client %v", err)
		panic(0)
	}

	go client.Listen(callsChannel, resultsChannel)

	client.Send(factories.NewCall("BootNotification", nil))

	go func() {
		for msg := range callsChannel {
			fmt.Printf("Received call: %v\n", msg)
		}
	}()

	go func() {
		for msg := range resultsChannel {
			fmt.Printf("Received result: %v\n", msg)
		}
	}()

	select {}
}
