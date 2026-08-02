package services

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"EV-Client-Simulator/app/domain/abstracts"
	"EV-Client-Simulator/app/domain/entities"
	"EV-Client-Simulator/app/domain/factories"
	"EV-Client-Simulator/app/domain/factories/utils"
)

// OCPICommandService issues OCPI commands on behalf of a partner profile and
// records them so inbound pushes can be correlated back to them.
type OCPICommandService struct {
	partners *OCPIPartnerService
	events   *OCPIEventService
	client   abstracts.OCPICommandClient
}

// NewOCPICommandService creates the command sender.
func NewOCPICommandService(partners *OCPIPartnerService, events *OCPIEventService, client abstracts.OCPICommandClient) *OCPICommandService {
	return &OCPICommandService{partners: partners, events: events, client: client}
}

// CommandDispatchResult describes the outcome of a dispatched command.
type CommandDispatchResult struct {
	CommandType string              `json:"commandType"`
	CommandUID  string              `json:"commandUid"`
	TokenUID    string              `json:"tokenUid,omitempty"`
	ContractID  string              `json:"contractId,omitempty"`
	ResponseURL string              `json:"responseUrl"`
	TargetURL   string              `json:"targetUrl"`
	StatusCode  int                 `json:"statusCode,omitempty"`
	Response    json.RawMessage     `json:"response,omitempty"`
	Event       *entities.OCPIEvent `json:"event,omitempty"`
}

// StartSessionOptions carries the target of a START_SESSION and the knobs used
// to exercise failure scenarios.
type StartSessionOptions struct {
	LocationID  string
	EvseUID     string
	ConnectorID string
	// AuthTokenOverride replaces the partner's own credential on the wire. It
	// exists so the cockpit can provoke the platform's 401 on purpose; empty
	// means "use the partner's real token".
	AuthTokenOverride string
}

// StartSession builds and dispatches a START_SESSION command for the partner.
func (s *OCPICommandService) StartSession(slug, locationID, evseUID, connectorID string) (*CommandDispatchResult, error) {
	return s.StartSessionWith(slug, StartSessionOptions{
		LocationID:  locationID,
		EvseUID:     evseUID,
		ConnectorID: connectorID,
	})
}

// StartSessionWith dispatches a START_SESSION with explicit options.
func (s *OCPICommandService) StartSessionWith(slug string, options StartSessionOptions) (*CommandDispatchResult, error) {
	partner, err := s.partners.Get(slug)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(options.LocationID) == "" {
		return nil, fmt.Errorf("locationId is required")
	}

	commandUID := utils.GenerateUUIDV4()
	tokenUID := factories.NewTokenUID(partner.PartyID, randomID(8))
	contractID := newContractID()

	command := factories.CreateStartSessionCommand(factories.StartSessionParams{
		Partner:     &partner,
		CommandUID:  commandUID,
		TokenUID:    tokenUID,
		ContractID:  contractID,
		LocationID:  options.LocationID,
		EvseUID:     options.EvseUID,
		ConnectorID: options.ConnectorID,
		Now:         nowUTC(),
	})

	authToken := partner.TokenToCallUs
	if options.AuthTokenOverride != "" {
		authToken = options.AuthTokenOverride
	}

	return s.dispatch(&partner, entities.CommandStartSession, commandUID, command, tokenUID, command.Token.Type, contractID, "", authToken)
}

// StopSession builds and dispatches a STOP_SESSION command for the partner.
func (s *OCPICommandService) StopSession(slug, sessionID string) (*CommandDispatchResult, error) {
	partner, err := s.partners.Get(slug)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("sessionId is required")
	}

	commandUID := utils.GenerateUUIDV4()
	command := factories.CreateStopSessionCommand(&partner, commandUID, sessionID)

	return s.dispatch(&partner, entities.CommandStopSession, commandUID, command, "", "", "", sessionID, partner.TokenToCallUs)
}

// dispatch posts a command and records the attempt as a command_sent event.
func (s *OCPICommandService) dispatch(
	partner *entities.OCPIPartner,
	commandType, commandUID string,
	payload interface{},
	tokenUID, tokenType, contractID, sessionID string,
	authToken string,
) (*CommandDispatchResult, error) {
	targetURL := factories.BuildCommandURL(partner, commandType)
	responseURL := factories.BuildResponseURL(partner, commandType, commandUID)

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	result := &CommandDispatchResult{
		CommandType: commandType,
		CommandUID:  commandUID,
		TokenUID:    tokenUID,
		ContractID:  contractID,
		ResponseURL: responseURL,
		TargetURL:   targetURL,
	}

	event := entities.OCPIEvent{
		PartnerSlug: partner.Slug,
		Direction:   entities.OCPIDirectionOut,
		Kind:        entities.OCPIEventCommandSent,
		Method:      "POST",
		Path:        targetURL,
		Body:        body,
		CommandUID:  commandUID,
		TokenUID:    tokenUID,
		TokenType:   tokenType,
		ContractID:  contractID,
		SessionID:   sessionID,
	}

	response, err := s.client.PostCommand(targetURL, authToken, payload)
	if err != nil {
		event.Error = err.Error()
		recorded := s.events.Record(event)
		result.Event = &recorded
		return result, nil
	}

	event.StatusCode = response.StatusCode
	result.StatusCode = response.StatusCode
	if json.Valid(response.Body) {
		result.Response = response.Body
	} else {
		result.Response, _ = json.Marshal(string(response.Body))
	}

	recorded := s.events.Record(event)
	result.Event = &recorded
	return result, nil
}

// nowUTC returns the current time in UTC.
func nowUTC() time.Time {
	return time.Now().UTC()
}

// newContractID generates the ad-hoc payment identifier carried end to end.
func newContractID() string {
	return "PAY" + strings.ToUpper(randomID(12))
}

// randomID returns a random hex string of the given length.
func randomID(length int) string {
	buffer := make([]byte, (length+1)/2)
	if _, err := rand.Read(buffer); err != nil {
		return strings.Repeat("0", length)
	}
	return hex.EncodeToString(buffer)[:length]
}
