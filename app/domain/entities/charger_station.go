package entities

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

func (station *ChargerStation) GetPointByTransaction(transactionId int) *ChargerPoint {
	for _, point := range station.ChargerPoints {
		if point.CurrentTransaction == transactionId {
			return point
		}
	}
	return nil
}

func (station *ChargerStation) Authorize(isAuthorized bool) error {
	return nil
}

// GetPointByReservation finds a connector by reservation ID
func (station *ChargerStation) GetPointByReservation(reservationId int) *ChargerPoint {
	for _, point := range station.ChargerPoints {
		if point.ReservationID == reservationId {
			return point
		}
	}
	return nil
}

// GetAllPoints returns all charger points
func (station *ChargerStation) GetAllPoints() []*ChargerPoint {
	return station.ChargerPoints
}

// GetActiveTransactions returns all points with active transactions
func (station *ChargerStation) GetActiveTransactions() []*ChargerPoint {
	active := make([]*ChargerPoint, 0)
	for _, point := range station.ChargerPoints {
		if point.CurrentTransaction != 0 {
			active = append(active, point)
		}
	}
	return active
}

// Reset performs a station reset
func (station *ChargerStation) Reset(resetType ResetType) error {
	for _, point := range station.ChargerPoints {
		if resetType == ResetTypeHard {
			// Hard reset stops everything immediately
			if point.stop != nil {
				point.stop <- true
				close(point.stop)
				point.stop = nil
			}
			point.SetStatus(StatusAvailable)
		} else {
			// Soft reset waits for transactions to complete
			if point.CurrentTransaction == 0 {
				point.SetStatus(StatusAvailable)
			}
		}
	}
	return nil
}
