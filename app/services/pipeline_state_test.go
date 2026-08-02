package services

import (
	"testing"

	"EV-Client-Simulator/app/domain/entities"
)

// armedContext is the context every stage test runs against.
var armedContext = entities.PipelineContext{
	PartnerSlug:     "nayax-sim",
	LocationID:      "LOC-1",
	EvseUID:         "EVSE-1",
	OCPPConnectorID: 1,
}

func TestDeriveStage(t *testing.T) {
	tests := []struct {
		name          string
		inputContext  entities.PipelineContext
		inputEvidence pipelineEvidence
		inputCharger  chargerSnapshot
		inputRun      entities.PipelineRun
		expectedStage entities.PipelineStage
	}{
		{
			name:          "no context is idle",
			inputContext:  entities.PipelineContext{},
			expectedStage: entities.StageIdle,
		},
		{
			name:          "armed context with the cable out asks for a plug",
			inputContext:  armedContext,
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable},
			expectedStage: entities.StagePlug,
		},
		{
			name:          "cable plugged unlocks the start stage",
			inputContext:  armedContext,
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable, CablePlugged: true},
			expectedStage: entities.StageStart,
		},
		{
			name:          "command sent moves to starting",
			inputContext:  armedContext,
			inputEvidence: pipelineEvidence{CommandSent: true, CommandStatus: 200},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable, CablePlugged: true},
			expectedStage: entities.StageStarting,
		},
		{
			name:          "local start also moves to starting",
			inputContext:  armedContext,
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable, CablePlugged: true},
			inputRun:      entities.PipelineRun{StartDispatched: true, Variant: entities.ActionStartLocal},
			expectedStage: entities.StageStarting,
		},
		{
			name:          "preparing before any start is still the start stage",
			inputContext:  armedContext,
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusPreparing, CablePlugged: true},
			expectedStage: entities.StageStart,
		},
		{
			name:          "charging connector is the charging stage",
			inputContext:  armedContext,
			inputEvidence: pipelineEvidence{CommandSent: true, CommandStatus: 200, CommandResult: "ACCEPTED"},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusCharging, CablePlugged: true},
			expectedStage: entities.StageCharging,
		},
		{
			name:          "suspended by the EV is the suspended stage",
			inputContext:  armedContext,
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusSuspendedEV, CablePlugged: true},
			expectedStage: entities.StageSuspended,
		},
		{
			name:          "suspended by the EVSE is the suspended stage",
			inputContext:  armedContext,
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusSuspendedEVSE, CablePlugged: true},
			expectedStage: entities.StageSuspended,
		},
		{
			name:          "finishing connector is the finishing stage",
			inputContext:  armedContext,
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusFinishing, CablePlugged: true},
			expectedStage: entities.StageFinishing,
		},
		{
			name:          "a requested stop keeps us finishing until the cdr lands",
			inputContext:  armedContext,
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable},
			inputRun:      entities.PipelineRun{StartDispatched: true, StopRequested: true},
			expectedStage: entities.StageFinishing,
		},
		{
			name:          "the cdr completes the run",
			inputContext:  armedContext,
			inputEvidence: pipelineEvidence{CommandSent: true, CommandStatus: 200, CDRSeen: true},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable},
			inputRun:      entities.PipelineRun{StartDispatched: true, StopRequested: true},
			expectedStage: entities.StageDone,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualStage := deriveStage(test.inputContext, test.inputEvidence, test.inputCharger, test.inputRun)

			if actualStage != test.expectedStage {
				t.Errorf("deriveStage() = %q, expected %q", actualStage, test.expectedStage)
			}
		})
	}
}

// TestDeriveStageFailurePropagation locks in every way a run can be condemned.
func TestDeriveStageFailurePropagation(t *testing.T) {
	tests := []struct {
		name          string
		inputEvidence pipelineEvidence
		inputCharger  chargerSnapshot
		inputRun      entities.PipelineRun
	}{
		{
			name:          "transport error on the command",
			inputEvidence: pipelineEvidence{CommandSent: true, CommandError: "dial tcp: connection refused"},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable, CablePlugged: true},
		},
		{
			name:          "platform rejected the credential",
			inputEvidence: pipelineEvidence{CommandSent: true, CommandStatus: 401},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable, CablePlugged: true},
			inputRun:      entities.PipelineRun{Variant: entities.ActionStartInvalidAuth, StartDispatched: true},
		},
		{
			name:          "command result came back rejected",
			inputEvidence: pipelineEvidence{CommandSent: true, CommandStatus: 200, CommandResult: "REJECTED"},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusPreparing, CablePlugged: true},
		},
		{
			name:          "charger faulted mid charge",
			inputEvidence: pipelineEvidence{CommandSent: true, CommandStatus: 200, CommandResult: "ACCEPTED"},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusFaulted, CablePlugged: true},
		},
		{
			name: "cdr came back with a mangled cdr_token",
			inputEvidence: pipelineEvidence{
				CommandSent: true, CommandStatus: 200, CDRSeen: true,
				EchoMismatch: []string{"contract_id"},
			},
			inputCharger: chargerSnapshot{Found: true, Status: entities.StatusAvailable},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualStage := deriveStage(armedContext, test.inputEvidence, test.inputCharger, test.inputRun)

			if actualStage != entities.StageFailed {
				t.Errorf("deriveStage() = %q, expected %q", actualStage, entities.StageFailed)
			}
		})
	}
}

func TestActionsForStage(t *testing.T) {
	// pluggedCharger is the connector state most stages run against.
	pluggedCharger := chargerSnapshot{Found: true, Status: entities.StatusCharging, CablePlugged: true}

	tests := []struct {
		name            string
		inputStage      entities.PipelineStage
		inputRun        entities.PipelineRun
		inputCharger    chargerSnapshot
		expectedActions []entities.PipelineActionID
		expectedPrimary entities.PipelineActionID
	}{
		{
			name:            "idle offers nothing",
			inputStage:      entities.StageIdle,
			expectedActions: []entities.PipelineActionID{},
		},
		{
			name:       "plug stage",
			inputStage: entities.StagePlug,
			expectedActions: []entities.PipelineActionID{
				entities.ActionPlug, entities.ActionStartLocal, entities.ActionReset,
			},
			expectedPrimary: entities.ActionPlug,
		},
		{
			name:         "start stage carries every start variant",
			inputStage:   entities.StageStart,
			inputCharger: pluggedCharger,
			expectedActions: []entities.PipelineActionID{
				entities.ActionStartPartner,
				entities.ActionStartInvalidAuth,
				entities.ActionStartOccupied,
				entities.ActionStartLocal,
				entities.ActionUnplug,
				entities.ActionReset,
			},
			expectedPrimary: entities.ActionStartPartner,
		},
		{
			name:         "charging stage carries the charging variants",
			inputStage:   entities.StageCharging,
			inputRun:     entities.PipelineRun{SessionID: "sess-1"},
			inputCharger: pluggedCharger,
			expectedActions: []entities.PipelineActionID{
				entities.ActionStopRemote,
				entities.ActionStopLocal,
				entities.ActionSuspendEV,
				entities.ActionSuspendEVSE,
				entities.ActionFault,
				entities.ActionSendMeter,
				entities.ActionUnplug,
				entities.ActionReset,
			},
			expectedPrimary: entities.ActionStopRemote,
		},
		{
			name:         "suspended stage leads with resume",
			inputStage:   entities.StageSuspended,
			inputRun:     entities.PipelineRun{SessionID: "sess-1"},
			inputCharger: pluggedCharger,
			expectedActions: []entities.PipelineActionID{
				entities.ActionResume,
				entities.ActionStopRemote,
				entities.ActionStopLocal,
				entities.ActionFault,
				entities.ActionUnplug,
				entities.ActionReset,
			},
			expectedPrimary: entities.ActionResume,
		},
		{
			name:         "finishing stage waits for the cdr",
			inputStage:   entities.StageFinishing,
			inputCharger: pluggedCharger,
			expectedActions: []entities.PipelineActionID{
				entities.ActionUnplug,
				entities.ActionSendMeter,
				entities.ActionWaitTimeout,
				entities.ActionReset,
			},
			expectedPrimary: entities.ActionUnplug,
		},
		{
			name:            "done stage only restarts",
			inputStage:      entities.StageDone,
			expectedActions: []entities.PipelineActionID{entities.ActionReset},
			expectedPrimary: entities.ActionReset,
		},
		{
			name:         "failed stage restarts or recovers the connector",
			inputStage:   entities.StageFailed,
			inputCharger: chargerSnapshot{Found: true, Status: entities.StatusFaulted, CablePlugged: true},
			expectedActions: []entities.PipelineActionID{
				entities.ActionReset, entities.ActionClearFault, entities.ActionUnplug,
			},
			expectedPrimary: entities.ActionReset,
		},
		{
			name:         "without an ocpi session the local stop is emphasised",
			inputStage:   entities.StageCharging,
			inputCharger: pluggedCharger,
			expectedActions: []entities.PipelineActionID{
				entities.ActionStopRemote,
				entities.ActionStopLocal,
				entities.ActionSuspendEV,
				entities.ActionSuspendEVSE,
				entities.ActionFault,
				entities.ActionSendMeter,
				entities.ActionUnplug,
				entities.ActionReset,
			},
			expectedPrimary: entities.ActionStopLocal,
		},
		{
			// A button that would only return an error is not offered at all.
			name:            "failed stage without a fault or a cable hides the recovery actions",
			inputStage:      entities.StageFailed,
			inputCharger:    chargerSnapshot{Found: true, Status: entities.StatusAvailable},
			expectedActions: []entities.PipelineActionID{entities.ActionReset},
			expectedPrimary: entities.ActionReset,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualActions := actionsFor(test.inputStage, test.inputRun, test.inputCharger)

			if len(actualActions) != len(test.expectedActions) {
				t.Fatalf("actionsFor() returned %d actions, expected %d (%+v)", len(actualActions), len(test.expectedActions), actualActions)
			}
			for index, expected := range test.expectedActions {
				if actualActions[index].ID != expected {
					t.Errorf("action[%d] = %q, expected %q", index, actualActions[index].ID, expected)
				}
				if actualActions[index].Label == "" {
					t.Errorf("action %q has no label", actualActions[index].ID)
				}
			}

			for _, action := range actualActions {
				if action.Primary && action.ID != test.expectedPrimary {
					t.Errorf("primary action = %q, expected %q", action.ID, test.expectedPrimary)
				}
			}
		})
	}
}

// TestIsActionAllowedRejectsOutOfStageActions is the guard behind the HTTP 409:
// an action that the current stage does not offer can never run.
func TestIsActionAllowedRejectsOutOfStageActions(t *testing.T) {
	tests := []struct {
		name            string
		inputStage      entities.PipelineStage
		inputAction     entities.PipelineActionID
		expectedAllowed bool
	}{
		{name: "plug in the plug stage", inputStage: entities.StagePlug, inputAction: entities.ActionPlug, expectedAllowed: true},
		{name: "start via partner before plugging", inputStage: entities.StagePlug, inputAction: entities.ActionStartPartner},
		{name: "suspend before charging", inputStage: entities.StageStart, inputAction: entities.ActionSuspendEV},
		{name: "resume while charging", inputStage: entities.StageCharging, inputAction: entities.ActionResume},
		{name: "anything at all while idle", inputStage: entities.StageIdle, inputAction: entities.ActionPlug},
		{name: "reset once done", inputStage: entities.StageDone, inputAction: entities.ActionReset, expectedAllowed: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualAllowed := isActionAllowed(test.inputStage, test.inputAction)

			if actualAllowed != test.expectedAllowed {
				t.Errorf("isActionAllowed(%q, %q) = %v, expected %v", test.inputStage, test.inputAction, actualAllowed, test.expectedAllowed)
			}
		})
	}
}

func TestBuildHops(t *testing.T) {
	tests := []struct {
		name             string
		inputEvidence    pipelineEvidence
		inputCharger     chargerSnapshot
		inputStage       entities.PipelineStage
		inputRun         entities.PipelineRun
		expectedStatuses map[entities.PipelineHopID]entities.PipelineHopStatus
		expectedExpected map[entities.PipelineHopID]bool
	}{
		{
			name:       "fresh run highlights the partner hop",
			inputStage: entities.StageStart,
			expectedStatuses: map[entities.PipelineHopID]entities.PipelineHopStatus{
				entities.HopPartner:  entities.HopActive,
				entities.HopOCPI:     entities.HopPending,
				entities.HopPlatform: entities.HopPending,
				entities.HopCharger:  entities.HopPending,
				entities.HopCDR:      entities.HopPending,
			},
		},
		{
			name:          "accepted command lights the first three hops",
			inputEvidence: pipelineEvidence{CommandSent: true, CommandStatus: 200, CommandResult: "ACCEPTED"},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusCharging, CablePlugged: true},
			inputStage:    entities.StageCharging,
			expectedStatuses: map[entities.PipelineHopID]entities.PipelineHopStatus{
				entities.HopPartner:  entities.HopConfirmed,
				entities.HopOCPI:     entities.HopConfirmed,
				entities.HopPlatform: entities.HopConfirmed,
				entities.HopCharger:  entities.HopActive,
				entities.HopCDR:      entities.HopPending,
			},
		},
		{
			name:          "an invalid token reddens the ocpi hop as an expected failure",
			inputEvidence: pipelineEvidence{CommandSent: true, CommandStatus: 401},
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusAvailable, CablePlugged: true},
			inputStage:    entities.StageFailed,
			inputRun:      entities.PipelineRun{Variant: entities.ActionStartInvalidAuth, StartDispatched: true},
			expectedStatuses: map[entities.PipelineHopID]entities.PipelineHopStatus{
				entities.HopPartner: entities.HopConfirmed,
				entities.HopOCPI:    entities.HopFailed,
			},
			expectedExpected: map[entities.PipelineHopID]bool{entities.HopOCPI: true},
		},
		{
			name: "a mangled cdr_token reddens the cdr hop",
			inputEvidence: pipelineEvidence{
				CommandSent: true, CommandStatus: 200, CommandResult: "ACCEPTED",
				CDRSeen: true, EchoMismatch: []string{"contract_id"},
			},
			inputCharger: chargerSnapshot{Found: true, Status: entities.StatusAvailable},
			inputStage:   entities.StageFailed,
			inputRun:     entities.PipelineRun{StopRequested: true},
			expectedStatuses: map[entities.PipelineHopID]entities.PipelineHopStatus{
				entities.HopCharger: entities.HopConfirmed,
				entities.HopCDR:     entities.HopFailed,
			},
		},
		{
			name:          "a local start marks the partner hop as skipped",
			inputCharger:  chargerSnapshot{Found: true, Status: entities.StatusCharging, CablePlugged: true},
			inputStage:    entities.StageCharging,
			inputRun:      entities.PipelineRun{Variant: entities.ActionStartLocal, StartDispatched: true},
			inputEvidence: pipelineEvidence{},
			// The busy charger owns the blue highlight, so the bypassed OCPI
			// and Firebase hops stay grey instead of stealing it.
			expectedStatuses: map[entities.PipelineHopID]entities.PipelineHopStatus{
				entities.HopPartner:  entities.HopConfirmed,
				entities.HopOCPI:     entities.HopPending,
				entities.HopPlatform: entities.HopPending,
				entities.HopCharger:  entities.HopActive,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualHops := buildHops(test.inputEvidence, test.inputCharger, test.inputStage, test.inputRun)

			byID := make(map[entities.PipelineHopID]entities.PipelineHop, len(actualHops))
			for _, hop := range actualHops {
				byID[hop.ID] = hop
			}

			for id, expected := range test.expectedStatuses {
				if byID[id].Status != expected {
					t.Errorf("hop %q status = %q, expected %q", id, byID[id].Status, expected)
				}
			}
			for id, expected := range test.expectedExpected {
				if byID[id].Expected != expected {
					t.Errorf("hop %q Expected = %v, expected %v", id, byID[id].Expected, expected)
				}
			}
			for id, hop := range byID {
				if hop.Status == entities.HopFailed && hop.Error == "" {
					t.Errorf("hop %q is red but carries no error", id)
				}
			}
		})
	}
}

func TestCollectEvidence(t *testing.T) {
	inputEchoFailed := false
	inputEvents := []entities.OCPIEvent{
		{ID: 1, Kind: entities.OCPIEventCommandSent, StatusCode: 200},
		{ID: 2, Kind: entities.OCPIEventCommandResult, Body: []byte(`{"result":"accepted"}`)},
		{ID: 3, Kind: entities.OCPIEventSession, SessionID: "sess-1"},
		{ID: 4, Kind: entities.OCPIEventCDR, SessionID: "sess-1", EchoOK: &inputEchoFailed, EchoDiff: []string{"contract_id"}},
	}

	actualEvidence := collectEvidence(inputEvents)

	if !actualEvidence.CommandSent || actualEvidence.CommandStatus != 200 {
		t.Errorf("command evidence = %+v, expected a 200 command_sent", actualEvidence)
	}
	if actualEvidence.CommandResult != "ACCEPTED" {
		t.Errorf("CommandResult = %q, expected %q (upper cased)", actualEvidence.CommandResult, "ACCEPTED")
	}
	if !actualEvidence.SessionSeen || actualEvidence.SessionID != "sess-1" {
		t.Errorf("session evidence = %+v, expected sess-1", actualEvidence)
	}
	if !actualEvidence.CDRSeen || len(actualEvidence.EchoMismatch) != 1 {
		t.Errorf("cdr evidence = %+v, expected a mismatch on contract_id", actualEvidence)
	}
	if !actualEvidence.failed() {
		t.Errorf("failed() = false, expected an echo mismatch to condemn the run")
	}
}
