package messaging

import (
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/app/domain/abstracts"
	"EV-Client-Simulator/app/domain/entities"
	"fmt"
	"sync"
	"time"
)

type ChargerStationMessagingController struct {
	service                *services.ChargerStationService
	serverMessagesChannel  chan entities.Message
	stationMessagesChannel chan entities.Message
	errorsChannel          chan error
	client                 abstracts.MessagingClient
	wg                     sync.WaitGroup
}

func NewChargerStationMessagingController(station *entities.ChargerStation, client abstracts.MessagingClient) ChargerStationMessagingController {
	serverMessagesChannel := make(chan entities.Message)
	stationMessagesChannel := make(chan entities.Message)
	errorsChannel := make(chan error)

	return ChargerStationMessagingController{
		service:                services.NewChargerStationSerice(station, stationMessagesChannel, errorsChannel),
		serverMessagesChannel:  serverMessagesChannel,
		stationMessagesChannel: stationMessagesChannel,
		errorsChannel:          errorsChannel,
		client:                 client,
	}
}

func (controller *ChargerStationMessagingController) Init() {
	time.Sleep(3 * time.Second)

	defer close(controller.serverMessagesChannel)
	defer close(controller.stationMessagesChannel)
	defer close(controller.errorsChannel)

	controller.wg.Add(3)
	go controller.processMessages()
	go controller.processErrors()
	go controller.sendMessages()
	controller.service.InitHeartbeat(30 * time.Second)
	controller.service.NotifyBoot()
	controller.service.NotifyStatuses()

	controller.wg.Wait()
}

func (controller *ChargerStationMessagingController) processMessages() {
	defer controller.wg.Done()

	go controller.client.Listen(controller.serverMessagesChannel)

	for message := range controller.serverMessagesChannel {
		switch message.Type {
		case 2:
			controller.service.ProcessCall(message)
		case 3:
			controller.service.ProcessResult(message)
		case 4:
			fmt.Printf("Server Error: %v\n", message)
		default:
			fmt.Printf("Unsupported message type: %d\n", message.Type)
		}
	}
}

func (controller *ChargerStationMessagingController) processErrors() {
	defer controller.wg.Done()
	for message := range controller.errorsChannel {
		fmt.Printf("Error: %v\n", message)
	}
}

func (controller *ChargerStationMessagingController) sendMessages() {
	defer controller.wg.Done()
	for message := range controller.stationMessagesChannel {
		err := controller.client.Send(message, message.Type == 2)
		if err != nil {
			controller.close()
		}
	}
}

func (controller *ChargerStationMessagingController) close() {
	defer controller.wg.Done()
	controller.client.Disconnect()
}
