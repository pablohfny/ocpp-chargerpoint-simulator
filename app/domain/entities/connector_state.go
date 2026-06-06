package entities

// ConnectorStatus represents OCPP 1.6 ChargePointStatus values
type ConnectorStatus string

const (
	StatusAvailable     ConnectorStatus = "Available"
	StatusPreparing     ConnectorStatus = "Preparing"
	StatusCharging      ConnectorStatus = "Charging"
	StatusSuspendedEV   ConnectorStatus = "SuspendedEV"
	StatusSuspendedEVSE ConnectorStatus = "SuspendedEVSE"
	StatusFinishing     ConnectorStatus = "Finishing"
	StatusReserved      ConnectorStatus = "Reserved"
	StatusUnavailable   ConnectorStatus = "Unavailable"
	StatusFaulted       ConnectorStatus = "Faulted"
)

// ChargePointErrorCode represents OCPP 1.6 error codes
type ChargePointErrorCode string

const (
	ErrorNoError               ChargePointErrorCode = "NoError"
	ErrorConnectorLockFailure  ChargePointErrorCode = "ConnectorLockFailure"
	ErrorEVCommunicationError  ChargePointErrorCode = "EVCommunicationError"
	ErrorGroundFailure         ChargePointErrorCode = "GroundFailure"
	ErrorHighTemperature       ChargePointErrorCode = "HighTemperature"
	ErrorInternalError         ChargePointErrorCode = "InternalError"
	ErrorLocalListConflict     ChargePointErrorCode = "LocalListConflict"
	ErrorOtherError            ChargePointErrorCode = "OtherError"
	ErrorOverCurrentFailure    ChargePointErrorCode = "OverCurrentFailure"
	ErrorOverVoltage           ChargePointErrorCode = "OverVoltage"
	ErrorPowerMeterFailure     ChargePointErrorCode = "PowerMeterFailure"
	ErrorPowerSwitchFailure    ChargePointErrorCode = "PowerSwitchFailure"
	ErrorReaderFailure         ChargePointErrorCode = "ReaderFailure"
	ErrorResetFailure          ChargePointErrorCode = "ResetFailure"
	ErrorUnderVoltage          ChargePointErrorCode = "UnderVoltage"
	ErrorWeakSignal            ChargePointErrorCode = "WeakSignal"
)

// ResetType represents OCPP 1.6 Reset types
type ResetType string

const (
	ResetTypeSoft ResetType = "Soft"
	ResetTypeHard ResetType = "Hard"
)

// ResetStatus represents OCPP 1.6 Reset response status
type ResetStatus string

const (
	ResetStatusAccepted ResetStatus = "Accepted"
	ResetStatusRejected ResetStatus = "Rejected"
)

// UnlockStatus represents OCPP 1.6 UnlockConnector response status
type UnlockStatus string

const (
	UnlockStatusUnlocked     UnlockStatus = "Unlocked"
	UnlockStatusUnlockFailed UnlockStatus = "UnlockFailed"
	UnlockStatusNotSupported UnlockStatus = "NotSupported"
)

// DataTransferStatus represents OCPP 1.6 DataTransfer response status
type DataTransferStatus string

const (
	DataTransferStatusAccepted         DataTransferStatus = "Accepted"
	DataTransferStatusRejected         DataTransferStatus = "Rejected"
	DataTransferStatusUnknownMessageId DataTransferStatus = "UnknownMessageId"
	DataTransferStatusUnknownVendorId  DataTransferStatus = "UnknownVendorId"
)

// ClearCacheStatus represents OCPP 1.6 ClearCache response status
type ClearCacheStatus string

const (
	ClearCacheStatusAccepted ClearCacheStatus = "Accepted"
	ClearCacheStatusRejected ClearCacheStatus = "Rejected"
)

// TriggerMessageStatus represents OCPP 1.6 TriggerMessage response status
type TriggerMessageStatus string

const (
	TriggerMessageStatusAccepted       TriggerMessageStatus = "Accepted"
	TriggerMessageStatusRejected       TriggerMessageStatus = "Rejected"
	TriggerMessageStatusNotImplemented TriggerMessageStatus = "NotImplemented"
)

// ReservationStatus represents OCPP 1.6 ReserveNow response status
type ReservationStatus string

const (
	ReservationStatusAccepted    ReservationStatus = "Accepted"
	ReservationStatusFaulted     ReservationStatus = "Faulted"
	ReservationStatusOccupied    ReservationStatus = "Occupied"
	ReservationStatusRejected    ReservationStatus = "Rejected"
	ReservationStatusUnavailable ReservationStatus = "Unavailable"
)

// CancelReservationStatus represents OCPP 1.6 CancelReservation response status
type CancelReservationStatus string

const (
	CancelReservationStatusAccepted CancelReservationStatus = "Accepted"
	CancelReservationStatusRejected CancelReservationStatus = "Rejected"
)

// ConfigurationStatus represents OCPP 1.6 ChangeConfiguration response status
type ConfigurationStatus string

const (
	ConfigurationStatusAccepted       ConfigurationStatus = "Accepted"
	ConfigurationStatusRejected       ConfigurationStatus = "Rejected"
	ConfigurationStatusRebootRequired ConfigurationStatus = "RebootRequired"
	ConfigurationStatusNotSupported   ConfigurationStatus = "NotSupported"
)

// MessageTrigger represents OCPP 1.6 TriggerMessage types
type MessageTrigger string

const (
	TriggerBootNotification        MessageTrigger = "BootNotification"
	TriggerDiagnosticsStatusNotif  MessageTrigger = "DiagnosticsStatusNotification"
	TriggerFirmwareStatusNotif     MessageTrigger = "FirmwareStatusNotification"
	TriggerHeartbeat               MessageTrigger = "Heartbeat"
	TriggerMeterValues             MessageTrigger = "MeterValues"
	TriggerStatusNotification      MessageTrigger = "StatusNotification"
)
