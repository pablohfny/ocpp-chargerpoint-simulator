package websocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

type ChargingPoint struct {
	ID        int    `json:"id"`
	Status    string `json:"status"`
	ErrorCode string `json:"errorCode,omitempty"`
}

type Charger struct {
	ID            string           `json:"id"`
	ChargerPoints []*ChargingPoint `json:"chargerPoints"`
	Status        string           `json:"status"`
}

type WebSocketClient struct {
	conn    *websocket.Conn
	Charger *Charger
	mu      sync.Mutex // Mutex to handle concurrent access to Charger object
}

func NewClient(serverAddr string, clientID int) (*WebSocketClient, error) {
	u := url.URL{Scheme: "ws", Host: serverAddr, Path: fmt.Sprintf("/%d", clientID)}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)

	if err != nil {
		return nil, fmt.Errorf("client %d failed to connect: %v", clientID, err)
	}

	log.Printf("Client %d Connected", clientID)

	return &WebSocketClient{
		conn: conn,
		Charger: &Charger{
			ID: fmt.Sprintf("Charger_%d", clientID), // Example ID formation
			ChargerPoints: []*ChargingPoint{
				{
					ID:     1,
					Status: "Available",
				},
				{
					ID:     2,
					Status: "Available",
				},
			},
		},
	}, nil
}

func parseMessage(message []json.RawMessage) (OCPPMessage, error) {
	var ocppMessage OCPPMessage

	if err := json.Unmarshal(message[0], &ocppMessage.MessageTypeID); err != nil {
		log.Printf("Failed to parse message type ID: %v", err)
		return ocppMessage, nil
	}

	if err := json.Unmarshal(message[1], &ocppMessage.MessageID); err != nil {
		log.Printf("Failed to parse message ID: %v", err)
		return ocppMessage, nil
	}

	if len(message) == 4 {
		if err := json.Unmarshal(message[2], &ocppMessage.Action); err != nil {
			log.Printf("Failed to parse action: %v", err)
			return ocppMessage, nil
		}

		ocppMessage.Payload = message[3]
	} else {
		ocppMessage.Payload = message[2]
	}

	return ocppMessage, nil
}

func (client *WebSocketClient) HandleMessages() {
	defer client.conn.Close()

	for {
		_, message, err := client.conn.ReadMessage()

		if err != nil {
			log.Printf("error reading message: %v", err)
			break
		}

		var msgParts []json.RawMessage

		if err := json.Unmarshal(message, &msgParts); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		msg, err := parseMessage(msgParts)

		if err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		switch msg.MessageTypeID {
		case 2:
			err := client.ProcessMessage(msg)
			if err != nil {
				log.Printf("Failed to process message: %v", err)
			}
		case 3:
			log.Printf("Received result: %v", msg)
		case 4:
			log.Printf("Server Error: %v", msg)
		default:
			log.Printf("unsupported message type: %d", msg.MessageTypeID)
		}
	}
}

func (client *WebSocketClient) ProcessMessage(msg OCPPMessage) error {
	client.mu.Lock()
	defer client.mu.Unlock()

	log.Printf("Processing message with ID %s and Action %s", msg.MessageID, msg.Action)

	switch msg.Action {
	case "TriggerMessage":
		return client.HandleTriggerMessage(msg)
	default:
		return client.SendNotImplemented(msg)
	}
}

func (client *WebSocketClient) sendMessage(msg OCPPMessage) error {
	messageArray := []interface{}{
		msg.MessageTypeID,
		msg.MessageID,
		msg.Action,
		msg.Payload,
	}

	message, err := json.Marshal(messageArray)

	if err != nil {
		return fmt.Errorf("failed to marshal message: %v", err)
	}

	if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	log.Printf("Sent Message %+v", messageArray)

	return nil
}
