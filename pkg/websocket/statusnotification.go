package websocket

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
)

type StatusNotificationReq struct {
	ConnectorID int    `json:"connectorId"`
	Status      string `json:"status"`
	ErrorCode   string `json:"errorCode"`
}

type StatusNotificationConf struct {
	ConnectorID int    `json:"connectorId"`
	Status      string `json:"status"`
	ErrorCode   string `json:"errorCode"`
}

func (client *WebSocketClient) SendStatusNotification(connectorID int) error {
	for _, point := range client.Charger.ChargerPoints {
		if point.ID == connectorID {
			resp := StatusNotificationConf{
				ConnectorID: point.ID,
				Status:      point.Status,
				ErrorCode:   point.ErrorCode,
			}

			respJson, err := json.Marshal(resp)

			if err != nil {
				log.Printf("Failed to marshal StatusNotification response: %v", err)
				return err
			}

			message := OCPPMessage{
				MessageTypeID: 2,
				MessageID:     uuid.New().String(),
				Action:        "StatusNotification",
				Payload:       respJson,
			}

			if err := client.sendMessage(message); err != nil {
				log.Printf("Failed to send StatusNotification response: %v", err)
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("connectorId not found: %v", connectorID)
}
