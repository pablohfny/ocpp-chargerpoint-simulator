package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

func CreateStatusNotificationCall(connectorId int, status string) entities.Message {
	return CreateCallMessage("StatusNotification", map[string]any{
		"connectorId": connectorId,
		"status":      status,
	}, connectorId)
}
