package main

import (
	"flag"
	"log"
	"sync"
	"time"

	"EV-Client-Simulator/pkg/websocket"
)

func main() {
	var serverAddr string
	var numClients int

	// Setup command-line flags
	flag.StringVar(&serverAddr, "serverAddr", "localhost:8080", "WebSocket server address")
	flag.IntVar(&numClients, "clients", 1, "Number of clients to simulate")
	flag.Parse() // Parse command line arguments

	const batchSize = 100
	var wg sync.WaitGroup
	wg.Add(numClients)
	for batchStart := 0; batchStart < numClients; batchStart += batchSize {
		batchEnd := batchStart + batchSize

		if batchEnd > numClients {
			batchEnd = numClients
		}
		for i := batchStart; i < batchEnd; i++ {
			go func(clientID int) {
				defer wg.Done()

				// Each goroutine creates a new client
				client, err := websocket.NewClient(serverAddr, clientID)
				if err != nil {
					log.Printf("Failed to create client %d: %v", clientID, err)
					return
				}

				// Send a boot notification for each client
				if err := client.SendBootNotification(); err != nil {
					log.Printf("Failed to send boot notification for client %d: %v", clientID, err)
					return
				}

				// Start handling messages
				client.HandleMessages()
			}(i)
		}
		time.Sleep(1 * time.Second)
	}

	wg.Wait()
}
