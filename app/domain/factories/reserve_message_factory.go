package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

// CreateReserveNowResult creates a ReserveNow.conf response
func CreateReserveNowResult(messageID string, status entities.ReservationStatus) entities.Message {
	payload := map[string]interface{}{
		"status": string(status),
	}
	return CreateResultMessage(messageID, payload, 0)
}

// CreateCancelReservationResult creates a CancelReservation.conf response
func CreateCancelReservationResult(messageID string, status entities.CancelReservationStatus) entities.Message {
	payload := map[string]interface{}{
		"status": string(status),
	}
	return CreateResultMessage(messageID, payload, 0)
}
