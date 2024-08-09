package main

import (
	"EV-Client-Simulator/core/entities"
	"EV-Client-Simulator/core/factories"
	"EV-Client-Simulator/infrastructure/messaging"
	"flag"
	"fmt"
	"time"
)

func main() {
	var serverAddr string
	// var numClients int

	// Setup command-line flags
	flag.StringVar(&serverAddr, "serverAddr", "localhost:8080", "WebSocket server address")
	// flag.IntVar(&numClients, "clients", 1, "Number of clients to simulate")
	flag.Parse()

	callsChannel := make(chan entities.Message)
	resultsChannel := make(chan entities.Message)

	client, err := messaging.NewWebsocketClient(serverAddr, "virtual")

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

	time.Sleep(5 * time.Second)
	if err != nil {
		fmt.Printf("Error creating client %v", err)
		panic(0)
	}

	done := make(chan bool)

	go func() {
		defer close(callsChannel)
		defer close(resultsChannel)
		client.SendPeriodically(factories.CreateHeartbeatMessage(nil), 30*time.Second)
		client.Send(factories.CreateBootNotificationMessage(nil))
		client.Send(factories.CreateStatusNotificationMessage(1, "AVAILABLE"))
		client.Send(factories.CreateStatusNotificationMessage(2, "AVAILABLE"))
		client.Listen(callsChannel, resultsChannel)
		done <- true
	}()

	<-done
}
