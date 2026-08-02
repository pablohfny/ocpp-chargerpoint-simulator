package dto

import "EV-Client-Simulator/app/domain/entities"

// RuntimeInfo holds the values fixed at boot. The Config tab shows them
// read-only so it is obvious what can and cannot be changed from the UI.
type RuntimeInfo struct {
	ServerAddr   string `json:"serverAddr"`
	ClientID     string `json:"clientId"`
	HTTPPort     string `json:"httpPort"`
	AuthEnabled  bool   `json:"authEnabled"`
	SettingsPath string `json:"settingsPath,omitempty"`
	PartnersPath string `json:"partnersPath,omitempty"`
}

// SettingsResponse is the Config tab payload: what is editable and what is not.
type SettingsResponse struct {
	Settings entities.AppSettings `json:"settings"`
	Runtime  RuntimeInfo          `json:"runtime"`
}
