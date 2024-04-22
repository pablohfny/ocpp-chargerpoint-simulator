package websocket

import (
	"encoding/json"
	"log"
)

func (client *WebSocketClient) HandleUnsupportedMessage(msg OCPPMessage) error {
	return client.SendNotImplemented(msg)
}

func (client *WebSocketClient) SendNotImplemented(msg OCPPMessage) error {
	log.Printf("Received unsupported action: %s", msg.Action)

	message := OCPPMessage{
		MessageTypeID: 3,
		MessageID:     msg.MessageID,
		Action:        "NotImplemented",
		Payload:       json.RawMessage(`{}`),
	}

	return client.sendMessage(message)
}
