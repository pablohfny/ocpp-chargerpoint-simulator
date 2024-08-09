package factories

import (
	"EV-Client-Simulator/core/entities"
)

func CreateStatusNotificationMessage(connectorId int, status string) entities.Message {
	return CreateCallMessage("BootNotification", map[string]any{
		"connectorId": connectorId,
		"status":      status,
	})
}
