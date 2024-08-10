package messaging

import (
	"EV-Client-Simulator/app/services"
	"EV-Client-Simulator/domain/abstracts"
	"EV-Client-Simulator/domain/entities"
	"fmt"
	"time"
)

type ChargerStationMessagingController struct {
	service                *services.ChargerStationService
	serverMessagesChannel  chan entities.Message
	stationMessagesChannel chan entities.Message
	errorsChannel          chan error
	client                 abstracts.MessagingClient
}

func NewChargerStationMessagingController(station *entities.ChargerStation, client abstracts.MessagingClient) ChargerStationMessagingController {
	serverMessagesChannel := make(chan entities.Message)
	stationMessagesChannel := make(chan entities.Message)
	errorsChannel := make(chan error)

	return ChargerStationMessagingController{
		service:                services.NewChargerStationSerice(station, stationMessagesChannel),
		serverMessagesChannel:  serverMessagesChannel,
		stationMessagesChannel: stationMessagesChannel,
		errorsChannel:          errorsChannel,
		client:                 client,
	}
}

func (controller *ChargerStationMessagingController) Init() {
	time.Sleep(3 * time.Second)

	go controller.processMessages()
	go controller.processErrors()
	go controller.sendMessages()

	done := make(chan bool)

	go func() {
		defer close(controller.serverMessagesChannel)
		defer close(controller.stationMessagesChannel)
		defer close(controller.errorsChannel)
		controller.service.InitHeartbeat(30 * time.Second)
		controller.service.NotifyBoot()
		controller.service.NotifyStatuses()
		controller.client.Listen(controller.serverMessagesChannel)
		done <- true
	}()

	<-done
}

func (controller *ChargerStationMessagingController) processMessages() {
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
	for message := range controller.errorsChannel {
		fmt.Printf("Error: %v\n", message)
	}
}

func (controller *ChargerStationMessagingController) sendMessages() {
	for message := range controller.stationMessagesChannel {
		controller.client.Send(message, message.Type == 2)
	}
}
