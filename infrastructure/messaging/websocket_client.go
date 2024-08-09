package messaging

import (
	"EV-Client-Simulator/core/entities"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct {
	Id              string
	conn            *websocket.Conn
	mu              *sync.Mutex
	expectedMessage string
}

func NewWebsocketClient(serverAddr string, clientId string) (*WebSocketClient, error) {
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: clientId}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		return nil, fmt.Errorf("client %s failed to connect: %v", clientId, err)
	}

	fmt.Println("Client %s Connected", clientId)

	return &WebSocketClient{
		Id:   clientId,
		conn: conn,
		mu:   &sync.Mutex{},
	}, nil
}

func NewWebsocketClientBatch(serverAddr string, numClients int, callsChannel chan entities.Message, resultsChannel chan entities.Message) {
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

				client, err := NewWebsocketClient(serverAddr, string(i))

				if err != nil {
					fmt.Printf("Failed to create client %d: %v", clientID, err)
					return
				}

				client.Listen(callsChannel, resultsChannel)
			}(i)
		}
	}

	wg.Wait()
}

func (client *WebSocketClient) GetId() string {
	return client.Id
}

func (client *WebSocketClient) GetConn() any {
	return client.conn
}

func (client *WebSocketClient) Listen(callsChannel chan entities.Message, resultsChannel chan entities.Message) {
	defer client.conn.Close()

	for {
		_, rawMessage, err := client.conn.ReadMessage()

		if err != nil {
			fmt.Printf("error reading message: %v", err)
			break
		}

		message, err := entities.New(rawMessage)

		if err != nil {
			fmt.Printf("Failed to parse message: %v", err)
			continue
		}

		if client.expectedMessage == message.ID {
			client.expectedMessage = ""
		}

		switch message.Type {
		case 2:
			callsChannel <- message
		case 3:
			resultsChannel <- message
		case 4:
			fmt.Printf("Server Error: %v", message)
		default:
			fmt.Printf("Unsupported message type: %d", message.Type)
		}
	}
}

func (client *WebSocketClient) Send(message entities.Message) {
	go func() {
		client.mu.Lock()
		defer client.mu.Unlock()
		timeElapsed := 0 * time.Second

		for client.expectedMessage != "" && timeElapsed < 5*time.Second {
			oneSecond := 1 * time.Second
			time.Sleep(oneSecond)
			timeElapsed += oneSecond
		}

		rawMessage, err := message.ConvertToRawMessage()

		if err != nil {
			fmt.Printf("failed to send message: %v", err)
		}

		if err := client.conn.WriteMessage(websocket.TextMessage, rawMessage); err != nil {
			fmt.Printf("failed to send message: %v", err)
		}

		fmt.Println("Sent Message: %v", message)
		client.expectedMessage = message.ID

	}()
}

func (client WebSocketClient) SendPeriodically(message entities.Message, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				client.Send(message)
			}
		}
	}()
}
