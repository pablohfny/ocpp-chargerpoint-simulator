package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

func CreateResultMessage(id string, payload map[string]interface{}, connectorId int) entities.Message {
	return entities.Message{
		Type:        3,
		ID:          id,
		Payload:     payload,
		ConnectorId: connectorId,
	}
}
