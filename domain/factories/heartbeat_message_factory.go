package factories

import (
	"EV-Client-Simulator/domain/entities"
)

func CreateHeartbeatMessage(payload map[string]interface{}) entities.Message {
	return CreateCallMessage("Heartbeat", payload)
}
