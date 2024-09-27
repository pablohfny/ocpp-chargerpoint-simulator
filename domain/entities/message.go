package entities

import (
	"encoding/json"
	"log"
)

type Message struct {
	Type        int8
	ID          string
	Action      string
	Payload     map[string]any
	ConnectorId int
}

func (message Message) ConvertToRawMessage() ([]byte, error) {
	messageArray := []any{
		message.Type,
		message.ID,
		message.Action,
		message.Payload,
	}

	return json.Marshal(messageArray)
}

func New(rawMessage []byte) (Message, error) {
	var message []json.RawMessage
	var ocppMessage Message

	if err := json.Unmarshal(rawMessage, &message); err != nil {
		log.Printf("Failed to parse message: %v", err)
		return ocppMessage, err
	}

	if err := json.Unmarshal(message[0], &ocppMessage.Type); err != nil {
		log.Printf("Failed to parse message type ID: %v", err)
		return ocppMessage, err
	}

	if err := json.Unmarshal(message[1], &ocppMessage.ID); err != nil {
		log.Printf("Failed to parse message ID: %v", err)
		return ocppMessage, err
	}

	if len(message) == 4 {
		if err := json.Unmarshal(message[2], &ocppMessage.Action); err != nil {
			log.Printf("Failed to parse action: %v", err)
			return ocppMessage, err
		}

		if err := json.Unmarshal(message[3], &ocppMessage.Payload); err != nil {
			log.Printf("Failed to parse payload: %v", err)
			return ocppMessage, err
		}

	} else {
		if err := json.Unmarshal(message[2], &ocppMessage.Payload); err != nil {
			log.Printf("Failed to parse action: %v", err)
			return ocppMessage, err
		}
	}

	return ocppMessage, nil
}
