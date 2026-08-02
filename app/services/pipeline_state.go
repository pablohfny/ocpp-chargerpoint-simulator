package services

import (
	"fmt"
	"strings"

	"EV-Client-Simulator/app/domain/entities"
)

// pipelineEvidence is what the recorded OCPI events say about the current run.
// It is derived, never stored, so the cockpit always reflects reality.
type pipelineEvidence struct {
	CommandSent   bool
	CommandError  string
	CommandStatus int
	CommandResult string
	SessionSeen   bool
	SessionID     string
	CDRSeen       bool
	EchoMismatch  []string
	AuthFailed    bool
}

// failed reports whether the evidence already condemns the run.
func (e pipelineEvidence) failed() bool {
	return e.CommandError != "" ||
		e.CommandStatus >= 400 ||
		(e.CommandResult != "" && e.CommandResult != commandResultAccepted) ||
		len(e.EchoMismatch) > 0
}

// commandResultAccepted is the only CommandResult that keeps a run alive.
const commandResultAccepted = "ACCEPTED"

// chargerSnapshot is the slice of connector state the pipeline reasons about.
type chargerSnapshot struct {
	Found         bool
	Status        entities.ConnectorStatus
	CablePlugged  bool
	TransactionID int
	ErrorDetail   string
}

// collectEvidence folds the events of a run into the evidence the hops need.
// Only events newer than the run cursor are considered.
func collectEvidence(events []entities.OCPIEvent) pipelineEvidence {
	evidence := pipelineEvidence{}

	for _, event := range events {
		switch event.Kind {
		case entities.OCPIEventCommandSent:
			evidence.CommandSent = true
			evidence.CommandError = event.Error
			evidence.CommandStatus = event.StatusCode
		case entities.OCPIEventCommandResult:
			if result := ExtractCommandResult(event.Body); result != "" {
				evidence.CommandResult = strings.ToUpper(result)
			}
		case entities.OCPIEventSession:
			evidence.SessionSeen = true
			if event.SessionID != "" {
				evidence.SessionID = event.SessionID
			}
			applyEcho(&evidence, event)
		case entities.OCPIEventCDR:
			evidence.CDRSeen = true
			if evidence.SessionID == "" {
				evidence.SessionID = event.SessionID
			}
			applyEcho(&evidence, event)
		case entities.OCPIEventAuthFailed:
			evidence.AuthFailed = true
		}
	}

	return evidence
}

// applyEcho records a cdr_token echo mismatch reported by the event service.
func applyEcho(evidence *pipelineEvidence, event entities.OCPIEvent) {
	if event.EchoOK == nil || *event.EchoOK {
		return
	}
	evidence.EchoMismatch = event.EchoDiff
}

// deriveStage decides which phase the run is in. The charger state wins over
// the event evidence for everything the charger itself can prove.
func deriveStage(context entities.PipelineContext, evidence pipelineEvidence, charger chargerSnapshot, run entities.PipelineRun) entities.PipelineStage {
	if context.PartnerSlug == "" {
		return entities.StageIdle
	}
	if charger.Status == entities.StatusFaulted {
		return entities.StageFailed
	}
	if evidence.failed() {
		return entities.StageFailed
	}
	if evidence.CDRSeen {
		return entities.StageDone
	}

	switch charger.Status {
	case entities.StatusCharging:
		return entities.StageCharging
	case entities.StatusSuspendedEV, entities.StatusSuspendedEVSE:
		return entities.StageSuspended
	case entities.StatusFinishing:
		return entities.StageFinishing
	}

	if run.StopRequested {
		return entities.StageFinishing
	}
	if run.StartDispatched || evidence.CommandSent {
		return entities.StageStarting
	}
	if charger.CablePlugged || charger.Status == entities.StatusPreparing {
		return entities.StageStart
	}
	return entities.StagePlug
}

// buildHops paints the Parceiro → OCPI → Firebase → Carregador → CDR chain from
// the same evidence, then marks the first unfinished hop as the current one.
func buildHops(evidence pipelineEvidence, charger chargerSnapshot, stage entities.PipelineStage, run entities.PipelineRun) []entities.PipelineHop {
	expected := isNegativeVariant(run.Variant)

	hops := []entities.PipelineHop{
		partnerHop(evidence, run),
		ocpiHop(evidence, expected),
		platformHop(evidence, expected),
		chargerHop(charger, run),
		cdrHop(evidence),
	}

	if stage == entities.StageIdle || hasActiveHop(hops) {
		return hops
	}

	for index := range hops {
		if hops[index].Status == entities.HopPending {
			hops[index].Status = entities.HopActive
			break
		}
	}
	return hops
}

// hasActiveHop reports whether a hop already claimed the current position, as
// the charger does while it is busy.
func hasActiveHop(hops []entities.PipelineHop) bool {
	for _, hop := range hops {
		if hop.Status == entities.HopActive {
			return true
		}
	}
	return false
}

// partnerHop reports whether we managed to issue the command at all.
func partnerHop(evidence pipelineEvidence, run entities.PipelineRun) entities.PipelineHop {
	hop := entities.PipelineHop{ID: entities.HopPartner, Label: "Parceiro"}

	switch {
	case evidence.CommandError != "":
		hop.Status = entities.HopFailed
		hop.Error = evidence.CommandError
	case evidence.CommandSent:
		hop.Status = entities.HopConfirmed
		hop.Detail = "comando enviado"
	case run.Variant == entities.ActionStartLocal:
		hop.Status = entities.HopConfirmed
		hop.Detail = "ignorado (START local)"
	default:
		hop.Status = entities.HopPending
	}
	return hop
}

// ocpiHop reports how our ocpi-service replied to the command.
func ocpiHop(evidence pipelineEvidence, expected bool) entities.PipelineHop {
	hop := entities.PipelineHop{ID: entities.HopOCPI, Label: "OCPI"}

	switch {
	case evidence.CommandStatus == 0:
		hop.Status = entities.HopPending
	case evidence.CommandStatus >= 400:
		hop.Status = entities.HopFailed
		hop.Error = fmt.Sprintf("HTTP %d", evidence.CommandStatus)
		hop.Expected = expected
	default:
		hop.Status = entities.HopConfirmed
		hop.Detail = fmt.Sprintf("HTTP %d", evidence.CommandStatus)
	}
	return hop
}

// platformHop reports the async verdict our platform pushed back.
func platformHop(evidence pipelineEvidence, expected bool) entities.PipelineHop {
	hop := entities.PipelineHop{ID: entities.HopPlatform, Label: "Firebase"}

	switch {
	case evidence.CommandResult == commandResultAccepted:
		hop.Status = entities.HopConfirmed
		hop.Detail = "CommandResult ACCEPTED"
	case evidence.CommandResult != "":
		hop.Status = entities.HopFailed
		hop.Error = "CommandResult " + evidence.CommandResult
		hop.Expected = expected
	case evidence.SessionSeen:
		hop.Status = entities.HopConfirmed
		hop.Detail = "sessão recebida"
	default:
		hop.Status = entities.HopPending
	}
	return hop
}

// chargerHop mirrors the virtual charger's own connector state. It stays blue
// while the charger is busy and only turns green once the charge is over.
func chargerHop(charger chargerSnapshot, run entities.PipelineRun) entities.PipelineHop {
	hop := entities.PipelineHop{ID: entities.HopCharger, Label: "Carregador"}

	switch charger.Status {
	case entities.StatusFaulted:
		hop.Status = entities.HopFailed
		hop.Error = charger.ErrorDetail
	case entities.StatusCharging:
		hop.Status = entities.HopActive
		hop.Detail = "carregando"
	case entities.StatusSuspendedEV, entities.StatusSuspendedEVSE:
		hop.Status = entities.HopActive
		hop.Detail = "pausado"
	case entities.StatusPreparing:
		hop.Status = entities.HopActive
		hop.Detail = "preparando"
	case entities.StatusFinishing:
		hop.Status = entities.HopConfirmed
		hop.Detail = "finalizando"
	default:
		if run.StopRequested {
			hop.Status = entities.HopConfirmed
			hop.Detail = "recarga encerrada"
			return hop
		}
		hop.Status = entities.HopPending
	}
	return hop
}

// cdrHop closes the loop: the CDR came back and its cdr_token matched.
func cdrHop(evidence pipelineEvidence) entities.PipelineHop {
	hop := entities.PipelineHop{ID: entities.HopCDR, Label: "CDR"}

	switch {
	case len(evidence.EchoMismatch) > 0:
		hop.Status = entities.HopFailed
		hop.Error = "cdr_token divergente: " + strings.Join(evidence.EchoMismatch, ", ")
	case evidence.CDRSeen:
		hop.Status = entities.HopConfirmed
		hop.Detail = "cdr_token confere"
	default:
		hop.Status = entities.HopPending
	}
	return hop
}

// isNegativeVariant reports whether the run deliberately provoked a failure, so
// a red hop can be presented as a passing negative test.
func isNegativeVariant(variant entities.PipelineActionID) bool {
	return variant == entities.ActionStartInvalidAuth || variant == entities.ActionStartOccupied
}
