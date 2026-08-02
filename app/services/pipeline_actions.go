package services

import "EV-Client-Simulator/app/domain/entities"

// stageLabels are the pt-BR names the cockpit shows for each stage.
var stageLabels = map[entities.PipelineStage]string{
	entities.StageIdle:      "Sem contexto",
	entities.StagePlug:      "Plugar o cabo",
	entities.StageStart:     "Iniciar recarga",
	entities.StageStarting:  "Aguardando início",
	entities.StageCharging:  "Carregando",
	entities.StageSuspended: "Pausado",
	entities.StageFinishing: "Encerrando",
	entities.StageDone:      "Concluído",
	entities.StageFailed:    "Falhou",
}

// stageHints explain, in one line, what the cockpit is waiting for.
var stageHints = map[entities.PipelineStage]string{
	entities.StageIdle:      "Escolha o parceiro, a estação e o EVSE para armar o contexto.",
	entities.StagePlug:      "Plugue o cabo no carregador virtual para liberar o início.",
	entities.StageStart:     "Dispare o START pelo parceiro ou escolha uma variação de cenário.",
	entities.StageStarting:  "START despachado. Aguardando a plataforma e o carregador responderem.",
	entities.StageCharging:  "Recarga em andamento. Você pode pausar, injetar falha ou encerrar.",
	entities.StageSuspended: "Recarga pausada. Retome ou encerre a sessão.",
	entities.StageFinishing: "Recarga encerrada. Aguardando o CDR chegar no parceiro.",
	entities.StageDone:      "Ciclo completo: CDR recebido e cdr_token conferido.",
	entities.StageFailed:    "O fluxo parou. Veja o passo em vermelho para o motivo.",
}

// stageActions is the closed set of actions valid in each stage. The contextual
// action bar renders exactly this list, so an action that is not here can never
// be executed.
var stageActions = map[entities.PipelineStage][]entities.PipelineActionID{
	entities.StageIdle: {},
	entities.StagePlug: {
		entities.ActionPlug,
		entities.ActionStartLocal,
		entities.ActionReset,
	},
	entities.StageStart: {
		entities.ActionStartPartner,
		entities.ActionStartInvalidAuth,
		entities.ActionStartOccupied,
		entities.ActionStartLocal,
		entities.ActionUnplug,
		entities.ActionReset,
	},
	entities.StageStarting: {
		entities.ActionStopRemote,
		entities.ActionStopLocal,
		entities.ActionUnplug,
		entities.ActionWaitTimeout,
		entities.ActionReset,
	},
	entities.StageCharging: {
		entities.ActionStopRemote,
		entities.ActionStopLocal,
		entities.ActionSuspendEV,
		entities.ActionSuspendEVSE,
		entities.ActionFault,
		entities.ActionSendMeter,
		entities.ActionUnplug,
		entities.ActionReset,
	},
	entities.StageSuspended: {
		entities.ActionResume,
		entities.ActionStopRemote,
		entities.ActionStopLocal,
		entities.ActionFault,
		entities.ActionUnplug,
		entities.ActionReset,
	},
	entities.StageFinishing: {
		entities.ActionUnplug,
		entities.ActionSendMeter,
		entities.ActionWaitTimeout,
		entities.ActionReset,
	},
	entities.StageDone: {
		entities.ActionReset,
	},
	entities.StageFailed: {
		entities.ActionReset,
		entities.ActionClearFault,
		entities.ActionUnplug,
	},
}

// stagePrimaryAction is the action the cockpit emphasises in each stage.
var stagePrimaryAction = map[entities.PipelineStage]entities.PipelineActionID{
	entities.StagePlug:      entities.ActionPlug,
	entities.StageStart:     entities.ActionStartPartner,
	entities.StageStarting:  entities.ActionStopRemote,
	entities.StageCharging:  entities.ActionStopRemote,
	entities.StageSuspended: entities.ActionResume,
	entities.StageFinishing: entities.ActionUnplug,
	entities.StageDone:      entities.ActionReset,
	entities.StageFailed:    entities.ActionReset,
}

// actionCatalog holds the presentation of every action, independent of stage.
var actionCatalog = map[entities.PipelineActionID]entities.PipelineAction{
	entities.ActionPlug: {
		ID: entities.ActionPlug, Label: "Plugar cabo",
		Hint: "Conecta o cabo no conector do carregador virtual.",
	},
	entities.ActionUnplug: {
		ID: entities.ActionUnplug, Label: "Desplugar",
		Hint: "Remove o cabo — encerra a recarga pelo lado do carregador.",
	},
	entities.ActionStartPartner: {
		ID: entities.ActionStartPartner, Label: "START via parceiro",
		Hint: "Envia START_SESSION OCPI com a credencial real do parceiro.",
	},
	entities.ActionStartInvalidAuth: {
		ID: entities.ActionStartInvalidAuth, Label: "START token inválido",
		Hint:        "Envia START_SESSION com Authorization errado: esperamos 401 da plataforma.",
		Destructive: true,
	},
	entities.ActionStartOccupied: {
		ID: entities.ActionStartOccupied, Label: "START EVSE ocupado",
		Hint:        "Ocupa o EVSE com uma recarga local e então envia o START: esperamos rejeição.",
		Destructive: true,
	},
	entities.ActionStartLocal: {
		ID: entities.ActionStartLocal, Label: "START local (OCPP)",
		Hint: "Inicia pelo carregador (plug, Authorize, StartTransaction), sem passar pelo parceiro.",
	},
	entities.ActionStopRemote: {
		ID: entities.ActionStopRemote, Label: "STOP remoto (OCPI)",
		Hint: "Envia STOP_SESSION OCPI para a sessão em curso.",
	},
	entities.ActionStopLocal: {
		ID: entities.ActionStopLocal, Label: "Parar no carregador",
		Hint: "Encerra a transação pelo lado do carregador (StopTransaction).",
	},
	entities.ActionSuspendEV: {
		ID: entities.ActionSuspendEV, Label: "Suspender EV",
		Hint: "Carro para de puxar energia (SuspendedEV).",
	},
	entities.ActionSuspendEVSE: {
		ID: entities.ActionSuspendEVSE, Label: "Suspender EVSE",
		Hint: "Carregador para de entregar energia (SuspendedEVSE).",
	},
	entities.ActionResume: {
		ID: entities.ActionResume, Label: "Retomar recarga",
		Hint: "Volta do estado pausado para Charging.",
	},
	entities.ActionFault: {
		ID: entities.ActionFault, Label: "Injetar falha",
		Hint:        "Coloca o conector em Faulted com EVCommunicationError.",
		Destructive: true,
	},
	entities.ActionClearFault: {
		ID: entities.ActionClearFault, Label: "Limpar falha",
		Hint: "Tira o conector de Faulted.",
	},
	entities.ActionSendMeter: {
		ID: entities.ActionSendMeter, Label: "Enviar meter",
		Hint: "Dispara um MeterValues avulso para a transação atual.",
	},
	entities.ActionWaitTimeout: {
		ID: entities.ActionWaitTimeout, Label: "Deixar dar timeout",
		Hint: "Não envia nada: serve para observar o timeout da plataforma.",
	},
	entities.ActionReset: {
		ID: entities.ActionReset, Label: "Nova rodada",
		Hint: "Zera o acompanhamento e mantém o contexto para um novo teste.",
	},
}

// actionsFor returns the actions valid in a stage, already decorated for the
// UI. The primary action follows the run: without an OCPI session id there is
// nothing to STOP remotely, so the local stop is emphasised instead.
func actionsFor(stage entities.PipelineStage, run entities.PipelineRun, charger chargerSnapshot) []entities.PipelineAction {
	ids := stageActions[stage]
	primary := stagePrimaryAction[stage]

	if primary == entities.ActionStopRemote && run.SessionID == "" {
		primary = entities.ActionStopLocal
	}

	actions := make([]entities.PipelineAction, 0, len(ids))
	for _, id := range ids {
		if !appliesToCharger(id, charger) {
			continue
		}
		action := actionCatalog[id]
		action.Primary = id == primary
		actions = append(actions, action)
	}
	return actions
}

// appliesToCharger drops the actions the connector cannot honour right now, so
// the bar never offers a button that is guaranteed to fail.
func appliesToCharger(action entities.PipelineActionID, charger chargerSnapshot) bool {
	if !charger.Found {
		return true
	}

	switch action {
	case entities.ActionClearFault:
		return charger.Status == entities.StatusFaulted
	case entities.ActionUnplug:
		return charger.CablePlugged
	default:
		return true
	}
}

// isActionAllowed reports whether an action may run in the given stage.
func isActionAllowed(stage entities.PipelineStage, action entities.PipelineActionID) bool {
	for _, id := range stageActions[stage] {
		if id == action {
			return true
		}
	}
	return false
}
