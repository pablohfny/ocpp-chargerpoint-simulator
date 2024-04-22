package websocket

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type OCPPMessage struct {
	MessageTypeID int             `json:"messageTypeId"`
	MessageID     string          `json:"messageId"`
	Action        string          `json:"action"`
	Payload       json.RawMessage `json:"payload"`
}

type BootNotificationReq struct {
	ChargePointModel  string `json:"chargePointModel"`
	ChargePointVendor string `json:"chargePointVendor"`
}

func (client *WebSocketClient) SendBootNotification() error {
	uniqueID := uuid.New().String()

	req := BootNotificationReq{
		ChargePointModel:  "ModelX",
		ChargePointVendor: "VendorY",
	}

	reqJson, err := json.Marshal(req)

	if err != nil {
		return fmt.Errorf("failed to marshal BootNotification request: %v", err)
	}

	message := OCPPMessage{MessageTypeID: 2, MessageID: uniqueID, Action: "BootNotification", Payload: reqJson}

	return client.sendMessage(message)
}
