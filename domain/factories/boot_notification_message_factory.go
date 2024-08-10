package factories

import (
	"EV-Client-Simulator/domain/entities"
)

func CreateBootNotificationMessage(payload map[string]interface{}) entities.Message {
	return CreateCallMessage("BootNotification", payload)
}
