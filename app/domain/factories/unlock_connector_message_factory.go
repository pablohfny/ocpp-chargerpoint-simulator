package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

// CreateUnlockConnectorResult creates an UnlockConnector.conf response
func CreateUnlockConnectorResult(messageID string, status entities.UnlockStatus) entities.Message {
	payload := map[string]interface{}{
		"status": string(status),
	}
	return CreateResultMessage(messageID, payload, 0)
}
