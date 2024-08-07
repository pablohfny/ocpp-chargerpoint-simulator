package abstracts

type MessagingClient interface {
	Listen()
	Send()
}
