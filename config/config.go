package config

import (
	"flag"
	"os"
)

// Config holds application configuration
type Config struct {
	ServerAddr string
	ClientID   string
	HTTPPort   string
}

// Load loads configuration from flags and environment variables
func Load() *Config {
	config := &Config{}

	flag.StringVar(&config.ServerAddr, "serverAddr", getEnv("SERVER_ADDR", ""), "WebSocket server address")
	flag.StringVar(&config.ClientID, "clientId", getEnv("CLIENT_ID", "virtual"), "Client ID (default: virtual)")
	flag.StringVar(&config.HTTPPort, "httpPort", getEnv("HTTP_PORT", "8080"), "HTTP API port")
	flag.Parse()

	return config
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
