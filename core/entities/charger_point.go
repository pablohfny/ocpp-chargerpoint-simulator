package entities

type ChargerPoint struct {
	Id     int    `json:"id"`
	Status string `json:"status"`
}

func NewChargerPoint(id int) ChargerPoint {
	return ChargerPoint{
		Id:     id,
		Status: "AVAILABLE",
	}
}
