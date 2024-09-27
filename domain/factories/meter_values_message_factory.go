package factories

import (
	"EV-Client-Simulator/domain/entities"
	"time"
)

func CreateMeterValuesCall(connectorId int, transactionId int, value float64) entities.Message {
	energyValue := map[string]any{
		"value":     value,
		"measurand": "Energy.Active.Import.Register",
	}
	meterValue := map[string]any{
		"timestamp":    time.Now().Unix(),
		"sampledValue": []map[string]any{energyValue},
	}
	return CreateCallMessage("MeterValues", map[string]any{
		"transactionId": transactionId,
		"meterValue":    []map[string]any{meterValue},
	}, connectorId)
}
