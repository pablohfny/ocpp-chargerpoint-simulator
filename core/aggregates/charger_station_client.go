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

	callsChannel := make(chan entities.Message)
	resultsChannel := make(chan entities.Message)

	go func() {
		for msg := range callsChannel {
			fmt.Printf("Received call: %v\n", msg)
		}
	}()

	go func() {
		for msg := range resultsChannel {
			fmt.Printf("Received result: %v\n", msg)
		}
	}()

	done := make(chan bool)

	go func() {
		defer close(callsChannel)
		defer close(resultsChannel)
		station.boot()
		station.Client.Listen(callsChannel, resultsChannel)
		done <- true
	}()

	<-done
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
