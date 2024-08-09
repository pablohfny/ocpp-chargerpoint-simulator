package entities

type ChargerPoint struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

func NewChargerPoint(id int) *ChargerPoint {
	return &ChargerPoint{
		ID:     id,
		Status: "AVAILABLE",
	}
}
