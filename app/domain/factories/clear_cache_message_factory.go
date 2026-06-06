package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

// CreateClearCacheResult creates a ClearCache.conf response
func CreateClearCacheResult(messageID string, status entities.ClearCacheStatus) entities.Message {
	payload := map[string]interface{}{
		"status": string(status),
	}
	return CreateResultMessage(messageID, payload, 0)
}
