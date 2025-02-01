package factories

import (
	"EV-Client-Simulator/domain/entities"
	"time"
)

func CreateMeterValuesCall(connectorId int, transactionId int, value float64, soc int16) entities.Message {
	energyValue := map[string]any{
		"value":     value,
		"measurand": "Energy.Active.Import.Register",
	}
	socValue := map[string]any{
		"value":     soc,
		"measurand": "SoC",
	}
	meterValue := map[string]any{
		"timestamp":    time.Now().Unix(),
		"sampledValue": []map[string]any{energyValue, socValue},
	}
	return CreateCallMessage("MeterValues", map[string]any{
		"transactionId": transactionId,
		"meterValue":    []map[string]any{meterValue},
	}, connectorId)
}
