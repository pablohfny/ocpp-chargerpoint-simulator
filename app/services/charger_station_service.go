package services

import (
	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/domain/factories"
	"fmt"
	"sync"
	"time"
)

type ChargerStationService struct {
	station        *entities.ChargerStation
	messageChannel chan entities.Message
	errorsChannel  chan error
	recentMessages map[string]entities.Message
	mu             *sync.Mutex
}

func NewChargerStationSerice(station *entities.ChargerStation, messageChannel chan entities.Message, errorsChannel chan error) *ChargerStationService {
	return &ChargerStationService{
		station:        station,
		messageChannel: messageChannel,
		errorsChannel:  errorsChannel,
		mu:             &sync.Mutex{},
		recentMessages: make(map[string]entities.Message),
	}
}

func (service *ChargerStationService) ProcessCall(call entities.Message) {
	fmt.Printf("Received call: %v\n", call)

	switch call.Action {
	case "RemoteStartTransaction":
		service.processStartRemoteTransactionCall(call)
	case "RemoteStopTransaction":
		service.processStopRemoteTransactionCall(call)
	default:
		fmt.Printf("Call not processed: %v\n", call)
	}
}

func (service *ChargerStationService) ProcessResult(result entities.Message) {
	fmt.Printf("Received result: %v\n", result)
	var call entities.Message
	var ok bool

	if call, ok = service.recentMessages[result.ID]; !ok {
		return
	}

	delete(service.recentMessages, result.ID)

	switch call.Action {
	case "Authorize":
		service.processAuthorizeResult(call, result)
	case "StartTransaction":
		service.processStartTransactionResult(call, result)
	default:
		break
	}
}

func (service *ChargerStationService) processStartRemoteTransactionCall(message entities.Message) {
	connectorId := int(message.Payload["connectorId"].(float64))
	idTag := message.Payload["idTag"].(string)

	point := service.station.GetPoint(connectorId)

	if err := point.StartRemoteTransaction(); err != nil {
		service.sendMessage(factories.CreateRemoteStartTransactionResult(connectorId, message.ID, map[string]interface{}{"status": "Rejected"}))
		time.Sleep(1 * time.Second)
		point.SetStatus("AVAILABLE")
		service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.Status))
		service.errorsChannel <- err
		return
	}

	if err := point.Authorize(idTag); err != nil {
		service.errorsChannel <- err
		return
	}

	// point.SetStatus("PREPARING")
	// service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.Status))
	// time.Sleep(3 * time.Second)
	// service.sendMessage(factories.CreateRemoteStartTransactionResult(connectorId, message.ID, map[string]interface{}{"status": "Rejected"}))
	// time.Sleep(5 * time.Second)
	// point.SetStatus("AVAILABLE")

	service.sendMessage(factories.CreateRemoteStartTransactionResult(connectorId, message.ID, map[string]interface{}{"status": "Accepted"}))
	service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.Status))
	service.sendMessage(factories.CreateAuthorizationCall(connectorId, idTag))
}

func (service *ChargerStationService) processAuthorizeResult(call entities.Message, result entities.Message) {
	idTagInfo := result.Payload["idTagInfo"].(map[string]interface{})
	point := service.station.GetPoint(call.ConnectorId)

	if idTagInfo["status"] != "Accepted" {
		point.RemoveCurrentTransaction()
		point.SetStatus("FINISHING")
		service.sendMessage(factories.CreateStatusNotificationCall(call.ConnectorId, point.Status))
	} else {
		service.sendMessage(factories.CreateStartTransactionCall(call.ConnectorId, call.Payload["idTag"].(string)))
	}
}

func (service *ChargerStationService) processStartTransactionResult(call entities.Message, result entities.Message) {
	idTagInfo := result.Payload["idTagInfo"].(map[string]interface{})
	point := service.station.GetPoint(call.ConnectorId)

	if idTagInfo["status"] != "Accepted" {
		point.RemoveCurrentTransaction()
		point.SetStatus("FINISHING")
		service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.Status))
		time.Sleep(5 * time.Second)
		point.SetStatus("AVAILABLE")
	} else {
		transactionId := int(result.Payload["transactionId"].(float64))
		err := point.StartTransaction(transactionId)

		if err != nil {
			service.errorsChannel <- err
			point.RemoveCurrentTransaction()
			point.SetStatus("FINISHING")
		}

		go func() {
			if point.Status == "CHARGING" {
				service.sendMessage(factories.CreateMeterValuesCall(point.ID, transactionId, point.MeterValue, point.Soc))
			}
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if point.Status == "CHARGING" {
					service.sendMessage(factories.CreateMeterValuesCall(point.ID, transactionId, point.MeterValue, point.Soc))
				} else if point.Status == "FINISHING" && point.CurrentTransaction == transactionId {
					service.sendMessage(factories.CreateMeterValuesCall(point.ID, transactionId, point.MeterValue, point.Soc))
					service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.Status))
					time.Sleep(3 * time.Second)
					service.sendMessage(factories.CreateStopTransactionCall(point.ID, transactionId, point.MeterValue))
					time.Sleep(5 * time.Second)
					point.SetStatus("AVAILABLE")
					service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.Status))
					point.RemoveCurrentTransaction()
					return
				} else {
					service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.Status))
					return
				}
			}
		}()
	}

	service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.Status))
}

func (service *ChargerStationService) processStopRemoteTransactionCall(message entities.Message) {
	transactionId := int(message.Payload["transactionId"].(float64))
	point := service.station.GetPointByTransaction(transactionId)
	service.sendMessage(factories.CreateRemoteStopTransactionResult(point.ID, message.ID, map[string]interface{}{"status": "Accepted"}))

	if err := point.StopTransaction(); err != nil {
		service.errorsChannel <- err
		return
	}
}

func (service *ChargerStationService) InitHeartbeat(interval time.Duration) {
	service.sendMessage(factories.CreateHeartbeatCall(make(map[string]interface{})))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			service.sendMessage(factories.CreateHeartbeatCall(make(map[string]interface{})))
		}
	}()
}

func (service *ChargerStationService) NotifyStatuses() {
	for _, point := range service.station.ChargerPoints {
		service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.Status))
	}
}

func (service *ChargerStationService) NotifyBoot() {
	service.sendMessage(factories.CreateBootNotificationCall(make(map[string]interface{})))
}

func (service *ChargerStationService) sendMessage(message entities.Message) {
	service.recentMessages[message.ID] = message
	service.messageChannel <- message
}
