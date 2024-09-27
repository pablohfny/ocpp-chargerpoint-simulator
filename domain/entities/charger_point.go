package entities

import (
	"EV-Client-Simulator/utils"
	"errors"
	"math/rand/v2"
	"sync"
	"time"
)

type ChargerPoint struct {
	ID                 int    `json:"id"`
	Status             string `json:"status"`
	CurrentTransaction int    `json:"currentTransaction"`
	MeterValue         float64
	stop               chan bool
	mu                 *sync.Mutex
}

func NewChargerPoint(id int) *ChargerPoint {
	return &ChargerPoint{
		ID:         id,
		Status:     "AVAILABLE",
		MeterValue: 0,
		mu:         &sync.Mutex{},
	}
}

func (point *ChargerPoint) SetStatus(status string) error {
	point.mu.Lock()
	defer point.mu.Unlock()

	point.Status = status

	if status == "AVAILABLE" {
		point.RemoveCurrentTransaction()
	}

	return nil
}

func (point *ChargerPoint) SetCurrentTransaction(transactionId int) error {
	if point.CurrentTransaction != 0 {
		return errors.New("point already has current transaction")
	}

	point.CurrentTransaction = transactionId
	return nil
}

func (point *ChargerPoint) RemoveCurrentTransaction() error {
	point.CurrentTransaction = 0
	point.MeterValue = 0
	return nil
}

func (point *ChargerPoint) StartRemoteTransaction() error {
	if point.Status != "AVAILABLE" {
		return errors.New("point is not available")
	}

	point.SetStatus("PREPARING")

	return nil
}

func (point *ChargerPoint) Authorize(idTag string) error {
	if point.Status == "AVAILABLE" || point.Status == "PREPARING" {
		return nil
	}

	return errors.New("point is not available or not preparing")
}

func (point *ChargerPoint) StartTransaction(transactionId int) error {
	err := point.SetCurrentTransaction(transactionId)

	if err != nil {
		return err
	}

	point.SetStatus("CHARGING")
	point.startMeterIncrement()

	return nil
}

func (point *ChargerPoint) startMeterIncrement() {
	point.stop = make(chan bool)
	stopValue := 100 + rand.Float64()*500

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				increment := 5 + rand.Float64()*10
				point.MeterValue = utils.RoundFloat(point.MeterValue+increment, 2)
				if point.MeterValue >= stopValue {
					point.StopTransaction()
				}
			case <-point.stop:
				return
			}
		}
	}()
}

func (point *ChargerPoint) StopTransaction() error {
	if point.stop != nil {
		close(point.stop)
		point.stop = nil
	}

	return point.SetStatus("FINISHING")
}
