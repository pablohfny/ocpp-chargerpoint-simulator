package services

import (
	"EV-Client-Simulator/domain/entities"
	"EV-Client-Simulator/domain/factories"
	"fmt"
	"sync"
	"time"
)

type ChargerStationService struct {
	station        *entities.ChargerStation
	messageChannel chan entities.Message
	errorsChannel  chan error
	mu             *sync.Mutex
}

func NewChargerStationSerice(station *entities.ChargerStation, messageChannel chan entities.Message) *ChargerStationService {
	return &ChargerStationService{
		station:        station,
		messageChannel: messageChannel,
		mu:             &sync.Mutex{},
	}
}

func (service *ChargerStationService) ProcessCall(message entities.Message) {
	switch message.Action {
	case "RemoteStartTransaction":
		service.startRemoteTransaction(message)
	default:
		fmt.Printf("Received call: %v\n", message)
	}
}

func (service *ChargerStationService) ProcessResult(message entities.Message) {
	fmt.Printf("Received result: %v\n", message)
}

func (service *ChargerStationService) startRemoteTransaction(message entities.Message) {
	connectorId := message.Payload["connectorId"].(int)
	idTag := message.Payload["idTag"].(string)

	if err := service.station.StartRemoteTransaction(connectorId, idTag); err != nil {
		service.errorsChannel <- err
		return
	}

	service.messageChannel <- factories.CreateRemoteTransactionResult(message.ID, nil)
}

func (service *ChargerStationService) InitHeartbeat(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				service.messageChannel <- factories.CreateHeartbeatMessage(nil)
			}
		}
	}()
}

func (service *ChargerStationService) NotifyStatuses() {
	for _, point := range service.station.ChargerPoints {
		service.messageChannel <- factories.CreateStatusNotificationMessage(point.ID, point.Status)
	}
}

func (service *ChargerStationService) NotifyBoot() {
	service.messageChannel <- factories.CreateBootNotificationMessage(nil)
}
