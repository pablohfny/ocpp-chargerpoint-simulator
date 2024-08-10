package factories

import (
	"EV-Client-Simulator/domain/entities"
)

func CreateResultMessage(id string, action string, payload map[string]interface{}) entities.Message {
	return entities.Message{
		Type:    3,
		ID:      id,
		Action:  action,
		Payload: payload,
	}
}
