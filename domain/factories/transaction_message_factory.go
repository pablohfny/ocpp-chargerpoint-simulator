package factories

import (
	"EV-Client-Simulator/domain/entities"
)

func CreateStartTransactionCall(connectorId int, idTag string) entities.Message {
	return CreateCallMessage("StartTransaction", map[string]interface{}{
		"idTag":       idTag,
		"connectorId": connectorId,
		"meterStart":  0,
	}, connectorId)
}

func CreateRemoteStartTransactionResult(connectorId int, callId string, payload map[string]interface{}) entities.Message {
	return CreateResultMessage(callId, "RemoteStartTransactionResult", payload, connectorId)
}

func CreateRemoteStopTransactionResult(connectorId int, callId string, payload map[string]interface{}) entities.Message {
	return CreateResultMessage(callId, "RemoteStartTransactionResult", payload, connectorId)
}

func CreateStopTransactionCall(connectorId int, transactionId int, meterStop float64) entities.Message {
	return CreateCallMessage("StopTransaction", map[string]interface{}{
		"transactionId": transactionId,
		"meterStop":     meterStop,
	}, connectorId)
}
