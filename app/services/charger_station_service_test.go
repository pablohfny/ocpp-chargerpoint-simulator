package services

import (
	"testing"

	"EV-Client-Simulator/app/domain/entities"
)

// newStopTestService builds a service around a fresh two-point station with
// buffered channels so processStopRemoteTransactionCall can run synchronously.
func newStopTestService() (*ChargerStationService, chan entities.Message) {
	station := entities.NewChargerStation("virtual")
	messageChannel := make(chan entities.Message, 4)
	errorsChannel := make(chan error, 4)
	return NewChargerStationSerice(&station, messageChannel, errorsChannel), messageChannel
}

// A RemoteStopTransaction for a transaction no point holds must answer
// Rejected instead of nil-dereferencing the missing point (this crashed the
// whole process when the platform stopped a charge that never started).
func TestProcessStopRemoteTransactionCallUnknownTransactionRejects(t *testing.T) {
	service, messages := newStopTestService()

	service.processStopRemoteTransactionCall(entities.Message{
		Type:    2,
		ID:      "call-1",
		Action:  "RemoteStopTransaction",
		Payload: map[string]any{"transactionId": float64(184113355)},
	})

	result := <-messages
	if status := result.Payload["status"]; status != "Rejected" {
		t.Fatalf("expected Rejected for unknown transaction, got %v", status)
	}
}

// A payload without a numeric transactionId must also answer Rejected instead
// of panicking on the type assertion.
func TestProcessStopRemoteTransactionCallMalformedPayloadRejects(t *testing.T) {
	service, messages := newStopTestService()

	service.processStopRemoteTransactionCall(entities.Message{
		Type:    2,
		ID:      "call-2",
		Action:  "RemoteStopTransaction",
		Payload: map[string]any{"transactionId": "not-a-number"},
	})

	result := <-messages
	if status := result.Payload["status"]; status != "Rejected" {
		t.Fatalf("expected Rejected for malformed payload, got %v", status)
	}
}
