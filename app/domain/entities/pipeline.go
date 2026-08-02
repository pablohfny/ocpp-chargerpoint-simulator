package entities

import "time"

// PipelineStage is the phase of the end-to-end test the cockpit is in. The
// stage decides which actions the UI may offer, nothing else.
type PipelineStage string

const (
	// StageIdle means no context is armed yet.
	StageIdle PipelineStage = "idle"
	// StagePlug means the context is armed and the cable is still out.
	StagePlug PipelineStage = "plug"
	// StageStart means the cable is in and a charge can be started.
	StageStart PipelineStage = "start"
	// StageStarting means a start was dispatched and we await the charger.
	StageStarting PipelineStage = "starting"
	// StageCharging means the connector is charging.
	StageCharging PipelineStage = "charging"
	// StageSuspended means charging is paused by the EV or the EVSE.
	StageSuspended PipelineStage = "suspended"
	// StageFinishing means the charge is ending and we await the CDR.
	StageFinishing PipelineStage = "finishing"
	// StageDone means the CDR arrived and the round trip is complete.
	StageDone PipelineStage = "done"
	// StageFailed means a hop failed in a way that ends the run.
	StageFailed PipelineStage = "failed"
)

// PipelineHopID identifies a node of the Parceiro → OCPI → Firebase →
// Carregador → CDR visualization.
type PipelineHopID string

const (
	HopPartner  PipelineHopID = "partner"
	HopOCPI     PipelineHopID = "ocpi"
	HopPlatform PipelineHopID = "platform"
	HopCharger  PipelineHopID = "charger"
	HopCDR      PipelineHopID = "cdr"
)

// PipelineHopStatus is the colour a hop shows: grey, blue, green or red.
type PipelineHopStatus string

const (
	// HopPending is grey: the flow has not reached this hop.
	HopPending PipelineHopStatus = "pending"
	// HopActive is blue: this hop is where the flow currently is.
	HopActive PipelineHopStatus = "active"
	// HopConfirmed is green: real evidence says this hop succeeded.
	HopConfirmed PipelineHopStatus = "confirmed"
	// HopFailed is red: this hop failed, with the error carried inline.
	HopFailed PipelineHopStatus = "failed"
)

// PipelineActionID identifies an action the cockpit can execute.
type PipelineActionID string

const (
	ActionPlug             PipelineActionID = "plug"
	ActionUnplug           PipelineActionID = "unplug"
	ActionStartPartner     PipelineActionID = "start_partner"
	ActionStartInvalidAuth PipelineActionID = "start_invalid_token"
	ActionStartOccupied    PipelineActionID = "start_occupied_evse"
	ActionStartLocal       PipelineActionID = "start_local"
	ActionStopRemote       PipelineActionID = "stop_remote"
	ActionStopLocal        PipelineActionID = "stop_local"
	ActionSuspendEV        PipelineActionID = "suspend_ev"
	ActionSuspendEVSE      PipelineActionID = "suspend_evse"
	ActionResume           PipelineActionID = "resume"
	ActionFault            PipelineActionID = "fault"
	ActionClearFault       PipelineActionID = "clear_fault"
	ActionSendMeter        PipelineActionID = "send_meter"
	ActionWaitTimeout      PipelineActionID = "wait_timeout"
	ActionReset            PipelineActionID = "reset"
)

// PipelineContext is the target a pipeline run exercises.
type PipelineContext struct {
	PartnerSlug string `json:"partnerSlug"`
	LocationID  string `json:"locationId"`
	EvseUID     string `json:"evseUid"`
	// ConnectorID is the OCPI connector id sent in START_SESSION.
	ConnectorID string `json:"connectorId"`
	// OCPPConnectorID is the local OCPP connector the virtual charger drives.
	OCPPConnectorID int `json:"ocppConnectorId"`
}

// PipelineHop is one node of the pipeline visualization.
type PipelineHop struct {
	ID     PipelineHopID     `json:"id"`
	Label  string            `json:"label"`
	Status PipelineHopStatus `json:"status"`
	// Detail is a short human readable note, e.g. "HTTP 200" or "Carregando".
	Detail string `json:"detail,omitempty"`
	// Error carries the failure inline so the UI can show it on the red hop.
	Error string `json:"error,omitempty"`
	// Expected marks a failure the chosen variant deliberately provoked, so a
	// red hop can read as a passing negative test.
	Expected bool `json:"expected,omitempty"`
}

// PipelineAction describes an action offered for the current stage.
type PipelineAction struct {
	ID      PipelineActionID `json:"id"`
	Label   string           `json:"label"`
	Hint    string           `json:"hint,omitempty"`
	Primary bool             `json:"primary,omitempty"`
	// Destructive marks a variant that provokes an error on purpose.
	Destructive bool `json:"destructive,omitempty"`
}

// PipelineRun records what the current run has dispatched, so inbound pushes
// can be correlated back to it.
type PipelineRun struct {
	// SinceEventID is the event id the run started after: only events newer
	// than this belong to it.
	SinceEventID int64     `json:"sinceEventId"`
	StartedAt    time.Time `json:"startedAt"`
	// Variant is the start action the run used, which tells the UI whether a
	// failure was provoked on purpose.
	Variant PipelineActionID `json:"variant,omitempty"`
	// StartDispatched records that some start action already ran, including the
	// local OCPP one that never touches a partner.
	StartDispatched bool `json:"startDispatched,omitempty"`
	// CommandUID/TokenUID/ContractID identify the dispatched START_SESSION.
	CommandUID string `json:"commandUid,omitempty"`
	TokenUID   string `json:"tokenUid,omitempty"`
	ContractID string `json:"contractId,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	// StopRequested records that a stop was already asked for.
	StopRequested bool `json:"stopRequested,omitempty"`
	// Note is the outcome of the last executed action.
	Note string `json:"note,omitempty"`
}
