package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"EV-Client-Simulator/pkg/websocket"
)

func main() {
	var serverAddr string
	var numClients int
	var batchSize int = 100 // Fixed batch size

	// Setup command-line flags
	flag.StringVar(&serverAddr, "serverAddr", "localhost:8080", "WebSocket server address")
	flag.IntVar(&numClients, "clients", 1, "Number of clients to simulate")
	flag.Parse() // Don't forget to parse the command line arguments

	for i := 0; i < numClients; i += batchSize {
		end := i + batchSize
		if end > numClients {
			end = numClients
		}

		var wg sync.WaitGroup
		// Adjust the number of goroutines to wait for
		wg.Add(end - i)

		for j := i; j < end; j++ {
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
			}(j)
		}

		// Wait for the current batch to complete before proceeding
		wg.Wait()
		fmt.Printf("Batch %d to %d completed\n", i, end-1)

		// Optionally, include a slight pause before starting the next batch
		// to reduce the load spike on the server or to simulate staggered connections
		time.Sleep(1 * time.Second)
	}

	fmt.Println("All clients have finished their operations.")
}
