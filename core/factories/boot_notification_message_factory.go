package factories

import (
	"EV-Client-Simulator/core/entities"
)

func CreateBootNotificationMessage(payload map[string]interface{}) entities.Message {
	return CreateCallMessage("BootNotification", payload)
}
