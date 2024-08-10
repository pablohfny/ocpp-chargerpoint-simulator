package factories

import (
	"EV-Client-Simulator/domain/entities"
)

func CreateStatusNotificationMessage(connectorId int, status string) entities.Message {
	return CreateCallMessage("StatusNotification", map[string]any{
		"connectorId": connectorId,
		"status":      status,
	})
}
