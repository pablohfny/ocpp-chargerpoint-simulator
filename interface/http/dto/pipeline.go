package dto

// PipelineContextRequest arms a pipeline run against a partner and an EVSE.
type PipelineContextRequest struct {
	PartnerSlug string `json:"partnerSlug"`
	LocationID  string `json:"locationId"`
	EvseUID     string `json:"evseUid"`
	ConnectorID string `json:"connectorId"`
	// OCPPConnectorID is the local connector the virtual charger drives.
	OCPPConnectorID int `json:"ocppConnectorId"`
}

// PipelineActionRequest asks the orchestrator to run one action.
type PipelineActionRequest struct {
	Action string `json:"action"`
}
