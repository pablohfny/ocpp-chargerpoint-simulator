package entities

import "sync"

// OCPPConfigKey represents an OCPP 1.6 configuration key
type OCPPConfigKey struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Readonly bool   `json:"readonly"`
}

// OCPPConfiguration holds all OCPP configuration keys
type OCPPConfiguration struct {
	keys map[string]*OCPPConfigKey
	mu   sync.RWMutex
}

// NewOCPPConfiguration creates a new configuration with default OCPP 1.6 keys
func NewOCPPConfiguration() *OCPPConfiguration {
	config := &OCPPConfiguration{
		keys: make(map[string]*OCPPConfigKey),
	}
	config.initDefaults()
	return config
}

// initDefaults initializes standard OCPP 1.6 configuration keys
func (c *OCPPConfiguration) initDefaults() {
	defaults := []OCPPConfigKey{
		{Key: "AuthorizeRemoteTxRequests", Value: "true", Readonly: false},
		{Key: "ClockAlignedDataInterval", Value: "0", Readonly: false},
		{Key: "ConnectionTimeOut", Value: "60", Readonly: false},
		{Key: "GetConfigurationMaxKeys", Value: "100", Readonly: true},
		{Key: "HeartbeatInterval", Value: "30", Readonly: false},
		{Key: "LocalAuthorizeOffline", Value: "true", Readonly: false},
		{Key: "LocalPreAuthorize", Value: "false", Readonly: false},
		{Key: "MeterValuesAlignedData", Value: "Energy.Active.Import.Register", Readonly: false},
		{Key: "MeterValuesSampledData", Value: "Energy.Active.Import.Register,SoC", Readonly: false},
		{Key: "MeterValueSampleInterval", Value: "10", Readonly: true},
		{Key: "NumberOfConnectors", Value: "2", Readonly: true},
		{Key: "ResetRetries", Value: "3", Readonly: false},
		{Key: "ConnectorPhaseRotation", Value: "0.RST,1.RST,2.RST", Readonly: false},
		{Key: "StopTransactionOnEVSideDisconnect", Value: "true", Readonly: false},
		{Key: "StopTransactionOnInvalidId", Value: "true", Readonly: false},
		{Key: "StopTxnAlignedData", Value: "Energy.Active.Import.Register", Readonly: false},
		{Key: "StopTxnSampledData", Value: "Energy.Active.Import.Register,SoC", Readonly: false},
		{Key: "SupportedFeatureProfiles", Value: "Core,FirmwareManagement,LocalAuthListManagement,Reservation,SmartCharging,RemoteTrigger", Readonly: true},
		{Key: "TransactionMessageAttempts", Value: "3", Readonly: false},
		{Key: "TransactionMessageRetryInterval", Value: "30", Readonly: false},
		{Key: "UnlockConnectorOnEVSideDisconnect", Value: "true", Readonly: false},
		{Key: "WebSocketPingInterval", Value: "30", Readonly: false},
		{Key: "ChargePointModel", Value: "VirtualCharger", Readonly: true},
		{Key: "ChargePointVendor", Value: "NuCharge", Readonly: true},
		{Key: "FirmwareVersion", Value: "1.0.0", Readonly: true},
	}

	for _, key := range defaults {
		c.keys[key.Key] = &OCPPConfigKey{
			Key:      key.Key,
			Value:    key.Value,
			Readonly: key.Readonly,
		}
	}
}

// GetKey returns a configuration key by name
func (c *OCPPConfiguration) GetKey(key string) (*OCPPConfigKey, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	configKey, exists := c.keys[key]
	if !exists {
		return nil, false
	}
	return &OCPPConfigKey{
		Key:      configKey.Key,
		Value:    configKey.Value,
		Readonly: configKey.Readonly,
	}, true
}

// SetKey sets a configuration key value
func (c *OCPPConfiguration) SetKey(key, value string) ConfigurationStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	configKey, exists := c.keys[key]
	if !exists {
		return ConfigurationStatusNotSupported
	}

	if configKey.Readonly {
		return ConfigurationStatusRejected
	}

	configKey.Value = value
	return ConfigurationStatusAccepted
}

// GetAllKeys returns all configuration keys
func (c *OCPPConfiguration) GetAllKeys() []OCPPConfigKey {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := make([]OCPPConfigKey, 0, len(c.keys))
	for _, key := range c.keys {
		keys = append(keys, OCPPConfigKey{
			Key:      key.Key,
			Value:    key.Value,
			Readonly: key.Readonly,
		})
	}
	return keys
}

// GetKeys returns specific configuration keys by name
func (c *OCPPConfiguration) GetKeys(keyNames []string) ([]OCPPConfigKey, []string) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	found := make([]OCPPConfigKey, 0)
	unknown := make([]string, 0)

	for _, name := range keyNames {
		if key, exists := c.keys[name]; exists {
			found = append(found, OCPPConfigKey{
				Key:      key.Key,
				Value:    key.Value,
				Readonly: key.Readonly,
			})
		} else {
			unknown = append(unknown, name)
		}
	}

	return found, unknown
}
