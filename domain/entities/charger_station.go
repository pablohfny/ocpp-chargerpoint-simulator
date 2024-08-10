package entities

import (
	"errors"
)

type ChargerStation struct {
	ID            string          `json:"id"`
	ChargerPoints []*ChargerPoint `json:"charger_points"`
	Status        string          `json:"status"`
}

func NewChargerStation(id string) ChargerStation {
	return ChargerStation{
		ID:            id,
		ChargerPoints: []*ChargerPoint{NewChargerPoint(1), NewChargerPoint(2)},
		Status:        "ONLINE",
	}
}

func (station *ChargerStation) GetPoint(connectorId int) *ChargerPoint {
	for _, point := range station.ChargerPoints {
		if point.ID == connectorId {
			return point
		}
	}
	return nil
}

func (station *ChargerStation) StartRemoteTransaction(connectorId int, idTag string) error {

	point := station.GetPoint(connectorId)

	if point == nil {
		return errors.New("point not found")
	}

	point.StartRemoteTransaction(idTag)

	return nil
}
