package factories

import (
	"EV-Client-Simulator/core/entities"
	"EV-Client-Simulator/core/factories/utils"
)

func CreateCallMessage(action string, payload map[string]interface{}) entities.Message {
	return entities.Message{
		Type:    2,
		ID:      utils.GenerateUUIDV4(),
		Action:  action,
		Payload: payload,
	}
}
