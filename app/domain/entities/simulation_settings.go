package entities

import (
	"math/rand/v2"
	"sync"
	"time"
)

// SimulationSettings controls the simulator behavior for testing
type SimulationSettings struct {
	ManualMode       bool                     `json:"manualMode"`
	Delays           DelaySettings            `json:"delays"`
	FailureRates     FailureRateSettings      `json:"failureRates"`
	ErrorInjection   ErrorInjectionSettings   `json:"errorInjection"`
	ChargingBehavior ChargingBehaviorSettings `json:"chargingBehavior"`
	Transaction      TransactionSettings      `json:"transaction"`
	mu               sync.RWMutex
}

// DelaySettings configures delays for various operations (in milliseconds)
type DelaySettings struct {
	BootNotificationDelayMs   int `json:"bootNotificationDelayMs"`
	AuthorizationDelayMs      int `json:"authorizationDelayMs"`
	StartTransactionDelayMs   int `json:"startTransactionDelayMs"`
	StopTransactionDelayMs    int `json:"stopTransactionDelayMs"`
	StatusNotificationDelayMs int `json:"statusNotificationDelayMs"`
	HeartbeatDelayMs          int `json:"heartbeatDelayMs"`
	MeterValuesDelayMs        int `json:"meterValuesDelayMs"`
	RemoteStartDelayMs        int `json:"remoteStartDelayMs"`
	RemoteStopDelayMs         int `json:"remoteStopDelayMs"`
}

// FailureRateSettings configures failure probabilities (0.0 to 1.0)
type FailureRateSettings struct {
	AuthorizationFailureRate    float64 `json:"authorizationFailureRate"`
	StartTransactionFailureRate float64 `json:"startTransactionFailureRate"`
	StopTransactionFailureRate  float64 `json:"stopTransactionFailureRate"`
	MeterValuesFailureRate      float64 `json:"meterValuesFailureRate"`
	HeartbeatFailureRate        float64 `json:"heartbeatFailureRate"`
}

// ErrorInjectionSettings configures error injection for testing error handling
type ErrorInjectionSettings struct {
	Enabled              bool    `json:"enabled"`
	InternalErrorRate    float64 `json:"internalErrorRate"`
	NotImplementedRate   float64 `json:"notImplementedRate"`
	NotSupportedRate     float64 `json:"notSupportedRate"`
	MessageTimeoutRate   float64 `json:"messageTimeoutRate"`
	MalformedMessageRate float64 `json:"malformedMessageRate"`
}

// ChargingBehaviorSettings configures charging simulation parameters
type ChargingBehaviorSettings struct {
	MinChargingPowerKW        float64 `json:"minChargingPowerKW"`
	MaxChargingPowerKW        float64 `json:"maxChargingPowerKW"`
	MeterValueIntervalSec     int     `json:"meterValueIntervalSec"`
	MeterIncrementIntervalSec int     `json:"meterIncrementIntervalSec"`
	AutoStopAtSOC             int     `json:"autoStopAtSOC"`
	EVDisconnectProbability   float64 `json:"evDisconnectProbability"`
}

// TransactionSettings configures per-transaction simulation parameters
type TransactionSettings struct {
	MeterStepWh          float64 `json:"meterStepWh"`          // Fixed Wh increment per tick (default: 1000)
	StartSOC             int     `json:"startSOC"`             // Starting SOC percentage (default: 0)
	FinalSOC             int     `json:"finalSOC"`             // Target SOC to trigger stop behavior (default: 100)
	FinalSOCBehavior     string  `json:"finalSOCBehavior"`     // "Finishing" or "SuspendedEV" (default: "Finishing")
	StopDelaySec         int     `json:"stopDelaySec"`         // Delay in seconds before StopTransaction after FinalSOC (default: 5)
	PreparingDurationSec int     `json:"preparingDurationSec"` // Seconds to stay in Preparing before Charging (default: 30)
}

// NewSimulationSettings creates simulation settings with sensible defaults
func NewSimulationSettings() *SimulationSettings {
	return &SimulationSettings{
		Delays: DelaySettings{
			BootNotificationDelayMs:   0,
			AuthorizationDelayMs:      0,
			StartTransactionDelayMs:   0,
			StopTransactionDelayMs:    0,
			StatusNotificationDelayMs: 0,
			HeartbeatDelayMs:          0,
			MeterValuesDelayMs:        0,
			RemoteStartDelayMs:        0,
			RemoteStopDelayMs:         0,
		},
		FailureRates: FailureRateSettings{
			AuthorizationFailureRate:    0.0,
			StartTransactionFailureRate: 0.0,
			StopTransactionFailureRate:  0.0,
			MeterValuesFailureRate:      0.0,
			HeartbeatFailureRate:        0.0,
		},
		ErrorInjection: ErrorInjectionSettings{
			Enabled:              false,
			InternalErrorRate:    0.0,
			NotImplementedRate:   0.0,
			NotSupportedRate:     0.0,
			MessageTimeoutRate:   0.0,
			MalformedMessageRate: 0.0,
		},
		ChargingBehavior: ChargingBehaviorSettings{
			MinChargingPowerKW:        7.0,
			MaxChargingPowerKW:        22.0,
			MeterValueIntervalSec:     5,
			MeterIncrementIntervalSec: 10,
			AutoStopAtSOC:             100,
			EVDisconnectProbability:   0.0,
		},
		Transaction: TransactionSettings{
			MeterStepWh:          1000,
			StartSOC:             0,
			FinalSOC:             100,
			FinalSOCBehavior:     "Finishing",
			StopDelaySec:         5,
			PreparingDurationSec: 10,
		},
	}
}

// ApplyDelay applies the configured delay for an operation
func (s *SimulationSettings) ApplyDelay(operation string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var delayMs int
	switch operation {
	case "BootNotification":
		delayMs = s.Delays.BootNotificationDelayMs
	case "Authorize":
		delayMs = s.Delays.AuthorizationDelayMs
	case "StartTransaction":
		delayMs = s.Delays.StartTransactionDelayMs
	case "StopTransaction":
		delayMs = s.Delays.StopTransactionDelayMs
	case "StatusNotification":
		delayMs = s.Delays.StatusNotificationDelayMs
	case "Heartbeat":
		delayMs = s.Delays.HeartbeatDelayMs
	case "MeterValues":
		delayMs = s.Delays.MeterValuesDelayMs
	case "RemoteStart":
		delayMs = s.Delays.RemoteStartDelayMs
	case "RemoteStop":
		delayMs = s.Delays.RemoteStopDelayMs
	}

	if delayMs > 0 {
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
}

// ShouldFail checks if an operation should fail based on configured failure rate
func (s *SimulationSettings) ShouldFail(operation string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rate float64
	switch operation {
	case "Authorize":
		rate = s.FailureRates.AuthorizationFailureRate
	case "StartTransaction":
		rate = s.FailureRates.StartTransactionFailureRate
	case "StopTransaction":
		rate = s.FailureRates.StopTransactionFailureRate
	case "MeterValues":
		rate = s.FailureRates.MeterValuesFailureRate
	case "Heartbeat":
		rate = s.FailureRates.HeartbeatFailureRate
	default:
		return false
	}

	return rand.Float64() < rate
}

// GetInjectedError returns an error type to inject, or empty string if none
func (s *SimulationSettings) GetInjectedError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.ErrorInjection.Enabled {
		return ""
	}

	r := rand.Float64()
	cumulative := 0.0

	cumulative += s.ErrorInjection.InternalErrorRate
	if r < cumulative {
		return "InternalError"
	}

	cumulative += s.ErrorInjection.NotImplementedRate
	if r < cumulative {
		return "NotImplemented"
	}

	cumulative += s.ErrorInjection.NotSupportedRate
	if r < cumulative {
		return "NotSupported"
	}

	cumulative += s.ErrorInjection.MessageTimeoutRate
	if r < cumulative {
		return "MessageTimeout"
	}

	cumulative += s.ErrorInjection.MalformedMessageRate
	if r < cumulative {
		return "MalformedMessage"
	}

	return ""
}

// GetManualMode returns the manual mode status
func (s *SimulationSettings) GetManualMode() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ManualMode
}

// SetManualMode toggles manual mode
func (s *SimulationSettings) SetManualMode(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ManualMode = enabled
}

// GetDelays returns a copy of delay settings
func (s *SimulationSettings) GetDelays() DelaySettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Delays
}

// SetDelays updates delay settings
func (s *SimulationSettings) SetDelays(delays DelaySettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Delays = delays
}

// GetFailureRates returns a copy of failure rate settings
func (s *SimulationSettings) GetFailureRates() FailureRateSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.FailureRates
}

// SetFailureRates updates failure rate settings
func (s *SimulationSettings) SetFailureRates(rates FailureRateSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FailureRates = rates
}

// GetErrorInjection returns a copy of error injection settings
func (s *SimulationSettings) GetErrorInjection() ErrorInjectionSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ErrorInjection
}

// SetErrorInjection updates error injection settings
func (s *SimulationSettings) SetErrorInjection(settings ErrorInjectionSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorInjection = settings
}

// GetChargingBehavior returns a copy of charging behavior settings
func (s *SimulationSettings) GetChargingBehavior() ChargingBehaviorSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ChargingBehavior
}

// SetChargingBehavior updates charging behavior settings
func (s *SimulationSettings) SetChargingBehavior(behavior ChargingBehaviorSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ChargingBehavior = behavior
}

// Reset resets all settings to defaults
func (s *SimulationSettings) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaults := NewSimulationSettings()
	s.ManualMode = false
	s.Delays = defaults.Delays
	s.FailureRates = defaults.FailureRates
	s.ErrorInjection = defaults.ErrorInjection
	s.ChargingBehavior = defaults.ChargingBehavior
	s.Transaction = defaults.Transaction
}

// GetTransaction returns a copy of transaction settings
func (s *SimulationSettings) GetTransaction() TransactionSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Transaction
}

// SetTransaction updates transaction settings
func (s *SimulationSettings) SetTransaction(tx TransactionSettings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Transaction = tx
}
