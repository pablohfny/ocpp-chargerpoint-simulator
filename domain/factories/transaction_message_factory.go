package factories

import (
	"EV-Client-Simulator/domain/entities"
)

func CreateRemoteTransactionResult(id string, payload map[string]interface{}) entities.Message {
	return CreateResultMessage(id, "RemoteStartTransactionResult", payload)
}
