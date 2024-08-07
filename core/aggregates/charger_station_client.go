package aggregates

import (
	"EV-Client-Simulator/core/abstracts"
	"EV-Client-Simulator/core/entities"
)

type ChargerStationClient struct {
	ChargerStation entities.ChargerStation
	Client         abstracts.MessagingClient
}
