package services

import (
	"EV-Client-Simulator/app/domain/entities"
)

// SimulationService manages simulation settings for testing
type SimulationService struct {
	settings *entities.SimulationSettings
}

// NewSimulationService creates a new simulation service with default settings
func NewSimulationService() *SimulationService {
	return &SimulationService{
		settings: entities.NewSimulationSettings(),
	}
}

// ApplyDelay applies configured delay for an operation
func (s *SimulationService) ApplyDelay(operation string) {
	s.settings.ApplyDelay(operation)
}

// ShouldFail checks if an operation should fail based on failure rate
func (s *SimulationService) ShouldFail(operation string) bool {
	return s.settings.ShouldFail(operation)
}

// GetInjectedError returns an error to inject, or empty string if none
func (s *SimulationService) GetInjectedError() string {
	return s.settings.GetInjectedError()
}

// GetSettings returns all simulation settings
func (s *SimulationService) GetSettings() *entities.SimulationSettings {
	return s.settings
}

// GetDelays returns delay settings
func (s *SimulationService) GetDelays() entities.DelaySettings {
	return s.settings.GetDelays()
}

// SetDelays updates delay settings
func (s *SimulationService) SetDelays(delays entities.DelaySettings) {
	s.settings.SetDelays(delays)
}

// GetFailureRates returns failure rate settings
func (s *SimulationService) GetFailureRates() entities.FailureRateSettings {
	return s.settings.GetFailureRates()
}

// SetFailureRates updates failure rate settings
func (s *SimulationService) SetFailureRates(rates entities.FailureRateSettings) {
	s.settings.SetFailureRates(rates)
}

// GetErrorInjection returns error injection settings
func (s *SimulationService) GetErrorInjection() entities.ErrorInjectionSettings {
	return s.settings.GetErrorInjection()
}

// SetErrorInjection updates error injection settings
func (s *SimulationService) SetErrorInjection(settings entities.ErrorInjectionSettings) {
	s.settings.SetErrorInjection(settings)
}

// GetChargingBehavior returns charging behavior settings
func (s *SimulationService) GetChargingBehavior() entities.ChargingBehaviorSettings {
	return s.settings.GetChargingBehavior()
}

// SetChargingBehavior updates charging behavior settings
func (s *SimulationService) SetChargingBehavior(behavior entities.ChargingBehaviorSettings) {
	s.settings.SetChargingBehavior(behavior)
}

// GetTransactionSettings returns transaction settings
func (s *SimulationService) GetTransactionSettings() entities.TransactionSettings {
	return s.settings.GetTransaction()
}

// SetTransactionSettings updates transaction settings
func (s *SimulationService) SetTransactionSettings(tx entities.TransactionSettings) {
	s.settings.SetTransaction(tx)
}

// IsManualMode returns whether manual mode is enabled
func (s *SimulationService) IsManualMode() bool {
	return s.settings.GetManualMode()
}

// SetManualMode toggles manual mode
func (s *SimulationService) SetManualMode(enabled bool) {
	s.settings.SetManualMode(enabled)
}

// Reset resets all settings to defaults
func (s *SimulationService) Reset() {
	s.settings.Reset()
}
