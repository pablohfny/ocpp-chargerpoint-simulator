package services

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"EV-Client-Simulator/app/domain/entities"
)

// ErrPipelineNoContext is returned when an action runs before a context is armed.
var ErrPipelineNoContext = errors.New("pipeline context not set")

// ErrActionNotAvailable is returned when an action is not valid for the stage.
var ErrActionNotAvailable = errors.New("action not available in the current stage")

// pipelineLocalIdTag is the idTag the local OCPP start uses.
const pipelineLocalIdTag = "PIPELINE_LOCAL"

// pipelineInvalidToken is the deliberately wrong credential the "token
// inválido" variant presents, so the platform answers 401.
const pipelineInvalidToken = "invalid-token-on-purpose"

// PipelineCharger is the connector summary the cockpit shows next to the hops.
type PipelineCharger struct {
	ConnectorID    int    `json:"connectorId"`
	Status         string `json:"status"`
	CablePlugged   bool   `json:"cablePlugged"`
	TransactionID  int    `json:"transactionId,omitempty"`
	BatteryPercent int    `json:"batteryPercent"`
}

// PipelineState is everything the cockpit polls: where the run is, what the
// hops look like and which actions are valid right now.
type PipelineState struct {
	Context    entities.PipelineContext  `json:"context"`
	Stage      entities.PipelineStage    `json:"stage"`
	StageLabel string                    `json:"stageLabel"`
	StageHint  string                    `json:"stageHint"`
	Hops       []entities.PipelineHop    `json:"hops"`
	Actions    []entities.PipelineAction `json:"actions"`
	Run        entities.PipelineRun      `json:"run"`
	Charger    *PipelineCharger          `json:"charger,omitempty"`
}

// PipelineService is the end-to-end test orchestrator. It owns the stage state
// machine and drives the existing senders — the OCPI command service and the
// charger station service — instead of reimplementing what they do.
type PipelineService struct {
	mu       sync.Mutex
	station  *ChargerStationService
	partners *OCPIPartnerService
	events   *OCPIEventService
	commands *OCPICommandService
	settings *AppSettingsService

	context entities.PipelineContext
	run     entities.PipelineRun
}

// NewPipelineService wires the orchestrator to the services it coordinates.
func NewPipelineService(
	station *ChargerStationService,
	partners *OCPIPartnerService,
	events *OCPIEventService,
	commands *OCPICommandService,
	settings *AppSettingsService,
) *PipelineService {
	return &PipelineService{
		station:  station,
		partners: partners,
		events:   events,
		commands: commands,
		settings: settings,
	}
}

// StartContext arms a run against a partner, location and EVSE. Blank fields
// fall back to the persisted defaults from the Config tab.
func (s *PipelineService) StartContext(context entities.PipelineContext) (PipelineState, error) {
	defaults := s.settings.Get()

	if strings.TrimSpace(context.PartnerSlug) == "" {
		return PipelineState{}, errors.New("partnerSlug is required")
	}
	if _, err := s.partners.Get(context.PartnerSlug); err != nil {
		return PipelineState{}, err
	}

	if context.LocationID == "" {
		context.LocationID = defaults.DefaultLocationID
	}
	if context.EvseUID == "" {
		context.EvseUID = defaults.DefaultEvseUID
	}
	if context.ConnectorID == "" {
		context.ConnectorID = defaults.DefaultConnectorID
	}
	if context.OCPPConnectorID <= 0 {
		context.OCPPConnectorID = 1
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.context = context
	s.resetRun()
	return s.currentState(), nil
}

// State returns the current cockpit state, recomputed from real evidence.
func (s *PipelineService) State() PipelineState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentState()
}

// Execute runs an action, rejecting anything the current stage does not offer.
func (s *PipelineService) Execute(action entities.PipelineActionID) (PipelineState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := s.currentState()
	if state.Stage == entities.StageIdle {
		return state, ErrPipelineNoContext
	}
	if !isActionAllowed(state.Stage, action) {
		return state, fmt.Errorf("%w: %s", ErrActionNotAvailable, action)
	}

	if err := s.perform(action); err != nil {
		return s.currentState(), err
	}
	return s.currentState(), nil
}

// perform dispatches a validated action to the service that owns it.
func (s *PipelineService) perform(action entities.PipelineActionID) error {
	switch action {
	case entities.ActionPlug:
		return s.connectorAction(action, func(point *entities.ChargerPoint) error { return point.PlugCable() }, "cabo plugado")
	case entities.ActionUnplug:
		return s.connectorAction(action, func(point *entities.ChargerPoint) error { return point.UnplugCable() }, "cabo removido")
	case entities.ActionSuspendEV:
		return s.connectorAction(action, func(point *entities.ChargerPoint) error { return point.SuspendEV() }, "recarga pausada pelo veículo")
	case entities.ActionSuspendEVSE:
		return s.connectorAction(action, func(point *entities.ChargerPoint) error { return point.SuspendEVSE() }, "recarga pausada pelo carregador")
	case entities.ActionResume:
		return s.connectorAction(action, func(point *entities.ChargerPoint) error { return point.ResumeCharging() }, "recarga retomada")
	case entities.ActionClearFault:
		return s.connectorAction(action, func(point *entities.ChargerPoint) error { return point.ClearFault() }, "falha limpa")
	case entities.ActionFault:
		return s.connectorAction(action, func(point *entities.ChargerPoint) error {
			return point.SetFault(entities.ErrorEVCommunicationError, "falha injetada pelo cockpit", "")
		}, "falha injetada no conector")
	case entities.ActionStopLocal:
		s.run.StopRequested = true
		return s.connectorAction(action, func(point *entities.ChargerPoint) error { return point.StopTransaction() }, "StopTransaction enviado pelo carregador")

	case entities.ActionStartLocal:
		return s.startLocal()
	case entities.ActionStartPartner:
		return s.startViaPartner(entities.ActionStartPartner, "")
	case entities.ActionStartInvalidAuth:
		return s.startViaPartner(entities.ActionStartInvalidAuth, pipelineInvalidToken)
	case entities.ActionStartOccupied:
		return s.startOnOccupiedEVSE()
	case entities.ActionStopRemote:
		return s.stopViaPartner()

	case entities.ActionSendMeter:
		s.station.TriggerMeterValues(s.context.OCPPConnectorID)
		s.run.Note = "MeterValues disparado"
		return nil
	case entities.ActionWaitTimeout:
		s.run.Note = "aguardando timeout — nenhuma mensagem enviada"
		return nil
	case entities.ActionReset:
		s.resetRun()
		return nil
	}

	return fmt.Errorf("%w: %s", ErrActionNotAvailable, action)
}

// connectorAction applies a domain method to the run's connector and pushes the
// resulting StatusNotification, exactly as the connector API does.
func (s *PipelineService) connectorAction(action entities.PipelineActionID, apply func(*entities.ChargerPoint) error, note string) error {
	point := s.point()
	if point == nil {
		return fmt.Errorf("connector %d not found", s.context.OCPPConnectorID)
	}

	if err := apply(point); err != nil {
		return err
	}

	s.station.TriggerStatusNotification(s.context.OCPPConnectorID)
	s.run.Note = note
	return nil
}

// startLocal drives the charger's own start flow: plug, Preparing, Authorize
// and StartTransaction, without involving any partner.
func (s *PipelineService) startLocal() error {
	point := s.point()
	if point == nil {
		return fmt.Errorf("connector %d not found", s.context.OCPPConnectorID)
	}

	// An already plugged cable is not an error for this flow.
	_ = point.PlugCable()

	if err := point.StartRemoteTransaction(); err != nil {
		return err
	}
	s.station.TriggerStatusNotification(s.context.OCPPConnectorID)
	s.station.SendAuthorize(s.context.OCPPConnectorID, pipelineLocalIdTag)
	s.station.SendStartTransaction(s.context.OCPPConnectorID, pipelineLocalIdTag, point.MeterValue)

	s.run.Variant = entities.ActionStartLocal
	s.run.StartDispatched = true
	s.run.Note = "START local disparado (Authorize + StartTransaction)"
	return nil
}

// startViaPartner dispatches a START_SESSION through the OCPI command service.
// An override token is how the "token inválido" variant provokes a 401.
func (s *PipelineService) startViaPartner(variant entities.PipelineActionID, authTokenOverride string) error {
	result, err := s.commands.StartSessionWith(s.context.PartnerSlug, StartSessionOptions{
		LocationID:        s.context.LocationID,
		EvseUID:           s.context.EvseUID,
		ConnectorID:       s.context.ConnectorID,
		AuthTokenOverride: authTokenOverride,
	})
	if err != nil {
		return err
	}

	s.run.Variant = variant
	s.run.StartDispatched = true
	s.run.CommandUID = result.CommandUID
	s.run.TokenUID = result.TokenUID
	s.run.ContractID = result.ContractID
	s.run.Note = fmt.Sprintf("START_SESSION enviado (HTTP %d)", result.StatusCode)
	return nil
}

// startOnOccupiedEVSE occupies the EVSE with a local charge first, so the
// partner's START lands on a connector that is already busy.
func (s *PipelineService) startOnOccupiedEVSE() error {
	point := s.point()
	if point == nil {
		return fmt.Errorf("connector %d not found", s.context.OCPPConnectorID)
	}

	if point.Status == entities.StatusAvailable || point.Status == entities.StatusPreparing {
		if err := s.startLocal(); err != nil {
			return err
		}
	}

	if err := s.startViaPartner(entities.ActionStartOccupied, ""); err != nil {
		return err
	}

	s.run.Note = "EVSE ocupado por recarga local; " + s.run.Note
	return nil
}

// stopViaPartner sends STOP_SESSION for the session our platform pushed back.
func (s *PipelineService) stopViaPartner() error {
	if s.run.SessionID == "" {
		return errors.New("nenhum session_id recebido ainda: use 'Parar no carregador'")
	}

	result, err := s.commands.StopSession(s.context.PartnerSlug, s.run.SessionID)
	if err != nil {
		return err
	}

	s.run.StopRequested = true
	s.run.Note = fmt.Sprintf("STOP_SESSION enviado (HTTP %d)", result.StatusCode)
	return nil
}

// currentState recomputes the whole cockpit view. The caller must hold the lock.
func (s *PipelineService) currentState() PipelineState {
	if s.context.PartnerSlug == "" {
		return PipelineState{
			Stage:      entities.StageIdle,
			StageLabel: stageLabels[entities.StageIdle],
			StageHint:  stageHints[entities.StageIdle],
			Hops:       buildHops(pipelineEvidence{}, chargerSnapshot{}, entities.StageIdle, entities.PipelineRun{}),
			Actions:    actionsFor(entities.StageIdle, entities.PipelineRun{}, chargerSnapshot{}),
		}
	}

	evidence := collectEvidence(s.events.List(s.context.PartnerSlug, s.run.SinceEventID, 0))
	if evidence.SessionID != "" {
		s.run.SessionID = evidence.SessionID
	}

	charger := s.snapshot()
	stage := deriveStage(s.context, evidence, charger, s.run)

	return PipelineState{
		Context:    s.context,
		Stage:      stage,
		StageLabel: stageLabels[stage],
		StageHint:  stageHints[stage],
		Hops:       buildHops(evidence, charger, stage, s.run),
		Actions:    actionsFor(stage, s.run, charger),
		Run:        s.run,
		Charger:    s.chargerView(charger),
	}
}

// resetRun opens a fresh run on the current context, moving the event cursor
// past everything the previous run produced.
func (s *PipelineService) resetRun() {
	s.run = entities.PipelineRun{
		SinceEventID: s.events.LastID(s.context.PartnerSlug),
		StartedAt:    time.Now().UTC(),
	}
}

// point returns the run's connector, or nil when it does not exist.
func (s *PipelineService) point() *entities.ChargerPoint {
	return s.station.GetStation().GetPoint(s.context.OCPPConnectorID)
}

// snapshot reads the connector state the derivation reasons about.
func (s *PipelineService) snapshot() chargerSnapshot {
	point := s.point()
	if point == nil {
		return chargerSnapshot{}
	}

	return chargerSnapshot{
		Found:         true,
		Status:        point.Status,
		CablePlugged:  point.CablePlugged,
		TransactionID: point.CurrentTransaction,
		ErrorDetail:   chargerErrorDetail(point),
	}
}

// chargerView builds the connector summary shown beside the pipeline.
func (s *PipelineService) chargerView(charger chargerSnapshot) *PipelineCharger {
	point := s.point()
	if point == nil || !charger.Found {
		return nil
	}

	return &PipelineCharger{
		ConnectorID:    point.ID,
		Status:         string(point.Status),
		CablePlugged:   point.CablePlugged,
		TransactionID:  point.CurrentTransaction,
		BatteryPercent: point.BatteryPercent(s.settings.BatteryCapacityKWh()),
	}
}

// chargerErrorDetail renders the connector fault for the red hop.
func chargerErrorDetail(point *entities.ChargerPoint) string {
	if point.Status != entities.StatusFaulted {
		return ""
	}
	if point.ErrorInfo != "" {
		return string(point.ErrorCode) + ": " + point.ErrorInfo
	}
	return string(point.ErrorCode)
}
