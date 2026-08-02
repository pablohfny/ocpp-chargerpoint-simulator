package entities

import (
	"EV-Client-Simulator/utils"
	"errors"
	"math"
	"math/rand/v2"
	"sync"
	"time"
)

// MeterQueueEntry represents a queued meter value snapshot
type MeterQueueEntry struct {
	MeterValue float64   `json:"meterValue"`
	Soc        int16     `json:"soc"`
	Timestamp  time.Time `json:"timestamp"`
}

// ChargerPoint represents an OCPP 1.6 connector with full state support
type ChargerPoint struct {
	ID                 int                  `json:"id"`
	Status             ConnectorStatus      `json:"status"`
	ErrorCode          ChargePointErrorCode `json:"errorCode"`
	ErrorInfo          string               `json:"errorInfo,omitempty"`
	VendorErrorCode    string               `json:"vendorErrorCode,omitempty"`
	CurrentTransaction int                  `json:"currentTransaction"`
	CurrentIdTag       string               `json:"currentIdTag,omitempty"`
	MeterValue         float64              `json:"meterValue"`
	Soc                int16                `json:"soc"`
	ReservationID      int                  `json:"reservationId,omitempty"`
	ReservationIdTag   string               `json:"reservationIdTag,omitempty"`
	ReservationExpiry  time.Time            `json:"reservationExpiry,omitempty"`
	CablePlugged       bool                 `json:"cablePlugged"`
	MeterQueue         []MeterQueueEntry    `json:"meterQueue,omitempty"`
	stop               chan bool
	meterQueueStop     chan bool
	mu                 *sync.Mutex
}

func NewChargerPoint(id int) *ChargerPoint {
	return &ChargerPoint{
		ID:           id,
		Status:       StatusAvailable,
		ErrorCode:    ErrorNoError,
		MeterValue:   0,
		Soc:          0,
		CablePlugged: false,
		mu:           &sync.Mutex{},
	}
}

func (point *ChargerPoint) SetStatus(status ConnectorStatus) error {
	point.mu.Lock()
	defer point.mu.Unlock()

	point.Status = status

	if status == StatusAvailable {
		point.RemoveCurrentTransaction()
		point.ErrorCode = ErrorNoError
		point.ErrorInfo = ""
		point.VendorErrorCode = ""
	}

	return nil
}

// SetStatusString sets the status from a string value (for compatibility)
func (point *ChargerPoint) SetStatusString(status string) error {
	return point.SetStatus(ConnectorStatus(status))
}

// GetStatusString returns the status as a string
func (point *ChargerPoint) GetStatusString() string {
	point.mu.Lock()
	defer point.mu.Unlock()
	return string(point.Status)
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
	point.MeterQueue = nil
	return nil
}

// EnqueueMeter snapshots the current meter value and SoC into the queue
func (point *ChargerPoint) EnqueueMeter() {
	point.mu.Lock()
	defer point.mu.Unlock()
	point.MeterQueue = append(point.MeterQueue, MeterQueueEntry{
		MeterValue: point.MeterValue,
		Soc:        point.Soc,
		Timestamp:  time.Now(),
	})
}

// DequeueMeter pops the first meter value from the queue
func (point *ChargerPoint) DequeueMeter() *MeterQueueEntry {
	point.mu.Lock()
	defer point.mu.Unlock()
	if len(point.MeterQueue) == 0 {
		return nil
	}
	entry := point.MeterQueue[0]
	point.MeterQueue = point.MeterQueue[1:]
	return &entry
}

// GetMeterQueueSnapshot returns a copy of the meter queue
func (point *ChargerPoint) GetMeterQueueSnapshot() []MeterQueueEntry {
	point.mu.Lock()
	defer point.mu.Unlock()
	result := make([]MeterQueueEntry, len(point.MeterQueue))
	copy(result, point.MeterQueue)
	return result
}

// FlushMeterQueue clears all entries from the meter queue and returns the count
func (point *ChargerPoint) FlushMeterQueue() int {
	point.mu.Lock()
	defer point.mu.Unlock()
	count := len(point.MeterQueue)
	point.MeterQueue = nil
	return count
}

// StartMeterQueuing initializes the meter queue stop channel and returns it
func (point *ChargerPoint) StartMeterQueuing() chan bool {
	point.StopMeterQueue() // clean up any existing
	point.meterQueueStop = make(chan bool, 1)
	return point.meterQueueStop
}

// StopMeterQueue stops the meter queuing goroutine if running
func (point *ChargerPoint) StopMeterQueue() {
	if point.meterQueueStop != nil {
		select {
		case point.meterQueueStop <- true:
		default:
		}
		close(point.meterQueueStop)
		point.meterQueueStop = nil
	}
}

func (point *ChargerPoint) StartRemoteTransaction() error {
	if point.Status != StatusAvailable && point.Status != StatusReserved {
		return errors.New("point is not available")
	}

	point.SetStatus(StatusPreparing)

	return nil
}

func (point *ChargerPoint) Authorize(idTag string) error {
	if point.Status == StatusAvailable || point.Status == StatusPreparing || point.Status == StatusReserved {
		point.CurrentIdTag = idTag
		return nil
	}

	return errors.New("point is not available or not preparing")
}

func (point *ChargerPoint) StartTransaction(transactionId int, txSettings TransactionSettings) error {
	err := point.SetCurrentTransaction(transactionId)

	if err != nil {
		return err
	}

	// Clear any reservation when starting transaction
	point.ReservationID = 0
	point.ReservationIdTag = ""
	point.ReservationExpiry = time.Time{}

	point.SetStatus(StatusCharging)
	point.startMeterIncrement(txSettings)

	return nil
}

// StartChargingWithSettings transitions from Preparing to Charging and starts meter increment.
// Expects SetCurrentTransaction to have been called already.
func (point *ChargerPoint) StartChargingWithSettings(txSettings TransactionSettings) error {
	if point.CurrentTransaction == 0 {
		return errors.New("no current transaction set")
	}

	// Clear any reservation when starting transaction
	point.ReservationID = 0
	point.ReservationIdTag = ""
	point.ReservationExpiry = time.Time{}

	point.SetStatus(StatusCharging)
	point.startMeterIncrement(txSettings)

	return nil
}

func (point *ChargerPoint) startMeterIncrement(tx TransactionSettings) {
	point.stop = make(chan bool)
	point.Soc = int16(tx.StartSOC)
	stopValue := 10000 + rand.Float64()*1000
	finalSOCReached := false

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				// Skip increment while suspended
				if point.Status == StatusSuspendedEV || point.Status == StatusSuspendedEVSE {
					continue
				}

				point.MeterValue = utils.RoundFloat(point.MeterValue+tx.MeterStepWh, 2)
				point.Soc = int16(math.Max(math.Min((point.MeterValue/stopValue)*100, 100), 0))

				// Check configurable FinalSOC target
				if !finalSOCReached && tx.FinalSOC > 0 && point.Soc >= int16(tx.FinalSOC) {
					finalSOCReached = true
					if tx.FinalSOCBehavior == "SuspendedEV" {
						point.SetStatus(StatusSuspendedEV)
					}
					go func() {
						if tx.StopDelaySec > 0 {
							time.Sleep(time.Duration(tx.StopDelaySec) * time.Second)
						}
						point.StopTransaction()
					}()
					continue
				}

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
	point.SetStatus(StatusFinishing)

	if point.stop != nil {
		point.stop <- true
		close(point.stop)
		point.stop = nil
	}
	return nil
}

// SetFault sets a fault on the connector
func (point *ChargerPoint) SetFault(errorCode ChargePointErrorCode, info string, vendorCode string) error {
	point.mu.Lock()
	defer point.mu.Unlock()

	point.Status = StatusFaulted
	point.ErrorCode = errorCode
	point.ErrorInfo = info
	point.VendorErrorCode = vendorCode
	return nil
}

// ClearFault clears the fault and returns to Available
func (point *ChargerPoint) ClearFault() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status != StatusFaulted {
		return errors.New("connector is not in faulted state")
	}

	point.Status = StatusAvailable
	point.ErrorCode = ErrorNoError
	point.ErrorInfo = ""
	point.VendorErrorCode = ""
	return nil
}

// SetReservation sets a reservation on the connector
func (point *ChargerPoint) SetReservation(reservationID int, idTag string, expiry time.Time) error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status != StatusAvailable {
		return errors.New("connector is not available for reservation")
	}

	point.Status = StatusReserved
	point.ReservationID = reservationID
	point.ReservationIdTag = idTag
	point.ReservationExpiry = expiry
	return nil
}

// CancelReservation cancels the current reservation
func (point *ChargerPoint) CancelReservation() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status != StatusReserved {
		return errors.New("connector has no active reservation")
	}

	point.Status = StatusAvailable
	point.ReservationID = 0
	point.ReservationIdTag = ""
	point.ReservationExpiry = time.Time{}
	return nil
}

// SetUnavailable sets the connector to unavailable
func (point *ChargerPoint) SetUnavailable() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status == StatusCharging {
		return errors.New("cannot set unavailable while charging")
	}

	point.Status = StatusUnavailable
	return nil
}

// SetAvailable sets the connector back to available
func (point *ChargerPoint) SetAvailable() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status == StatusCharging || point.Status == StatusPreparing {
		return errors.New("cannot set available during active session")
	}

	point.Status = StatusAvailable
	point.ErrorCode = ErrorNoError
	point.ErrorInfo = ""
	return nil
}

// PlugCable simulates plugging in the cable
func (point *ChargerPoint) PlugCable() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.CablePlugged {
		return errors.New("cable already plugged")
	}

	point.CablePlugged = true
	return nil
}

// UnplugCable simulates unplugging the cable
func (point *ChargerPoint) UnplugCable() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if !point.CablePlugged {
		return errors.New("cable not plugged")
	}

	point.CablePlugged = false
	// The battery belongs to the EV: once the car leaves, its state of charge
	// leaves with it. Without this reset the last session's SoC (often 100%)
	// lingers on the UI with no vehicle present.
	point.Soc = 0
	point.MeterValue = 0
	return nil
}

// IsReservationExpired checks if the current reservation has expired
func (point *ChargerPoint) IsReservationExpired() bool {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status != StatusReserved {
		return false
	}

	return time.Now().After(point.ReservationExpiry)
}

// CheckReservationIdTag checks if the idTag matches the reservation
func (point *ChargerPoint) CheckReservationIdTag(idTag string) bool {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status != StatusReserved {
		return true // No reservation, any tag is ok
	}

	return point.ReservationIdTag == idTag
}

// SuspendEV suspends charging from EV side
func (point *ChargerPoint) SuspendEV() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status != StatusCharging && point.Status != StatusSuspendedEVSE {
		return errors.New("not charging or suspended")
	}

	point.Status = StatusSuspendedEV
	return nil
}

// SuspendEVSE suspends charging from EVSE side
func (point *ChargerPoint) SuspendEVSE() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status != StatusCharging && point.Status != StatusSuspendedEV {
		return errors.New("not charging or suspended")
	}

	point.Status = StatusSuspendedEVSE
	return nil
}

// ResumeCharging resumes charging from suspended state
func (point *ChargerPoint) ResumeCharging() error {
	point.mu.Lock()
	defer point.mu.Unlock()

	if point.Status != StatusSuspendedEV && point.Status != StatusSuspendedEVSE {
		return errors.New("not in suspended state")
	}

	point.Status = StatusCharging
	return nil
}
