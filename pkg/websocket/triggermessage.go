package websocket

import (
	"encoding/json"
	"fmt"
)

type TriggerMessageReq struct {
	ConnectorId      int    `json:"connectorId"`      // Assuming each trigger message specifies which connector
	RequestedMessage string `json:"requestedMessage"` // Specific action to perform, e.g., 'StatusNotification'
}

// HandleTriggerMessage handles incoming requests to trigger specific actions
func (client *WebSocketClient) HandleTriggerMessage(msg OCPPMessage) error {
	var req TriggerMessageReq

	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return fmt.Errorf("failed to unmarshal TriggerMessage: %v", err)
	}

	switch req.RequestedMessage {
	case "StatusNotification":
		return client.SendStatusNotification(req.ConnectorId)
	default:
		return fmt.Errorf("unsupported trigger action: %s", req.RequestedMessage)
	}
}
