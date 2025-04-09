package factories

import (
	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/domain/factories/utils"
)

func CreateCallMessage(action string, payload map[string]interface{}, connectorId int) entities.Message {
	return entities.Message{
		Type:        2,
		ID:          utils.GenerateUUIDV4(),
		Action:      action,
		Payload:     payload,
		ConnectorId: connectorId,
	}
}
