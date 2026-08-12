package messaging

import (
	"errors"
	"sync"
	"testing"
	"time"

	"EV-Client-Simulator/app/domain/entities"
)

// mockMessagingClient records the calls the controller makes, which is all the
// connection lifecycle needs to be judged on.
type mockMessagingClient struct {
	mu             sync.Mutex
	disconnects    int
	reconnects     int
	listens        int
	reconnectError error
}

func (mock *mockMessagingClient) GetId() string { return "virtual" }

func (mock *mockMessagingClient) GetConn() any { return nil }

func (mock *mockMessagingClient) Listen(messagesChannel chan entities.Message) {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.listens++
}

func (mock *mockMessagingClient) Send(message entities.Message, expectResult bool) error {
	return nil
}

func (mock *mockMessagingClient) SendPeriodically(message entities.Message, expectResult bool, interval time.Duration) error {
	return nil
}

func (mock *mockMessagingClient) Disconnect() error {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	mock.disconnects++
	return nil
}

func (mock *mockMessagingClient) Reconnect() error {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.reconnectError != nil {
		return mock.reconnectError
	}
	mock.reconnects++
	return nil
}

func (mock *mockMessagingClient) counts() (int, int, int) {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	return mock.disconnects, mock.reconnects, mock.listens
}

func newTestController(mock *mockMessagingClient) ChargerStationMessagingController {
	station := entities.NewChargerStation("virtual")
	return NewChargerStationMessagingController(&station, mock)
}

func TestDisconnectIsIdempotent(t *testing.T) {
	mockClient := &mockMessagingClient{}
	controller := newTestController(mockClient)

	if err := controller.Disconnect(); err != nil {
		t.Fatalf("first disconnect failed: %v", err)
	}
	if err := controller.Disconnect(); err != nil {
		t.Fatalf("second disconnect failed: %v", err)
	}

	actualDisconnects, _, _ := mockClient.counts()
	if actualDisconnects != 1 {
		t.Errorf("expected the client to be closed once, got %d", actualDisconnects)
	}
	if *controller.IsConnected() {
		t.Error("expected the controller to report a dropped connection")
	}
}

func TestReconnectResumesListening(t *testing.T) {
	mockClient := &mockMessagingClient{}
	controller := newTestController(mockClient)
	controller.Disconnect()

	if err := controller.Reconnect(); err != nil {
		t.Fatalf("reconnect failed: %v", err)
	}

	if !*controller.IsConnected() {
		t.Error("expected the controller to report a live connection")
	}

	waitForListens(t, mockClient, 1)
}

// waitForListens waits for the listener goroutine the reconnect spawns, since a
// count read straight after the call would race it.
func waitForListens(t *testing.T, mockClient *mockMessagingClient, expected int) {
	t.Helper()

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, _, actual := mockClient.counts(); actual == expected {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	_, _, actual := mockClient.counts()
	t.Errorf("expected %d listener(s) after the reconnect, got %d", expected, actual)
}

// A live-looking connection must still be re-dialable: a socket that went stale
// without anyone noticing is exactly what the button is pressed for.
func TestReconnectWhileConnectedRedials(t *testing.T) {
	mockClient := &mockMessagingClient{}
	controller := newTestController(mockClient)

	if err := controller.Reconnect(); err != nil {
		t.Fatalf("reconnect on a live connection failed: %v", err)
	}

	if !*controller.IsConnected() {
		t.Error("expected the controller to report a live connection")
	}

	waitForListens(t, mockClient, 1)

	_, actualReconnects, _ := mockClient.counts()
	if actualReconnects != 1 {
		t.Errorf("expected exactly one re-dial, got %d", actualReconnects)
	}
}

func TestReconnectKeepsConnectionDownOnFailure(t *testing.T) {
	mockClient := &mockMessagingClient{reconnectError: errors.New("dial failed")}
	controller := newTestController(mockClient)
	controller.Disconnect()

	if err := controller.Reconnect(); err == nil {
		t.Fatal("expected the dial failure to surface")
	}

	if *controller.IsConnected() {
		t.Error("expected the controller to stay disconnected after a failed dial")
	}
}
