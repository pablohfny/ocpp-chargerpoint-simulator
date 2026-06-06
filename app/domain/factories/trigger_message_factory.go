package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

// CreateTriggerMessageResult creates a TriggerMessage.conf response
func CreateTriggerMessageResult(messageID string, status entities.TriggerMessageStatus) entities.Message {
	payload := map[string]interface{}{
		"status": string(status),
	}
	return CreateResultMessage(messageID, payload, 0)
}
