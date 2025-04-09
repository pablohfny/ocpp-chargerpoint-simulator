package abstracts

import (
	"EV-Client-Simulator/app/domain/entities"
	"time"
)

type MessagingClient interface {
	GetId() string
	GetConn() any
	Listen(messagesChannel chan entities.Message)
	Send(message entities.Message, expectResult bool) error
	SendPeriodically(message entities.Message, expectResult bool, interval time.Duration) error
	Disconnect() error
}
