package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

// CreateResetResult creates a Reset.conf response
func CreateResetResult(messageID string, status entities.ResetStatus) entities.Message {
	payload := map[string]interface{}{
		"status": string(status),
	}
	return CreateResultMessage(messageID, payload, 0)
}
