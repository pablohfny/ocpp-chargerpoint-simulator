package factories

import (
	"EV-Client-Simulator/domain/entities"
)

func CreateBootNotificationCall(payload map[string]interface{}) entities.Message {
	return CreateCallMessage("BootNotification", payload, 0)
}
