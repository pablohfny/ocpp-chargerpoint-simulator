package entities

import (
	"errors"
	"sync"
)

type ChargerPoint struct {
	ID                 int    `json:"id"`
	Status             string `json:"status"`
	CurrentTransaction int    `json:"currentTransaction"`
	mu                 *sync.Mutex
}

func NewChargerPoint(id int) *ChargerPoint {
	return &ChargerPoint{
		ID:     id,
		Status: "AVAILABLE",
		mu:     &sync.Mutex{},
	}
}

func (point *ChargerPoint) StartRemoteTransaction(idTag string) error {
	return nil
}

func (point *ChargerPoint) StartTransaction(transactionId int) error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.CurrentTransaction != 0 {
		return errors.New("point unavailable, transaction is already on course")
	}

	point.CurrentTransaction = transactionId
	return nil
}
