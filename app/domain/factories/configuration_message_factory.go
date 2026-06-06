package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

// CreateGetConfigurationResult creates a GetConfiguration.conf response
func CreateGetConfigurationResult(messageID string, configurationKey []entities.OCPPConfigKey, unknownKey []string) entities.Message {
	// Convert config keys to map format for OCPP
	keys := make([]map[string]interface{}, len(configurationKey))
	for i, key := range configurationKey {
		keys[i] = map[string]interface{}{
			"key":      key.Key,
			"value":    key.Value,
			"readonly": key.Readonly,
		}
	}

	payload := map[string]interface{}{
		"configurationKey": keys,
	}

	if len(unknownKey) > 0 {
		payload["unknownKey"] = unknownKey
	}

	return CreateResultMessage(messageID, payload, 0)
}

// CreateChangeConfigurationResult creates a ChangeConfiguration.conf response
func CreateChangeConfigurationResult(messageID string, status entities.ConfigurationStatus) entities.Message {
	payload := map[string]interface{}{
		"status": string(status),
	}
	return CreateResultMessage(messageID, payload, 0)
}
