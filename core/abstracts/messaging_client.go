package abstracts

import "EV-Client-Simulator/core/entities"

type MessagingClient interface {
	GetId() string
	GetConn() any
	Listen(callsChannel chan entities.Message, resultsChannel chan entities.Message)
	Send(message entities.Message)
}
