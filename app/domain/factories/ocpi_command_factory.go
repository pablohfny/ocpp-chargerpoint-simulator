package factories

import (
	"fmt"
	"time"

	"EV-Client-Simulator/app/domain/entities"
)

// StartSessionParams carries everything needed to build a START_SESSION body.
// The identifiers are passed in rather than generated here so the factory stays
// deterministic and testable.
type StartSessionParams struct {
	Partner     *entities.OCPIPartner
	CommandUID  string
	TokenUID    string
	ContractID  string
	LocationID  string
	EvseUID     string
	ConnectorID string
	Now         time.Time
}

// BuildResponseURL builds the callback URL our platform must POST the async
// CommandResult to.
func BuildResponseURL(partner *entities.OCPIPartner, commandType, commandUID string) string {
	return fmt.Sprintf("%s/ocpi/p/%s/commands/%s/%s", partner.PublicBaseURL, partner.Slug, commandType, commandUID)
}

// BuildCommandURL builds the CPO endpoint we POST the command to.
func BuildCommandURL(partner *entities.OCPIPartner, commandType string) string {
	return fmt.Sprintf("%s/ocpi/cpo/2.2.1/commands/%s", partner.OCPIBaseURL, commandType)
}

// CreateStartSessionCommand builds an OCPI 2.2.1 START_SESSION command for an
// ad-hoc (payment terminal style) token.
func CreateStartSessionCommand(params StartSessionParams) entities.StartSessionCommand {
	return entities.StartSessionCommand{
		ResponseURL: BuildResponseURL(params.Partner, entities.CommandStartSession, params.CommandUID),
		Token: entities.OCPIToken{
			CountryCode: params.Partner.CountryCode,
			PartyID:     params.Partner.PartyID,
			UID:         params.TokenUID,
			Type:        "AD_HOC_USER",
			ContractID:  params.ContractID,
			Issuer:      params.Partner.Name,
			Valid:       true,
			Whitelist:   "ALLOWED_OFFLINE",
			LastUpdated: params.Now.UTC(),
		},
		LocationID:  params.LocationID,
		EvseUID:     params.EvseUID,
		ConnectorID: params.ConnectorID,
	}
}

// CreateStopSessionCommand builds an OCPI 2.2.1 STOP_SESSION command.
func CreateStopSessionCommand(partner *entities.OCPIPartner, commandUID, sessionID string) entities.StopSessionCommand {
	return entities.StopSessionCommand{
		ResponseURL: BuildResponseURL(partner, entities.CommandStopSession, commandUID),
		SessionID:   sessionID,
	}
}

// NewTokenUID builds the token uid a partner presents, namespaced by party id.
func NewTokenUID(partyID, random string) string {
	return fmt.Sprintf("%s_%s", partyID, random)
}
