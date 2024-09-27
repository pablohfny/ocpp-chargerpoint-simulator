package factories

import (
	"EV-Client-Simulator/domain/entities"
)

func CreateHeartbeatCall(payload map[string]interface{}) entities.Message {
	return CreateCallMessage("Heartbeat", payload, 0)
}
