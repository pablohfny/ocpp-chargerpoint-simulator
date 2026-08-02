package entities

import "testing"

func TestOCPIPartnerMatchesAuthorization(t *testing.T) {
	tests := []struct {
		name           string
		inputExpected  string
		inputHeader    string
		expectedResult bool
	}{
		{
			name:           "exact token matches",
			inputExpected:  "abc123",
			inputHeader:    "Token abc123",
			expectedResult: true,
		},
		{
			name:           "token with special characters matches verbatim",
			inputExpected:  "Zm9vOmJhcg==",
			inputHeader:    "Token Zm9vOmJhcg==",
			expectedResult: true,
		},
		{
			name:           "missing header is rejected",
			inputExpected:  "abc123",
			inputHeader:    "",
			expectedResult: false,
		},
		{
			name:           "wrong token is rejected",
			inputExpected:  "abc123",
			inputHeader:    "Token abc124",
			expectedResult: false,
		},
		{
			name:           "missing Token prefix is rejected",
			inputExpected:  "abc123",
			inputHeader:    "abc123",
			expectedResult: false,
		},
		{
			name:           "lowercase scheme is rejected",
			inputExpected:  "abc123",
			inputHeader:    "token abc123",
			expectedResult: false,
		},
		{
			name:           "Bearer scheme is rejected",
			inputExpected:  "abc123",
			inputHeader:    "Bearer abc123",
			expectedResult: false,
		},
		{
			name:           "extra whitespace is rejected",
			inputExpected:  "abc123",
			inputHeader:    "Token  abc123",
			expectedResult: false,
		},
		{
			name:           "trailing whitespace is rejected",
			inputExpected:  "abc123",
			inputHeader:    "Token abc123 ",
			expectedResult: false,
		},
		{
			name:           "case differences are rejected",
			inputExpected:  "abc123",
			inputHeader:    "Token ABC123",
			expectedResult: false,
		},
		{
			name:           "partner without expected token rejects everything",
			inputExpected:  "",
			inputHeader:    "Token ",
			expectedResult: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockPartner := OCPIPartner{TokenExpected: test.inputExpected}

			actualResult := mockPartner.MatchesAuthorization(test.inputHeader)

			if actualResult != test.expectedResult {
				t.Errorf("MatchesAuthorization(%q) = %v, expected %v", test.inputHeader, actualResult, test.expectedResult)
			}
		})
	}
}

func TestOCPIPartnerValidate(t *testing.T) {
	validPartner := func() OCPIPartner {
		return OCPIPartner{
			Slug:          "nayax-sim",
			Name:          "Nayax Simulator",
			PartyID:       "NYX",
			CountryCode:   "BR",
			TokenExpected: "token-a",
			OCPIBaseURL:   "https://ocpi-dev.nucharge.com.br",
			PublicBaseURL: "https://sim-dev.nucharge.com.br",
		}
	}

	tests := []struct {
		name        string
		mutate      func(*OCPIPartner)
		expectError bool
	}{
		{name: "valid profile", mutate: func(p *OCPIPartner) {}, expectError: false},
		{name: "slug with underscore", mutate: func(p *OCPIPartner) { p.Slug = "nayax_sim" }, expectError: true},
		{name: "slug with uppercase", mutate: func(p *OCPIPartner) { p.Slug = "Nayax" }, expectError: true},
		{name: "slug with slash", mutate: func(p *OCPIPartner) { p.Slug = "nayax/sim" }, expectError: true},
		{name: "empty slug", mutate: func(p *OCPIPartner) { p.Slug = "" }, expectError: true},
		{name: "empty name", mutate: func(p *OCPIPartner) { p.Name = "" }, expectError: true},
		{name: "short party id", mutate: func(p *OCPIPartner) { p.PartyID = "NY" }, expectError: true},
		{name: "long party id", mutate: func(p *OCPIPartner) { p.PartyID = "NYXX" }, expectError: true},
		{name: "bad country code", mutate: func(p *OCPIPartner) { p.CountryCode = "BRA" }, expectError: true},
		{name: "missing expected token", mutate: func(p *OCPIPartner) { p.TokenExpected = "" }, expectError: true},
		{name: "missing ocpi base url", mutate: func(p *OCPIPartner) { p.OCPIBaseURL = "" }, expectError: true},
		{name: "missing public base url", mutate: func(p *OCPIPartner) { p.PublicBaseURL = "" }, expectError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mockPartner := validPartner()
			test.mutate(&mockPartner)

			actualError := mockPartner.Validate()

			if test.expectError && actualError == nil {
				t.Errorf("expected a validation error, got nil")
			}
			if !test.expectError && actualError != nil {
				t.Errorf("expected no validation error, got %v", actualError)
			}
		})
	}
}

func TestOCPIPartnerNormalize(t *testing.T) {
	inputPartner := OCPIPartner{
		Slug:          "  Nayax-Sim  ",
		PartyID:       " nyx ",
		CountryCode:   " br ",
		OCPIBaseURL:   " https://ocpi-dev.nucharge.com.br/ ",
		PublicBaseURL: " https://sim-dev.nucharge.com.br/ ",
	}

	inputPartner.Normalize()

	expected := OCPIPartner{
		Slug:          "nayax-sim",
		PartyID:       "NYX",
		CountryCode:   "BR",
		OCPIBaseURL:   "https://ocpi-dev.nucharge.com.br",
		PublicBaseURL: "https://sim-dev.nucharge.com.br",
	}
	if inputPartner != expected {
		t.Errorf("Normalize() = %+v, expected %+v", inputPartner, expected)
	}
}

func TestDeriveBatteryPercent(t *testing.T) {
	tests := []struct {
		name            string
		inputSoc        int16
		inputMeterWh    float64
		inputCapacity   float64
		expectedPercent int
	}{
		{name: "soc wins while charging", inputSoc: 42, inputMeterWh: 4200, inputCapacity: 60, expectedPercent: 42},
		{name: "soc is clamped at 100", inputSoc: 120, inputMeterWh: 0, inputCapacity: 60, expectedPercent: 100},
		{name: "idle with no energy is zero", inputSoc: 0, inputMeterWh: 0, inputCapacity: 60, expectedPercent: 0},
		{name: "falls back to meter over capacity", inputSoc: 0, inputMeterWh: 30000, inputCapacity: 60, expectedPercent: 50},
		{name: "meter fallback is clamped at 100", inputSoc: 0, inputMeterWh: 90000, inputCapacity: 60, expectedPercent: 100},
		{name: "zero capacity uses the default", inputSoc: 0, inputMeterWh: 30000, inputCapacity: 0, expectedPercent: 50},
		{name: "negative meter is treated as empty", inputSoc: 0, inputMeterWh: -5, inputCapacity: 60, expectedPercent: 0},
		{name: "small capacity charges fast", inputSoc: 0, inputMeterWh: 5000, inputCapacity: 10, expectedPercent: 50},
		{name: "rounds to nearest percent", inputSoc: 0, inputMeterWh: 1000, inputCapacity: 60, expectedPercent: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualPercent := DeriveBatteryPercent(test.inputSoc, test.inputMeterWh, test.inputCapacity)

			if actualPercent != test.expectedPercent {
				t.Errorf("DeriveBatteryPercent(%d, %v, %v) = %d, expected %d",
					test.inputSoc, test.inputMeterWh, test.inputCapacity, actualPercent, test.expectedPercent)
			}
		})
	}
}
