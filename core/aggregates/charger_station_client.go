package aggregates

import (
	"EV-Client-Simulator/core/abstracts"
	"EV-Client-Simulator/core/entities"
	"EV-Client-Simulator/core/factories"
	"fmt"
	"time"
)

type ChargerStationClient struct {
	ChargerStation entities.ChargerStation
	Client         abstracts.MessagingClient
}

func NewChargerStationClient(client abstracts.MessagingClient) *ChargerStationClient {
	return &ChargerStationClient{
		ChargerStation: entities.NewChargerStation(client.GetId()),
		Client:         client,
	}
}

func (station *ChargerStationClient) Init() {
	time.Sleep(3 * time.Second)

	messagesChannel := make(chan entities.Message)

	go processMessages(messagesChannel)

	done := make(chan bool)

	go func() {
		defer close(messagesChannel)
		station.boot()
		station.Client.Listen(messagesChannel)
		done <- true
	}()

	<-done
}

func processMessages(channel chan entities.Message) {
	for message := range channel {
		switch message.Type {
		case 2:
			processCall(message)
		case 3:
			processResult(message)
		case 4:
			fmt.Printf("Server Error: %v", message)
		default:
			fmt.Printf("Unsupported message type: %d", message.Type)
		}
	}
}

func processCall(message entities.Message) {
	switch message.Action {
	default:
		fmt.Printf("Received call: %v\n", message)
	}
}

func processResult(message entities.Message) {
	fmt.Printf("Received result: %v\n", message)
}

func (station *ChargerStationClient) initHeartbeat() {
	station.Client.SendPeriodically(factories.CreateHeartbeatMessage(nil), 30*time.Second)
}

func (station *ChargerStationClient) notifyBoot() {
	station.Client.Send(factories.CreateBootNotificationMessage(nil))
}

func (station *ChargerStationClient) notifyAllStatuses() {
	for _, point := range station.ChargerStation.ChargerPoints {
		station.Client.Send(factories.CreateStatusNotificationMessage(point.ID, point.Status))
	}
}

func (station *ChargerStationClient) notifyStatus(connectorId int) {
	point := station.ChargerStation.GetPoint(connectorId)
	station.Client.Send(factories.CreateStatusNotificationMessage(connectorId, point.Status))
}

func (station *ChargerStationClient) boot() {
	station.initHeartbeat()
	station.notifyBoot()
	station.notifyAllStatuses()
}
