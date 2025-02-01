package entities

import (
	"EV-Client-Simulator/utils"
	"errors"
	"math"
	"math/rand/v2"
	"sync"
	"time"
)

type ChargerPoint struct {
	ID                 int    `json:"id"`
	Status             string `json:"status"`
	CurrentTransaction int    `json:"currentTransaction"`
	MeterValue         float64
	Soc                int16
	stop               chan bool
	mu                 *sync.Mutex
}

func NewChargerPoint(id int) *ChargerPoint {
	return &ChargerPoint{
		ID:         id,
		Status:     "AVAILABLE",
		MeterValue: 0,
		Soc:        0,
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
	point.Soc = 0
	stopValue := 60000 + rand.Float64()*30000

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				increment := 1000 + rand.Float64()*2000
				point.MeterValue = utils.RoundFloat(point.MeterValue+increment, 2)
				point.Soc = int16(math.Max(math.Min((point.MeterValue/stopValue)*100, 100), 0))

				if point.MeterValue >= stopValue {
					point.StopTransaction()
					return
				}
			case <-point.stop:
				return
			}
		}
	}()
}

func (point *ChargerPoint) StopTransaction() error {
	point.SetStatus("FINISHING")

	if point.stop != nil {
		point.stop <- true
		close(point.stop)
		point.stop = nil
	}
	return nil
}
