package services

import (
	"encoding/json"
	"errors"
	"testing"

	"EV-Client-Simulator/app/domain/abstracts"
	"EV-Client-Simulator/app/domain/entities"
)

// mockPipelineCommandClient captures the dispatched command instead of sending
// it, and can be told to answer with any status.
type mockPipelineCommandClient struct {
	lastURL     string
	lastToken   string
	lastPayload interface{}
	status      int
	err         error
}

func (c *mockPipelineCommandClient) PostCommand(url, token string, payload interface{}) (*abstracts.OCPICommandResponse, error) {
	c.lastURL = url
	c.lastToken = token
	c.lastPayload = payload
	if c.err != nil {
		return nil, c.err
	}

	status := c.status
	if status == 0 {
		status = 200
	}
	return &abstracts.OCPICommandResponse{StatusCode: status, Body: []byte(`{"status_code":1000}`)}, nil
}

type pipelineTestEnv struct {
	pipeline *PipelineService
	station  *ChargerStationService
	events   *OCPIEventService
	client   *mockPipelineCommandClient
}

// newPipelineTestEnv wires the orchestrator against real services, draining the
// station's outbound channel the way the messaging controller does in
// production so a StatusNotification never blocks a test.
func newPipelineTestEnv(t *testing.T) *pipelineTestEnv {
	t.Helper()

	station := entities.NewChargerStation("test-station")
	messages := make(chan entities.Message, 64)
	errorsChannel := make(chan error, 8)
	go func() {
		for range messages {
		}
	}()

	stationService := NewChargerStationSerice(&station, messages, errorsChannel)

	partnerService := NewOCPIPartnerService(nil)
	if _, err := partnerService.Create(entities.OCPIPartner{
		Slug: "nayax-sim", Name: "Nayax", PartyID: "NYX", CountryCode: "BR",
		TokenToCallUs: "token-b", TokenExpected: "token-a",
		OCPIBaseURL: "https://ocpi-dev.nucharge.com.br", PublicBaseURL: "https://sim-dev.nucharge.com.br",
	}); err != nil {
		t.Fatalf("could not seed partner: %v", err)
	}

	eventService := NewOCPIEventService(nil)
	client := &mockPipelineCommandClient{}
	commandService := NewOCPICommandService(partnerService, eventService, client)
	settingsService := NewAppSettingsService(entities.AppSettings{
		OCPIBaseURL:        "https://ocpi-dev.nucharge.com.br",
		PublicBaseURL:      "https://sim-dev.nucharge.com.br",
		DefaultLocationID:  "LOC-DEFAULT",
		DefaultEvseUID:     "EVSE-DEFAULT",
		DefaultConnectorID: "1",
		BatteryCapacityKWh: 60,
	}, nil)

	return &pipelineTestEnv{
		pipeline: NewPipelineService(stationService, partnerService, eventService, commandService, settingsService),
		station:  stationService,
		events:   eventService,
		client:   client,
	}
}

// arm sets the default context and returns the resulting state.
func (e *pipelineTestEnv) arm(t *testing.T) PipelineState {
	t.Helper()

	state, err := e.pipeline.StartContext(entities.PipelineContext{PartnerSlug: "nayax-sim"})
	if err != nil {
		t.Fatalf("StartContext() returned an error: %v", err)
	}
	return state
}

// execute runs an action and fails the test when it errors.
func (e *pipelineTestEnv) execute(t *testing.T, action entities.PipelineActionID) PipelineState {
	t.Helper()

	state, err := e.pipeline.Execute(action)
	if err != nil {
		t.Fatalf("Execute(%q) returned an error: %v", action, err)
	}
	return state
}

func TestPipelineStartsIdleUntilArmed(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)

	actualState := mockEnv.pipeline.State()

	if actualState.Stage != entities.StageIdle {
		t.Errorf("Stage = %q, expected %q", actualState.Stage, entities.StageIdle)
	}
	if len(actualState.Actions) != 0 {
		t.Errorf("idle offered %d actions, expected none", len(actualState.Actions))
	}
	if len(actualState.Hops) != 5 {
		t.Errorf("Hops = %d, expected the 5 pipeline nodes even while idle", len(actualState.Hops))
	}
}

func TestPipelineStartContextAppliesDefaults(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)

	actualState := mockEnv.arm(t)

	if actualState.Context.LocationID != "LOC-DEFAULT" || actualState.Context.EvseUID != "EVSE-DEFAULT" {
		t.Errorf("Context = %+v, expected the configured defaults", actualState.Context)
	}
	if actualState.Context.OCPPConnectorID != 1 {
		t.Errorf("OCPPConnectorID = %d, expected 1", actualState.Context.OCPPConnectorID)
	}
	if actualState.Stage != entities.StagePlug {
		t.Errorf("Stage = %q, expected %q on a fresh connector", actualState.Stage, entities.StagePlug)
	}
}

func TestPipelineStartContextRejectsUnknownPartner(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)

	_, err := mockEnv.pipeline.StartContext(entities.PipelineContext{PartnerSlug: "ghost"})

	if !errors.Is(err, ErrPartnerNotFound) {
		t.Errorf("StartContext() error = %v, expected ErrPartnerNotFound", err)
	}
}

func TestPipelineRejectsActionOutsideItsStage(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)

	_, err := mockEnv.pipeline.Execute(entities.ActionSuspendEV)

	if !errors.Is(err, ErrActionNotAvailable) {
		t.Errorf("Execute() error = %v, expected ErrActionNotAvailable", err)
	}
}

func TestPipelineRejectsActionsWithoutContext(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)

	_, err := mockEnv.pipeline.Execute(entities.ActionPlug)

	if !errors.Is(err, ErrPipelineNoContext) {
		t.Errorf("Execute() error = %v, expected ErrPipelineNoContext", err)
	}
}

// TestPipelinePlugAdvancesToStart walks the first hand-off: plugging the cable
// is what unlocks the start variants.
func TestPipelinePlugAdvancesToStart(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)

	actualState := mockEnv.execute(t, entities.ActionPlug)

	if actualState.Stage != entities.StageStart {
		t.Fatalf("Stage = %q, expected %q", actualState.Stage, entities.StageStart)
	}
	if actualState.Charger == nil || !actualState.Charger.CablePlugged {
		t.Errorf("Charger = %+v, expected the cable to be plugged", actualState.Charger)
	}
}

func TestPipelineStartViaPartnerDispatchesCommand(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)

	actualState := mockEnv.execute(t, entities.ActionStartPartner)

	if mockEnv.client.lastToken != "token-b" {
		t.Errorf("outbound token = %q, expected the partner's own credential", mockEnv.client.lastToken)
	}
	command, ok := mockEnv.client.lastPayload.(entities.StartSessionCommand)
	if !ok {
		t.Fatalf("payload type = %T, expected StartSessionCommand", mockEnv.client.lastPayload)
	}
	if command.LocationID != "LOC-DEFAULT" || command.EvseUID != "EVSE-DEFAULT" {
		t.Errorf("command = %+v, expected the armed context to be used", command)
	}
	if actualState.Stage != entities.StageStarting {
		t.Errorf("Stage = %q, expected %q", actualState.Stage, entities.StageStarting)
	}
	if actualState.Run.CommandUID == "" || actualState.Run.ContractID == "" {
		t.Errorf("Run = %+v, expected the dispatched command to be recorded", actualState.Run)
	}
}

// TestPipelineInvalidTokenVariant is the deliberate negative test: a wrong
// Authorization must reach the wire and the 401 must land on the OCPI hop.
func TestPipelineInvalidTokenVariant(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.client.status = 401
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)

	actualState := mockEnv.execute(t, entities.ActionStartInvalidAuth)

	if mockEnv.client.lastToken == "token-b" {
		t.Errorf("outbound token = %q, expected the deliberately wrong one", mockEnv.client.lastToken)
	}
	if actualState.Stage != entities.StageFailed {
		t.Fatalf("Stage = %q, expected %q", actualState.Stage, entities.StageFailed)
	}

	ocpi := findHop(t, actualState.Hops, entities.HopOCPI)
	if ocpi.Status != entities.HopFailed || ocpi.Error != "HTTP 401" {
		t.Errorf("ocpi hop = %+v, expected a red HTTP 401", ocpi)
	}
	if !ocpi.Expected {
		t.Errorf("ocpi hop Expected = false, expected the provoked failure to be flagged")
	}
}

// TestPipelineTransportFailurePropagates covers the platform being unreachable.
func TestPipelineTransportFailurePropagates(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.client.err = errors.New("dial tcp: connection refused")
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)

	actualState := mockEnv.execute(t, entities.ActionStartPartner)

	if actualState.Stage != entities.StageFailed {
		t.Fatalf("Stage = %q, expected %q", actualState.Stage, entities.StageFailed)
	}
	partner := findHop(t, actualState.Hops, entities.HopPartner)
	if partner.Status != entities.HopFailed || partner.Error == "" {
		t.Errorf("partner hop = %+v, expected the transport error inline", partner)
	}
	if partner.Expected {
		t.Errorf("partner hop Expected = true, expected an unplanned failure")
	}
}

// TestPipelineFullRoundTrip drives the pilot scenario the cockpit exists for:
// START, platform accepts, charger charges, STOP, CDR echoes back.
func TestPipelineFullRoundTrip(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)
	startState := mockEnv.execute(t, entities.ActionStartPartner)

	mockEnv.events.Record(entities.OCPIEvent{
		PartnerSlug: "nayax-sim", Direction: entities.OCPIDirectionIn,
		Kind: entities.OCPIEventCommandResult, Body: []byte(`{"result":"ACCEPTED"}`),
	})

	mockEnv.startCharging(t)

	chargingState := mockEnv.pipeline.State()
	if chargingState.Stage != entities.StageCharging {
		t.Fatalf("Stage = %q, expected %q", chargingState.Stage, entities.StageCharging)
	}
	if platform := findHop(t, chargingState.Hops, entities.HopPlatform); platform.Status != entities.HopConfirmed {
		t.Errorf("platform hop = %+v, expected it green after an ACCEPTED CommandResult", platform)
	}

	mockEnv.recordSession(t, "sess-1", startState.Run.TokenUID, startState.Run.ContractID)

	stoppedState := mockEnv.execute(t, entities.ActionStopRemote)
	if !stoppedState.Run.StopRequested {
		t.Errorf("Run = %+v, expected the stop to be recorded", stoppedState.Run)
	}

	mockEnv.recordCDR(t, "sess-1", startState.Run.TokenUID, startState.Run.ContractID)

	doneState := mockEnv.pipeline.State()
	if doneState.Stage != entities.StageDone {
		t.Fatalf("Stage = %q, expected %q once the CDR lands", doneState.Stage, entities.StageDone)
	}
	if cdr := findHop(t, doneState.Hops, entities.HopCDR); cdr.Status != entities.HopConfirmed {
		t.Errorf("cdr hop = %+v, expected it green with a matching cdr_token", cdr)
	}
}

// TestPipelineMangledCDRTokenFailsTheRun is the regression the pilot cares
// about: our platform replacing the partner's contract_id.
func TestPipelineMangledCDRTokenFailsTheRun(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)
	startState := mockEnv.execute(t, entities.ActionStartPartner)

	mockEnv.recordCDR(t, "sess-1", startState.Run.TokenUID, "REPLACED-BY-PLATFORM")

	actualState := mockEnv.pipeline.State()

	if actualState.Stage != entities.StageFailed {
		t.Fatalf("Stage = %q, expected %q", actualState.Stage, entities.StageFailed)
	}
	cdr := findHop(t, actualState.Hops, entities.HopCDR)
	if cdr.Status != entities.HopFailed || cdr.Error == "" {
		t.Errorf("cdr hop = %+v, expected a red hop naming the divergent field", cdr)
	}
}

// TestPipelineResetMovesTheEventCursor proves a new run ignores the previous
// run's events instead of inheriting its green hops.
func TestPipelineResetMovesTheEventCursor(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)
	startState := mockEnv.execute(t, entities.ActionStartPartner)
	mockEnv.recordCDR(t, "sess-1", startState.Run.TokenUID, startState.Run.ContractID)

	if actualStage := mockEnv.pipeline.State().Stage; actualStage != entities.StageDone {
		t.Fatalf("Stage = %q, expected %q before the reset", actualStage, entities.StageDone)
	}

	actualState := mockEnv.execute(t, entities.ActionReset)

	if actualState.Stage != entities.StageStart {
		t.Errorf("Stage = %q, expected %q with the cable still plugged", actualState.Stage, entities.StageStart)
	}
	if partner := findHop(t, actualState.Hops, entities.HopPartner); partner.Status == entities.HopConfirmed {
		t.Errorf("partner hop = %+v, expected the previous run's evidence to be dropped", partner)
	}
}

// TestPipelineLocalStartSkipsThePartner covers the OCPP-only variant.
func TestPipelineLocalStartSkipsThePartner(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)

	actualState := mockEnv.execute(t, entities.ActionStartLocal)

	if mockEnv.client.lastPayload != nil {
		t.Errorf("dispatched %+v, expected the local start to skip OCPI", mockEnv.client.lastPayload)
	}
	if actualState.Run.Variant != entities.ActionStartLocal {
		t.Errorf("Run.Variant = %q, expected %q", actualState.Run.Variant, entities.ActionStartLocal)
	}
	if actualState.Stage != entities.StageStarting {
		t.Errorf("Stage = %q, expected %q", actualState.Stage, entities.StageStarting)
	}
}

// TestPipelineSuspendAndResume covers the charging-stage variants.
func TestPipelineSuspendAndResume(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)
	mockEnv.execute(t, entities.ActionStartPartner)
	mockEnv.startCharging(t)

	suspendedState := mockEnv.execute(t, entities.ActionSuspendEV)
	if suspendedState.Stage != entities.StageSuspended {
		t.Fatalf("Stage = %q, expected %q", suspendedState.Stage, entities.StageSuspended)
	}

	resumedState := mockEnv.execute(t, entities.ActionResume)
	if resumedState.Stage != entities.StageCharging {
		t.Errorf("Stage = %q, expected %q", resumedState.Stage, entities.StageCharging)
	}
}

// TestPipelineFaultInjectionFailsTheRun covers the charger-side failure.
func TestPipelineFaultInjectionFailsTheRun(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)
	mockEnv.execute(t, entities.ActionStartPartner)
	mockEnv.startCharging(t)

	actualState := mockEnv.execute(t, entities.ActionFault)

	if actualState.Stage != entities.StageFailed {
		t.Fatalf("Stage = %q, expected %q", actualState.Stage, entities.StageFailed)
	}
	charger := findHop(t, actualState.Hops, entities.HopCharger)
	if charger.Status != entities.HopFailed || charger.Error == "" {
		t.Errorf("charger hop = %+v, expected the fault inline", charger)
	}
}

// TestPipelineRemoteStopNeedsASession explains itself in the error message
// rather than silently stopping the wrong way.
func TestPipelineRemoteStopNeedsASession(t *testing.T) {
	mockEnv := newPipelineTestEnv(t)
	mockEnv.arm(t)
	mockEnv.execute(t, entities.ActionPlug)
	mockEnv.execute(t, entities.ActionStartPartner)
	mockEnv.startCharging(t)

	_, err := mockEnv.pipeline.Execute(entities.ActionStopRemote)

	if err == nil {
		t.Fatalf("Execute() returned no error, expected the missing session to be reported")
	}
}

// startCharging drives the virtual charger into Charging, as the OCPP server
// would after accepting the StartTransaction.
func (e *pipelineTestEnv) startCharging(t *testing.T) {
	t.Helper()

	point := e.station.GetStation().GetPoint(1)
	if point.Status != entities.StatusPreparing {
		if err := point.StartRemoteTransaction(); err != nil {
			t.Fatalf("StartRemoteTransaction() returned an error: %v", err)
		}
	}
	if err := point.StartTransaction(4242, entities.TransactionSettings{}); err != nil {
		t.Fatalf("StartTransaction() returned an error: %v", err)
	}
}

// recordSession stores the Session push our platform would send back.
func (e *pipelineTestEnv) recordSession(t *testing.T, sessionID, tokenUID, contractID string) {
	t.Helper()
	e.events.Record(entities.OCPIEvent{
		PartnerSlug: "nayax-sim", Direction: entities.OCPIDirectionIn, Kind: entities.OCPIEventSession,
		SessionID: sessionID, TokenUID: tokenUID, TokenType: "AD_HOC_USER", ContractID: contractID,
		Body: sessionBody(t, sessionID, tokenUID, contractID),
	})
}

// recordCDR stores the CDR push that closes the round trip.
func (e *pipelineTestEnv) recordCDR(t *testing.T, sessionID, tokenUID, contractID string) {
	t.Helper()
	e.events.Record(entities.OCPIEvent{
		PartnerSlug: "nayax-sim", Direction: entities.OCPIDirectionIn, Kind: entities.OCPIEventCDR,
		SessionID: sessionID, TokenUID: tokenUID, TokenType: "AD_HOC_USER", ContractID: contractID,
		Body: sessionBody(t, sessionID, tokenUID, contractID),
	})
}

// sessionBody renders the OCPI object body carrying a cdr_token.
func sessionBody(t *testing.T, sessionID, tokenUID, contractID string) json.RawMessage {
	t.Helper()

	body, err := json.Marshal(map[string]interface{}{
		"id": sessionID,
		"cdr_token": map[string]string{
			"uid": tokenUID, "type": "AD_HOC_USER", "contract_id": contractID,
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal() returned an error: %v", err)
	}
	return body
}

// findHop returns a hop by id, failing the test when it is missing.
func findHop(t *testing.T, hops []entities.PipelineHop, id entities.PipelineHopID) entities.PipelineHop {
	t.Helper()

	for _, hop := range hops {
		if hop.ID == id {
			return hop
		}
	}
	t.Fatalf("hop %q not found in %+v", id, hops)
	return entities.PipelineHop{}
}
