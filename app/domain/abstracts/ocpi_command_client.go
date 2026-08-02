package abstracts

// OCPICommandResponse captures what a platform replied to an OCPI command.
type OCPICommandResponse struct {
	StatusCode int
	Body       []byte
}

// OCPICommandClient sends OCPI commands to a platform's CPO endpoints.
type OCPICommandClient interface {
	PostCommand(url, token string, payload interface{}) (*OCPICommandResponse, error)
}
