package entities

type ChargerStation struct {
	ID            string         `json:"id"`
	ChargerPoints []ChargerPoint `json:"charger_points"`
	Status        string         `json:"status"`
}

func NewChargerStation(id string) ChargerStation {
	return ChargerStation{
		ID:            id,
		ChargerPoints: []ChargerPoint{NewChargerPoint(1), NewChargerPoint(2)},
		Status:        "ONLINE",
	}
}
