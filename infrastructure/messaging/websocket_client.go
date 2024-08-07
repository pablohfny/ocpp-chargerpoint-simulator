package messaging

import (
	"EV-Client-Simulator/core/entities"
	"fmt"
	"log"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketClient struct {
	Id   string
	conn *websocket.Conn
}

func NewWebsocketClient(serverAddr string, clientId string) (*WebSocketClient, error) {
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: clientId}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		return nil, fmt.Errorf("client %s failed to connect: %v", clientId, err)
	}

	log.Printf("Client %s Connected", clientId)

	return &WebSocketClient{
		Id:   clientId,
		conn: conn,
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
					log.Printf("Failed to create client %d: %v", clientID, err)
					return
				}

				client.Listen(callsChannel, resultsChannel)
			}(i)
		}
	}

	wg.Wait()
}

func (client *WebSocketClient) Listen(callsChannel chan entities.Message, resultsChannel chan entities.Message) {
	defer client.conn.Close()

	for {
		_, rawMessage, err := client.conn.ReadMessage()

		if err != nil {
			log.Printf("error reading message: %v", err)
			break
		}

		message, err := entities.New(rawMessage)

		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		switch message.Type {
		case 2:
			callsChannel <- message
		case 3:
			resultsChannel <- message
		case 4:
			log.Printf("Server Error: %v", message)
		default:
			log.Printf("Unsupported message type: %d", message.Type)
		}
	}
}

func (client *WebSocketClient) Send(message entities.Message) error {
	rawMessage, err := message.ConvertToRawMessage()

	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	if err := client.conn.WriteMessage(websocket.TextMessage, rawMessage); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	return nil
}
