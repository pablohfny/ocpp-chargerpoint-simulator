package entities

type ChargerStation struct {
	Id            string          `json:"id"`
	ChargerPoints []ChargingPoint `json:"charger_points"`
	Status        string          `json:"status"`
}
