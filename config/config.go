package config

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"

	"EV-Client-Simulator/app/domain/entities"
)

// Config holds application configuration
type Config struct {
	ServerAddr string
	ClientID   string
	HTTPPort   string

	// SimUser/SimPass protect the web UI and control APIs with HTTP basic auth.
	// When SimUser is empty, authentication is disabled (local development).
	SimUser string
	SimPass string

	// OCPIDataPath is the JSON file holding the OCPI partner profiles.
	OCPIDataPath string
	// SettingsPath is the JSON file holding the UI-editable settings. It sits
	// next to the partner profiles by default.
	SettingsPath string
	// OCPIPublicBaseURL is the default public URL of this simulator, used to
	// build response_url/receiver URLs for new partners.
	OCPIPublicBaseURL string
	// OCPIBaseURL is the default OCPI base URL new partners will command.
	OCPIBaseURL string
	// OCPIDefaultLocation/OCPIDefaultEVSE/OCPIDefaultConnector prefill the
	// START_SESSION form.
	OCPIDefaultLocation  string
	OCPIDefaultEVSE      string
	OCPIDefaultConnector string

	// BatteryCapacityKWh is the virtual EV battery size used to derive the
	// battery percentage shown on the simplified page.
	BatteryCapacityKWh float64
}

// Load loads configuration from flags and environment variables
func Load() *Config {
	config := &Config{}

	flag.StringVar(&config.ServerAddr, "serverAddr", getEnv("SERVER_ADDR", ""), "WebSocket server address")
	flag.StringVar(&config.ClientID, "clientId", getEnv("CLIENT_ID", "virtual"), "Client ID (default: virtual)")
	flag.StringVar(&config.HTTPPort, "httpPort", getEnv("HTTP_PORT", "8080"), "HTTP API port")
	flag.Parse()

	config.SimUser = getEnv("SIM_USER", "")
	config.SimPass = getEnv("SIM_PASS", "")
	config.OCPIDataPath = getEnv("OCPI_SIM_DATA", "./data/ocpi-partners.json")
	config.OCPIPublicBaseURL = getEnv("OCPI_SIM_PUBLIC_URL", "http://localhost:"+config.HTTPPort)
	config.OCPIBaseURL = getEnv("OCPI_SIM_BASE_URL", "https://ocpi-dev.nucharge.com.br")
	config.OCPIDefaultLocation = getEnv("OCPI_SIM_DEFAULT_LOCATION", "")
	config.OCPIDefaultEVSE = getEnv("OCPI_SIM_DEFAULT_EVSE", "")
	config.OCPIDefaultConnector = getEnv("OCPI_SIM_DEFAULT_CONNECTOR", "1")
	config.BatteryCapacityKWh = getEnvFloat("BATTERY_CAPACITY_KWH", 60)
	config.SettingsPath = getEnv("SIM_SETTINGS_FILE", filepath.Join(filepath.Dir(config.OCPIDataPath), "settings.json"))

	return config
}

// BootstrapSettings turns the environment into the defaults the settings
// service starts from. Anything persisted in the settings file wins over these.
func (c *Config) BootstrapSettings() entities.AppSettings {
	return entities.AppSettings{
		OCPIBaseURL:        c.OCPIBaseURL,
		PublicBaseURL:      c.OCPIPublicBaseURL,
		DefaultLocationID:  c.OCPIDefaultLocation,
		DefaultEvseUID:     c.OCPIDefaultEVSE,
		DefaultConnectorID: c.OCPIDefaultConnector,
		BatteryCapacityKWh: c.BatteryCapacityKWh,
	}
}

// AuthEnabled reports whether basic auth should be enforced.
func (c *Config) AuthEnabled() bool {
	return c.SimUser != ""
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultValue
}
