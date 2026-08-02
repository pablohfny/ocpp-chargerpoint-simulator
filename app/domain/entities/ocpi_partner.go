package entities

import (
	"errors"
	"regexp"
	"strings"
)

// slugPattern restricts partner slugs to url-safe identifiers.
var slugPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// OCPIPartner represents an OCPI 2.2.1 counter-party (eMSP/PTP) we impersonate.
type OCPIPartner struct {
	// Slug is the url-safe identifier used in the receiver routes.
	Slug string `json:"slug"`
	// Name is the human readable partner name, also used as the token issuer.
	Name string `json:"name"`
	// PartyID is the 3 character OCPI party identifier.
	PartyID string `json:"partyId"`
	// CountryCode is the 2 character OCPI country code.
	CountryCode string `json:"countryCode"`
	// TokenToCallUs is the Authorization token this partner SENDS when
	// commanding our ocpi-service.
	TokenToCallUs string `json:"tokenToCallUs"`
	// TokenExpected is the Authorization token our platform MUST send when
	// pushing data to this partner's receiver endpoints.
	TokenExpected string `json:"tokenExpected"`
	// OCPIBaseURL is the base URL of the platform this partner commands.
	OCPIBaseURL string `json:"ocpiBaseUrl"`
	// PublicBaseURL is this simulator's own public URL, used to build the
	// response_url and receiver URLs handed to the platform.
	PublicBaseURL string `json:"publicBaseUrl"`
}

// Validate checks the partner profile for required and well formed fields.
func (p *OCPIPartner) Validate() error {
	if !slugPattern.MatchString(p.Slug) {
		return errors.New("slug must be lowercase alphanumeric with dashes")
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}
	if len(p.PartyID) != 3 {
		return errors.New("partyId must be exactly 3 characters")
	}
	if len(p.CountryCode) != 2 {
		return errors.New("countryCode must be exactly 2 characters")
	}
	if strings.TrimSpace(p.TokenExpected) == "" {
		return errors.New("tokenExpected is required")
	}
	if strings.TrimSpace(p.OCPIBaseURL) == "" {
		return errors.New("ocpiBaseUrl is required")
	}
	if strings.TrimSpace(p.PublicBaseURL) == "" {
		return errors.New("publicBaseUrl is required")
	}
	return nil
}

// Normalize applies OCPI casing conventions to the identity fields.
func (p *OCPIPartner) Normalize() {
	p.Slug = strings.ToLower(strings.TrimSpace(p.Slug))
	p.PartyID = strings.ToUpper(strings.TrimSpace(p.PartyID))
	p.CountryCode = strings.ToUpper(strings.TrimSpace(p.CountryCode))
	p.OCPIBaseURL = strings.TrimRight(strings.TrimSpace(p.OCPIBaseURL), "/")
	p.PublicBaseURL = strings.TrimRight(strings.TrimSpace(p.PublicBaseURL), "/")
}

// MatchesAuthorization reports whether the given Authorization header value is
// exactly the `Token {tokenExpected}` this partner requires. The comparison is
// verbatim: no base64 decoding and no case folding, so a platform sending an
// encoded or differently cased token is correctly rejected.
func (p *OCPIPartner) MatchesAuthorization(header string) bool {
	if p.TokenExpected == "" {
		return false
	}
	return header == "Token "+p.TokenExpected
}
