package messaging

import (
	"EV-Client-Simulator/app/domain/abstracts"
	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/services"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type ChargerStationMessagingController struct {
	service                *services.ChargerStationService
	logService             *services.MessageLogService
	serverMessagesChannel  chan entities.Message
	stationMessagesChannel chan entities.Message
	errorsChannel          chan error
	client                 abstracts.MessagingClient
	wg                     sync.WaitGroup
	connected              bool
}

func NewChargerStationMessagingController(station *entities.ChargerStation, client abstracts.MessagingClient) ChargerStationMessagingController {
	serverMessagesChannel := make(chan entities.Message)
	stationMessagesChannel := make(chan entities.Message)
	errorsChannel := make(chan error)

	return ChargerStationMessagingController{
		service:                services.NewChargerStationSerice(station, stationMessagesChannel, errorsChannel),
		logService:             services.NewMessageLogService(1000),
		serverMessagesChannel:  serverMessagesChannel,
		stationMessagesChannel: stationMessagesChannel,
		errorsChannel:          errorsChannel,
		client:                 client,
		connected:              true,
	}
}

// GetService returns the charger station service
func (controller *ChargerStationMessagingController) GetService() *services.ChargerStationService {
	return controller.service
}

// GetLogService returns the message log service
func (controller *ChargerStationMessagingController) GetLogService() *services.MessageLogService {
	return controller.logService
}

// IsConnected returns the connection status
func (controller *ChargerStationMessagingController) IsConnected() *bool {
	return &controller.connected
}

// Reconnect re-opens the WebSocket, whether or not the simulator believes it is
// still up — a link that looks alive but has gone stale is precisely what the
// button is for. Listening has to be resumed here: the Listen goroutine ends
// when the connection drops, so a reconnect without it leaves the simulator
// deaf to everything the CSMS sends.
func (controller *ChargerStationMessagingController) Reconnect() error {
	controller.connected = false

	if err := controller.client.Reconnect(); err != nil {
		return err
	}

	controller.connected = true
	go controller.client.Listen(controller.serverMessagesChannel)

	return nil
}

// Disconnect closes the WebSocket connection
func (controller *ChargerStationMessagingController) Disconnect() error {
	if !controller.connected {
		return nil
	}

	controller.connected = false

	return controller.client.Disconnect()
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
		// Log incoming message
		rawBytes, _ := json.Marshal(message)
		controller.logService.LogIncoming(message, string(rawBytes))

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
		// Log outgoing message
		rawBytes, _ := json.Marshal(message)
		controller.logService.LogOutgoing(message, string(rawBytes))

		err := controller.client.Send(message, message.Type == 2)
		if err != nil {
			controller.Disconnect()
		}
	}
}
