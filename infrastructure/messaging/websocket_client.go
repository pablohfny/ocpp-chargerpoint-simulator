package messaging

import (
	"EV-Client-Simulator/app/domain/entities"
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
	// Try both secure and non-secure connections
	schemes := []string{"wss", "ws"}
	var lastErr error

	for _, scheme := range schemes {
		u := url.URL{Scheme: scheme, Host: serverAddr, Path: clientId}
		fmt.Printf("Attempting to connect to %s\n", u.String())

		// Create a dialer with timeout
		dialer := &websocket.Dialer{
			Proxy:            websocket.DefaultDialer.Proxy,
			HandshakeTimeout: 10 * time.Second,
		}

		conn, resp, err := dialer.Dial(u.String(), nil)

		if err == nil {
			fmt.Printf("Client %s Connected using %s\n", clientId, scheme)
			return &WebSocketClient{
				Id:   clientId,
				conn: conn,
				mu:   &sync.Mutex{},
			}, nil
		}

		lastErr = err
		var statusCode int
		if resp != nil {
			statusCode = resp.StatusCode
			fmt.Printf("Handshake failed with status: %d for %s\n", statusCode, u.String())
		}
		fmt.Printf("Connection attempt failed for %s: %v\n", u.String(), err)
	}

	return nil, fmt.Errorf("client %s failed to connect: %v", clientId, lastErr)
}

func NewWebsocketClientBatch(serverAddr string, numClients int, messagesChannel chan entities.Message) {
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

				client, err := NewWebsocketClient(serverAddr, fmt.Sprintf("%d", clientID))

				if err != nil {
					fmt.Printf("Failed to create client %d: %v\n", clientID, err)
					return
				}

				client.Listen(messagesChannel)
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

func (client *WebSocketClient) Listen(messagesChannel chan entities.Message) {
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

		messagesChannel <- message
	}
}

func (client *WebSocketClient) Send(message entities.Message, expectResult bool) error {
	client.mu.Lock()
	defer client.mu.Unlock()
	timeElapsed := 0 * time.Millisecond

	for client.expectedMessage != "" && timeElapsed < 900*time.Millisecond {
		sleepDuration := 300 * time.Millisecond
		time.Sleep(sleepDuration)
		timeElapsed += sleepDuration
	}

	rawMessage, err := message.ConvertToRawMessage()

	if err != nil {
		return err
	}

	if err := client.conn.WriteMessage(websocket.TextMessage, rawMessage); err != nil {
		return err
	}

	fmt.Printf("Sent Message: %v\n", message)

	if expectResult {
		client.expectedMessage = message.ID
	}

	return nil
}

func (client *WebSocketClient) SendPeriodically(message entities.Message, expectResult bool, interval time.Duration) error {
	var err error

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			err := client.Send(message, expectResult)
			if err != nil {
				return
			}
		}
	}()

	return err
}

func (client *WebSocketClient) Disconnect() error {
	return client.conn.Close()
}
