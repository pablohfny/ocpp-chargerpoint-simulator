package entities

type Transaction struct {
	ID                 int    `json:"id"`
	Status             string `json:"status"`
	CurrentTransaction int    `json:"currentTransaction"`
}
