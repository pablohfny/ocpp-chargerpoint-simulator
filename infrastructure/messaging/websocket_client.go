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
	serverAddr      string
	isConnected     bool
	stopReconnect   chan struct{}
}

func NewWebsocketClient(serverAddr string, clientId string) (*WebSocketClient, error) {
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
				Id:            clientId,
				conn:          conn,
				mu:            &sync.Mutex{},
				serverAddr:    serverAddr,
				isConnected:   true,
				stopReconnect: make(chan struct{}),
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
	defer func() {
		client.isConnected = false
		client.reconnect(messagesChannel)
	}()

	for {
		select {
		case <-client.stopReconnect:
			return
		default:
			_, rawMessage, err := client.conn.ReadMessage()

			if err != nil {
				fmt.Printf("Client %s: error reading message: %v\n", client.Id, err)
				return // Trigger reconnect via defer
			}

			message, err := entities.New(rawMessage)

			if err != nil {
				fmt.Printf("Client %s: Failed to parse message: %v\n", client.Id, err)
				continue
			}

			if client.expectedMessage == message.ID {
				client.expectedMessage = ""
			}

			messagesChannel <- message
		}
	}
}

func (client *WebSocketClient) reconnect(messagesChannel chan entities.Message) {
	maxRetries := 10
	initialBackoff := 500 * time.Millisecond
	maxBackoff := 30 * time.Second
	backoff := initialBackoff
	retries := 0

	for retries < maxRetries {
		select {
		case <-client.stopReconnect:
			return
		default:
			fmt.Printf("Client %s: Attempting to reconnect (retry %d/%d)...\n", client.Id, retries+1, maxRetries)

			schemes := []string{"wss", "ws"}
			for _, scheme := range schemes {
				u := url.URL{Scheme: scheme, Host: client.serverAddr, Path: client.Id}
				fmt.Printf("Client %s: Reconnecting to %s\n", client.Id, u.String())

				dialer := &websocket.Dialer{
					HandshakeTimeout: 10 * time.Second,
				}

				conn, _, err := dialer.Dial(u.String(), nil)

				if err == nil {
					fmt.Printf("Client %s: Successfully reconnected using %s\n", client.Id, scheme)

					client.mu.Lock()
					client.conn = conn
					client.isConnected = true
					client.mu.Unlock()

					// Resume listening
					go client.Listen(messagesChannel)
					return
				}

				fmt.Printf("Client %s: Reconnection attempt failed for %s: %v\n", client.Id, u.String(), err)
			}

			retries++
			time.Sleep(backoff)

			// Exponential backoff with jitter
			backoff = time.Duration(float64(backoff) * 1.5)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	fmt.Printf("Client %s: Failed to reconnect after %d attempts\n", client.Id, maxRetries)
}

func (client *WebSocketClient) Send(message entities.Message, expectResult bool) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	if !client.isConnected {
		return fmt.Errorf("client %s is not connected", client.Id)
	}

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
		client.isConnected = false
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
	client.mu.Lock()
	defer client.mu.Unlock()

	close(client.stopReconnect)
	client.isConnected = false
	return client.conn.Close()
}
