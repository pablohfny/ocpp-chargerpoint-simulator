package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

// CreateDataTransferResult creates a DataTransfer.conf response
func CreateDataTransferResult(messageID string, status entities.DataTransferStatus, data string) entities.Message {
	payload := map[string]interface{}{
		"status": string(status),
	}
	if data != "" {
		payload["data"] = data
	}
	return CreateResultMessage(messageID, payload, 0)
}

// CreateDataTransferCall creates a DataTransfer.req call (station to server)
func CreateDataTransferCall(vendorId string, messageId string, data string) entities.Message {
	payload := map[string]interface{}{
		"vendorId": vendorId,
	}
	if messageId != "" {
		payload["messageId"] = messageId
	}
	if data != "" {
		payload["data"] = data
	}
	return CreateCallMessage("DataTransfer", payload, 0)
}
