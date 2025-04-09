package factories

import (
	"EV-Client-Simulator/app/domain/entities"
)

func CreateAuthorizationCall(connectorId int, idTag string) entities.Message {
	return CreateCallMessage("Authorize",
		map[string]any{
			"idTag": idTag,
		},
		connectorId)
}
