package services

import (
	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/domain/factories"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type ChargerStationService struct {
	station           *entities.ChargerStation
	messageChannel    chan entities.Message
	errorsChannel     chan error
	recentMessages    map[string]entities.Message
	mu                *sync.Mutex
	configService     *ConfigurationService
	simulationService *SimulationService
}

func NewChargerStationSerice(station *entities.ChargerStation, messageChannel chan entities.Message, errorsChannel chan error) *ChargerStationService {
	return &ChargerStationService{
		station:           station,
		messageChannel:    messageChannel,
		errorsChannel:     errorsChannel,
		mu:                &sync.Mutex{},
		recentMessages:    make(map[string]entities.Message),
		configService:     NewConfigurationService(),
		simulationService: NewSimulationService(),
	}
}

// GetConfigService returns the configuration service
func (service *ChargerStationService) GetConfigService() *ConfigurationService {
	return service.configService
}

// GetSimulationService returns the simulation service
func (service *ChargerStationService) GetSimulationService() *SimulationService {
	return service.simulationService
}

// GetStation returns the charger station
func (service *ChargerStationService) GetStation() *entities.ChargerStation {
	return service.station
}

// getMeterValueInterval returns the configured meter value sample interval
func (service *ChargerStationService) getMeterValueInterval() time.Duration {
	if key, exists := service.configService.GetKey("MeterValueSampleInterval"); exists {
		if interval, err := strconv.Atoi(key.Value); err == nil && interval > 0 {
			return time.Duration(interval) * time.Second
		}
	}
	return 5 * time.Second // default
}

// getHeartbeatInterval returns the configured heartbeat interval
func (service *ChargerStationService) getHeartbeatInterval() time.Duration {
	if key, exists := service.configService.GetKey("HeartbeatInterval"); exists {
		if interval, err := strconv.Atoi(key.Value); err == nil && interval > 0 {
			return time.Duration(interval) * time.Second
		}
	}
	return 30 * time.Second // default
}

func (service *ChargerStationService) ProcessCall(call entities.Message) {
	fmt.Printf("Received call: %v\n", call)

	switch call.Action {
	case "RemoteStartTransaction":
		service.processStartRemoteTransactionCall(call)
	case "RemoteStopTransaction":
		service.processStopRemoteTransactionCall(call)
	case "GetConfiguration":
		service.processGetConfigurationCall(call)
	case "ChangeConfiguration":
		service.processChangeConfigurationCall(call)
	case "Reset":
		service.processResetCall(call)
	case "UnlockConnector":
		service.processUnlockConnectorCall(call)
	case "DataTransfer":
		service.processDataTransferCall(call)
	case "ClearCache":
		service.processClearCacheCall(call)
	case "TriggerMessage":
		service.processTriggerMessageCall(call)
	case "ReserveNow":
		service.processReserveNowCall(call)
	case "CancelReservation":
		service.processCancelReservationCall(call)
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
	service.simulationService.ApplyDelay("RemoteStart")

	connectorId := int(message.Payload["connectorId"].(float64))
	idTag := message.Payload["idTag"].(string)

	point := service.station.GetPoint(connectorId)

	if err := point.StartRemoteTransaction(); err != nil {
		service.sendMessage(factories.CreateRemoteStartTransactionResult(connectorId, message.ID, map[string]interface{}{"status": "Rejected"}))
		time.Sleep(1 * time.Second)
		point.SetStatus(entities.StatusAvailable)
		service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.GetStatusString()))
		service.errorsChannel <- err
		return
	}

	if err := point.Authorize(idTag); err != nil {
		service.errorsChannel <- err
		return
	}

	service.sendMessage(factories.CreateRemoteStartTransactionResult(connectorId, message.ID, map[string]interface{}{"status": "Accepted"}))
	service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.GetStatusString()))
	service.sendMessage(factories.CreateAuthorizationCall(connectorId, idTag))
}

func (service *ChargerStationService) processAuthorizeResult(call entities.Message, result entities.Message) {
	idTagInfo := result.Payload["idTagInfo"].(map[string]interface{})
	point := service.station.GetPoint(call.ConnectorId)

	if idTagInfo["status"] != "Accepted" {
		point.RemoveCurrentTransaction()
		point.SetStatus(entities.StatusFinishing)
		service.sendMessage(factories.CreateStatusNotificationCall(call.ConnectorId, point.GetStatusString()))
	} else {
		service.sendMessage(factories.CreateStartTransactionCall(call.ConnectorId, call.Payload["idTag"].(string)))
	}
}

func (service *ChargerStationService) processStartTransactionResult(call entities.Message, result entities.Message) {
	idTagInfo := result.Payload["idTagInfo"].(map[string]interface{})
	point := service.station.GetPoint(call.ConnectorId)

	if idTagInfo["status"] != "Accepted" {
		point.RemoveCurrentTransaction()
		point.SetStatus(entities.StatusFinishing)
		service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
		time.Sleep(5 * time.Second)
		point.SetStatus(entities.StatusAvailable)
	} else {
		transactionId := int(result.Payload["transactionId"].(float64))
		txSettings := service.simulationService.GetTransactionSettings()

		// Set transaction ID before preparing delay
		if err := point.SetCurrentTransaction(transactionId); err != nil {
			service.errorsChannel <- err
			point.SetStatus(entities.StatusFinishing)
			service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
			return
		}

		if service.simulationService.IsManualMode() {
			fmt.Printf("[MANUAL] Transaction %d accepted on connector %d — waiting for manual control\n", transactionId, point.ID)
		} else {
			go func() {
				// Stay in Preparing for configured duration before transitioning to Charging
				if txSettings.PreparingDurationSec > 0 {
					fmt.Printf("Connector %d: Preparing for %d seconds before Charging\n", point.ID, txSettings.PreparingDurationSec)
					time.Sleep(time.Duration(txSettings.PreparingDurationSec) * time.Second)
				}

				err := point.StartChargingWithSettings(txSettings)
				if err != nil {
					service.errorsChannel <- err
					point.RemoveCurrentTransaction()
					point.SetStatus(entities.StatusFinishing)
					service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
					return
				}

				service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))

				if point.Status == entities.StatusCharging {
					service.sendMessage(factories.CreateMeterValuesCall(point.ID, transactionId, point.MeterValue, point.Soc))
				}
				meterInterval := service.getMeterValueInterval()
				fmt.Printf("MeterValues interval: %v\n", meterInterval)
				ticker := time.NewTicker(meterInterval)
				defer ticker.Stop()
				for range ticker.C {
					if point.Status == entities.StatusCharging {
						service.sendMessage(factories.CreateMeterValuesCall(point.ID, transactionId, point.MeterValue, point.Soc))
					} else if point.Status == entities.StatusSuspendedEV || point.Status == entities.StatusSuspendedEVSE {
						// Keep loop alive but skip sending MeterValues while suspended
						continue
					} else if point.Status == entities.StatusFinishing && point.CurrentTransaction == transactionId {
						service.sendMessage(factories.CreateMeterValuesCall(point.ID, transactionId, point.MeterValue, point.Soc))
						service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
						time.Sleep(3 * time.Second)
						service.sendMessage(factories.CreateStopTransactionCall(point.ID, transactionId, point.MeterValue))
						time.Sleep(5 * time.Second)
						point.SetStatus(entities.StatusAvailable)
						service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
						point.RemoveCurrentTransaction()
						return
					} else {
						service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
						return
					}
				}
			}()
		}
	}

	service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
}

func (service *ChargerStationService) processStopRemoteTransactionCall(message entities.Message) {
	// A CSMS may ask us to stop a transaction we know nothing about (stale id,
	// restarted simulator, a charge that never actually started). OCPP 1.6 says
	// answer Rejected; before this guard both the type assertion and the nil
	// point crashed the whole process with a SIGSEGV, taking every subsequent
	// test down with it.
	rawTransactionId, ok := message.Payload["transactionId"].(float64)
	if !ok {
		service.sendMessage(factories.CreateRemoteStopTransactionResult(0, message.ID, map[string]interface{}{"status": "Rejected"}))
		return
	}

	transactionId := int(rawTransactionId)
	point := service.station.GetPointByTransaction(transactionId)
	if point == nil {
		service.sendMessage(factories.CreateRemoteStopTransactionResult(0, message.ID, map[string]interface{}{"status": "Rejected"}))
		return
	}

	service.sendMessage(factories.CreateRemoteStopTransactionResult(point.ID, message.ID, map[string]interface{}{"status": "Accepted"}))

	if err := point.StopTransaction(); err != nil {
		service.errorsChannel <- err
		return
	}
}

func (service *ChargerStationService) InitHeartbeat(interval time.Duration) {
	// Use configured interval if available, otherwise use provided default
	heartbeatInterval := service.getHeartbeatInterval()
	fmt.Printf("Heartbeat interval: %v\n", heartbeatInterval)

	service.sendMessage(factories.CreateHeartbeatCall(make(map[string]interface{})))

	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		for range ticker.C {
			service.sendMessage(factories.CreateHeartbeatCall(make(map[string]interface{})))
		}
	}()
}

func (service *ChargerStationService) NotifyStatuses() {
	for _, point := range service.station.ChargerPoints {
		service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
	}
}

func (service *ChargerStationService) NotifyBoot() {
	service.sendMessage(factories.CreateBootNotificationCall(make(map[string]interface{})))
}

func (service *ChargerStationService) sendMessage(message entities.Message) {
	service.recentMessages[message.ID] = message
	service.messageChannel <- message
}

// processGetConfigurationCall handles GetConfiguration requests
func (service *ChargerStationService) processGetConfigurationCall(message entities.Message) {
	var keys []string
	if keyList, ok := message.Payload["key"].([]interface{}); ok {
		for _, k := range keyList {
			if s, ok := k.(string); ok {
				keys = append(keys, s)
			}
		}
	}

	configKeys, unknownKeys := service.configService.GetConfiguration(keys)
	service.sendMessage(factories.CreateGetConfigurationResult(message.ID, configKeys, unknownKeys))
}

// processChangeConfigurationCall handles ChangeConfiguration requests
func (service *ChargerStationService) processChangeConfigurationCall(message entities.Message) {
	key := message.Payload["key"].(string)
	value := message.Payload["value"].(string)

	status := service.configService.ChangeConfiguration(key, value)
	service.sendMessage(factories.CreateChangeConfigurationResult(message.ID, status))

	// Log the configuration change
	if status == entities.ConfigurationStatusAccepted {
		fmt.Printf("Configuration changed: %s = %s\n", key, value)
	}
}

// processResetCall handles Reset requests
func (service *ChargerStationService) processResetCall(message entities.Message) {
	resetType := entities.ResetType(message.Payload["type"].(string))

	// Check if any transactions are active for soft reset
	if resetType == entities.ResetTypeSoft {
		for _, point := range service.station.ChargerPoints {
			if point.CurrentTransaction != 0 {
				// Wait for transactions to complete before resetting
				go func() {
					// Give time for transactions to finish
					time.Sleep(30 * time.Second)
					service.station.Reset(resetType)
					service.NotifyBoot()
					service.NotifyStatuses()
				}()
				service.sendMessage(factories.CreateResetResult(message.ID, entities.ResetStatusAccepted))
				return
			}
		}
	}

	service.station.Reset(resetType)
	service.sendMessage(factories.CreateResetResult(message.ID, entities.ResetStatusAccepted))

	// After reset, send boot notification and status
	go func() {
		time.Sleep(1 * time.Second)
		service.NotifyBoot()
		service.NotifyStatuses()
	}()
}

// processUnlockConnectorCall handles UnlockConnector requests
func (service *ChargerStationService) processUnlockConnectorCall(message entities.Message) {
	connectorId := int(message.Payload["connectorId"].(float64))
	point := service.station.GetPoint(connectorId)

	if point == nil {
		service.sendMessage(factories.CreateUnlockConnectorResult(message.ID, entities.UnlockStatusUnlockFailed))
		return
	}

	// Simulate unlocking - if charging, stop first
	if point.Status == entities.StatusCharging {
		point.StopTransaction()
	}

	point.UnplugCable()
	service.sendMessage(factories.CreateUnlockConnectorResult(message.ID, entities.UnlockStatusUnlocked))
	service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.GetStatusString()))
}

// processDataTransferCall handles DataTransfer requests
func (service *ChargerStationService) processDataTransferCall(message entities.Message) {
	vendorId := message.Payload["vendorId"].(string)
	var msgId, data string
	if m, ok := message.Payload["messageId"].(string); ok {
		msgId = m
	}
	if d, ok := message.Payload["data"].(string); ok {
		data = d
	}

	fmt.Printf("DataTransfer received - vendorId: %s, messageId: %s, data: %s\n", vendorId, msgId, data)

	// Accept all vendor-specific data transfers
	service.sendMessage(factories.CreateDataTransferResult(message.ID, entities.DataTransferStatusAccepted, ""))
}

// processClearCacheCall handles ClearCache requests
func (service *ChargerStationService) processClearCacheCall(message entities.Message) {
	// Simulator doesn't have a real cache, always accept
	service.sendMessage(factories.CreateClearCacheResult(message.ID, entities.ClearCacheStatusAccepted))
}

// processTriggerMessageCall handles TriggerMessage requests
func (service *ChargerStationService) processTriggerMessageCall(message entities.Message) {
	requestedMessage := entities.MessageTrigger(message.Payload["requestedMessage"].(string))
	var connectorId int
	if cid, ok := message.Payload["connectorId"].(float64); ok {
		connectorId = int(cid)
	}

	switch requestedMessage {
	case entities.TriggerBootNotification:
		service.sendMessage(factories.CreateTriggerMessageResult(message.ID, entities.TriggerMessageStatusAccepted))
		go service.NotifyBoot()

	case entities.TriggerHeartbeat:
		service.sendMessage(factories.CreateTriggerMessageResult(message.ID, entities.TriggerMessageStatusAccepted))
		go service.sendMessage(factories.CreateHeartbeatCall(make(map[string]interface{})))

	case entities.TriggerStatusNotification:
		service.sendMessage(factories.CreateTriggerMessageResult(message.ID, entities.TriggerMessageStatusAccepted))
		if connectorId > 0 {
			point := service.station.GetPoint(connectorId)
			if point != nil {
				go service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.GetStatusString()))
			}
		} else {
			go service.NotifyStatuses()
		}

	case entities.TriggerMeterValues:
		if connectorId > 0 {
			point := service.station.GetPoint(connectorId)
			if point != nil && point.CurrentTransaction != 0 {
				service.sendMessage(factories.CreateTriggerMessageResult(message.ID, entities.TriggerMessageStatusAccepted))
				go service.sendMessage(factories.CreateMeterValuesCall(connectorId, point.CurrentTransaction, point.MeterValue, point.Soc))
			} else {
				service.sendMessage(factories.CreateTriggerMessageResult(message.ID, entities.TriggerMessageStatusRejected))
			}
		} else {
			service.sendMessage(factories.CreateTriggerMessageResult(message.ID, entities.TriggerMessageStatusRejected))
		}

	default:
		service.sendMessage(factories.CreateTriggerMessageResult(message.ID, entities.TriggerMessageStatusNotImplemented))
	}
}

// processReserveNowCall handles ReserveNow requests
func (service *ChargerStationService) processReserveNowCall(message entities.Message) {
	connectorId := int(message.Payload["connectorId"].(float64))
	expiryDateStr := message.Payload["expiryDate"].(string)
	idTag := message.Payload["idTag"].(string)
	reservationId := int(message.Payload["reservationId"].(float64))

	expiryDate, err := time.Parse(time.RFC3339, expiryDateStr)
	if err != nil {
		service.sendMessage(factories.CreateReserveNowResult(message.ID, entities.ReservationStatusRejected))
		return
	}

	point := service.station.GetPoint(connectorId)
	if point == nil {
		service.sendMessage(factories.CreateReserveNowResult(message.ID, entities.ReservationStatusRejected))
		return
	}

	// Check if connector is available
	if point.Status == entities.StatusFaulted {
		service.sendMessage(factories.CreateReserveNowResult(message.ID, entities.ReservationStatusFaulted))
		return
	}

	if point.Status == entities.StatusUnavailable {
		service.sendMessage(factories.CreateReserveNowResult(message.ID, entities.ReservationStatusUnavailable))
		return
	}

	if point.Status != entities.StatusAvailable {
		service.sendMessage(factories.CreateReserveNowResult(message.ID, entities.ReservationStatusOccupied))
		return
	}

	err = point.SetReservation(reservationId, idTag, expiryDate)
	if err != nil {
		service.sendMessage(factories.CreateReserveNowResult(message.ID, entities.ReservationStatusRejected))
		return
	}

	service.sendMessage(factories.CreateReserveNowResult(message.ID, entities.ReservationStatusAccepted))
	service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.GetStatusString()))
}

// processCancelReservationCall handles CancelReservation requests
func (service *ChargerStationService) processCancelReservationCall(message entities.Message) {
	reservationId := int(message.Payload["reservationId"].(float64))

	point := service.station.GetPointByReservation(reservationId)
	if point == nil {
		service.sendMessage(factories.CreateCancelReservationResult(message.ID, entities.CancelReservationStatusRejected))
		return
	}

	err := point.CancelReservation()
	if err != nil {
		service.sendMessage(factories.CreateCancelReservationResult(message.ID, entities.CancelReservationStatusRejected))
		return
	}

	service.sendMessage(factories.CreateCancelReservationResult(message.ID, entities.CancelReservationStatusAccepted))
	service.sendMessage(factories.CreateStatusNotificationCall(point.ID, point.GetStatusString()))
}

// TriggerStatusNotification triggers a status notification for a connector
func (service *ChargerStationService) TriggerStatusNotification(connectorId int) {
	point := service.station.GetPoint(connectorId)
	if point != nil {
		service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.GetStatusString()))
	}
}

// TriggerMeterValues triggers meter values for a connector
func (service *ChargerStationService) TriggerMeterValues(connectorId int) {
	point := service.station.GetPoint(connectorId)
	if point != nil && point.CurrentTransaction != 0 {
		service.sendMessage(factories.CreateMeterValuesCall(connectorId, point.CurrentTransaction, point.MeterValue, point.Soc))
	}
}

// TriggerHeartbeat triggers a heartbeat
func (service *ChargerStationService) TriggerHeartbeat() {
	service.sendMessage(factories.CreateHeartbeatCall(make(map[string]interface{})))
}

// TriggerBootNotification triggers a boot notification
func (service *ChargerStationService) TriggerBootNotification() {
	service.NotifyBoot()
}

// SendAuthorize sends a standalone Authorize Call with the given idTag.
// Used to manually drive the auto-charge flow (charger authorizes the tag).
func (service *ChargerStationService) SendAuthorize(connectorId int, idTag string) {
	service.sendMessage(factories.CreateAuthorizationCall(connectorId, idTag))
}

// ManualSetStatus sets the connector status and sends StatusNotification.
// In manual mode, if transitioning to Charging, starts meter increment and meter queuing.
func (service *ChargerStationService) ManualSetStatus(connectorId int, status string) error {
	point := service.station.GetPoint(connectorId)
	if point == nil {
		return fmt.Errorf("connector %d not found", connectorId)
	}

	previousStatus := point.GetStatusString()

	if status == "Charging" && point.Status == entities.StatusPreparing {
		txSettings := service.simulationService.GetTransactionSettings()
		if err := point.StartChargingWithSettings(txSettings); err != nil {
			return err
		}
		service.startMeterQueuing(point)
	} else if status == "Finishing" {
		if err := point.StopTransaction(); err != nil {
			return err
		}
		point.StopMeterQueue()
		// Enqueue final meter value
		point.EnqueueMeter()
	} else if status == "Available" {
		point.StopMeterQueue()
		point.SetStatus(entities.StatusAvailable)
	} else {
		if err := point.SetStatusString(status); err != nil {
			return err
		}
	}

	fmt.Printf("[MANUAL] Connector %d: %s → %s\n", connectorId, previousStatus, point.GetStatusString())
	service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.GetStatusString()))
	return nil
}

// startMeterQueuing starts a goroutine that enqueues meter value snapshots at the configured interval
func (service *ChargerStationService) startMeterQueuing(point *entities.ChargerPoint) {
	meterInterval := service.getMeterValueInterval()
	stopCh := point.StartMeterQueuing()

	go func() {
		ticker := time.NewTicker(meterInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if point.Status == entities.StatusCharging {
					point.EnqueueMeter()
					fmt.Printf("[MANUAL] Queued meter value for connector %d: %.2f Wh, %d%% SoC (queue size: %d)\n",
						point.ID, point.MeterValue, point.Soc, len(point.GetMeterQueueSnapshot()))
				}
			case <-stopCh:
				return
			}
		}
	}()
}

// ManualSendNextMeter pops the next meter value from queue and sends it.
// If overrideWh or overrideSoc are non-nil, those values are used instead.
func (service *ChargerStationService) ManualSendNextMeter(connectorId int, overrideWh *float64, overrideSoc *int16) (*entities.MeterQueueEntry, error) {
	point := service.station.GetPoint(connectorId)
	if point == nil {
		return nil, fmt.Errorf("connector %d not found", connectorId)
	}
	if point.CurrentTransaction == 0 {
		return nil, fmt.Errorf("no active transaction on connector %d", connectorId)
	}

	entry := point.DequeueMeter()

	var meterWh float64
	var soc int16

	if entry != nil {
		meterWh = entry.MeterValue
		soc = entry.Soc
	} else {
		// Queue empty — use current point values
		meterWh = point.MeterValue
		soc = point.Soc
	}

	// Apply overrides
	if overrideWh != nil {
		meterWh = *overrideWh
	}
	if overrideSoc != nil {
		soc = *overrideSoc
	}

	service.sendMessage(factories.CreateMeterValuesCall(connectorId, point.CurrentTransaction, meterWh, soc))

	sent := &entities.MeterQueueEntry{MeterValue: meterWh, Soc: soc, Timestamp: time.Now()}
	fmt.Printf("[MANUAL] Sent meter value for connector %d: %.2f Wh, %d%% SoC\n", connectorId, meterWh, soc)
	return sent, nil
}

// ManualSendStopTransaction sends a StopTransaction for the active transaction
func (service *ChargerStationService) ManualSendStopTransaction(connectorId int) error {
	point := service.station.GetPoint(connectorId)
	if point == nil {
		return fmt.Errorf("connector %d not found", connectorId)
	}
	if point.CurrentTransaction == 0 {
		return fmt.Errorf("no active transaction on connector %d", connectorId)
	}

	service.sendMessage(factories.CreateStopTransactionCall(connectorId, point.CurrentTransaction, point.MeterValue))
	fmt.Printf("[MANUAL] Sent StopTransaction for connector %d, transaction %d\n", connectorId, point.CurrentTransaction)
	return nil
}

// SendStartTransaction sends a StartTransaction Call with the given idTag and
// meter start, with the simulator initiating the flow (auto-charge scenario).
//
// It mirrors the lead-up of the remote-start flow so the StartTransaction.conf
// flows straight into charging: the connector is moved to Preparing and the
// idTag is associated before the Call is sent. When the server accepts the
// transaction, the existing processStartTransactionResult handler transitions
// Preparing -> Charging and begins meter values (same path as remote start).
func (service *ChargerStationService) SendStartTransaction(connectorId int, idTag string, meterStart float64) {
	point := service.station.GetPoint(connectorId)
	if point == nil {
		service.errorsChannel <- fmt.Errorf("connector %d not found", connectorId)
		return
	}

	// Put the connector into Preparing + authorize the tag, just like the
	// remote-start flow does before sending StartTransaction.
	if point.Status == entities.StatusAvailable || point.Status == entities.StatusReserved {
		point.SetStatus(entities.StatusPreparing)
	}
	if err := point.Authorize(idTag); err != nil {
		service.errorsChannel <- err
		return
	}
	service.sendMessage(factories.CreateStatusNotificationCall(connectorId, point.GetStatusString()))

	service.sendMessage(factories.CreateCallMessage("StartTransaction", map[string]interface{}{
		"idTag":       idTag,
		"connectorId": connectorId,
		"meterStart":  meterStart,
	}, connectorId))
}
