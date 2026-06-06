package services

import (
	"EV-Client-Simulator/app/domain/entities"
)

// ConfigurationService manages OCPP configuration keys
type ConfigurationService struct {
	config *entities.OCPPConfiguration
}

// NewConfigurationService creates a new configuration service
func NewConfigurationService() *ConfigurationService {
	return &ConfigurationService{
		config: entities.NewOCPPConfiguration(),
	}
}

// GetConfiguration returns configuration keys
// If keys is nil or empty, returns all keys
func (s *ConfigurationService) GetConfiguration(keys []string) ([]entities.OCPPConfigKey, []string) {
	if len(keys) == 0 {
		return s.config.GetAllKeys(), nil
	}
	return s.config.GetKeys(keys)
}

// ChangeConfiguration updates a configuration key
func (s *ConfigurationService) ChangeConfiguration(key, value string) entities.ConfigurationStatus {
	return s.config.SetKey(key, value)
}

// GetKey returns a specific configuration key
func (s *ConfigurationService) GetKey(key string) (*entities.OCPPConfigKey, bool) {
	return s.config.GetKey(key)
}

// GetAllKeys returns all configuration keys
func (s *ConfigurationService) GetAllKeys() []entities.OCPPConfigKey {
	return s.config.GetAllKeys()
}
