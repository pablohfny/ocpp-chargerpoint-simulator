package http

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"EV-Client-Simulator/app/domain/abstracts"
)

// OCPIClient posts OCPI commands to a platform's CPO endpoints.
type OCPIClient struct {
	client *http.Client
}

// NewOCPIClient creates a client with a sane timeout for command dispatch.
func NewOCPIClient() *OCPIClient {
	return &OCPIClient{
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

// PostCommand sends a JSON body with an OCPI `Token` Authorization header.
func (c *OCPIClient) PostCommand(url, token string, payload interface{}) (*abstracts.OCPICommandResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &abstracts.OCPICommandResponse{StatusCode: resp.StatusCode, Body: responseBody}, nil
}
