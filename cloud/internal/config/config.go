package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the cloud server.
type Config struct {
	// Server
	Port int    // HTTP server port (default: 9000)
	Env  string // "development" or "production"

	// CORS
	AllowedOrigins []string // Allowed origins for admin routes

	// WebSocket
	MaxWSMessageSize int64 // Max WebSocket message size in bytes (default: 65536)

	// Database
	DatabaseURL string // PostgreSQL connection string

	// NATS
	NatsURL string // NATS server URL

	// Logging
	LogLevel string // "debug", "info", "warn", "error"

	// Cloud WebSocket URL for provisioning files
	CloudWSURL string // default: wss://cloud.tablehub.com/ws

	// Security
	ZitadelJWKSURL string

	// Auth Provider
	AuthProvider string // "zitadel", "logto", "mock" (default: "zitadel")
	JWKSURL      string // Generic JWKS URL

	// Zitadel Management API
	ZitadelAPIURL         string // ZITADEL_API_URL (default: http://localhost:8088)
	ZitadelMachineKeyPath string // ZITADEL_MACHINE_KEY_PATH (path to zitadel-admin.json)
	ZitadelProjectID      string // ZITADEL_PROJECT_ID
	ZitadelOrgID          string // ZITADEL_ORG_ID

	// Logto Management API
	LogtoAPIURL            string // LOGTO_API_URL (default: http://localhost:3002)
	LogtoApplicationID     string // LOGTO_APP_ID
	LogtoApplicationSecret string // LOGTO_APP_SECRET
	LogtoOrgID             string // LOGTO_ORG_ID
}

// Load reads configuration from environment variables with sensible defaults.
func Load() (*Config, error) {
	authProvider := getEnv("AUTH_PROVIDER", "zitadel")
	defaultJWKSURL := "http://localhost:8088/oauth/v2/keys"
	if authProvider == "logto" {
		defaultJWKSURL = "http://localhost:3002/oidc/jwks"
	}

	cfg := &Config{
		Port:                   getEnvInt("PORT", 9000),
		Env:                    getEnv("ENV", "development"),
		AllowedOrigins:         getEnvSlice("ALLOWED_ORIGINS", []string{"*"}),
		MaxWSMessageSize:       getEnvInt64("MAX_WS_MESSAGE_SIZE", 65536),
		DatabaseURL:            getEnv("DATABASE_URL", "postgres://tablehub:tablehub@localhost:5432/tablehub_cloud?sslmode=disable"),
		NatsURL:                getEnv("NATS_URL", "nats://localhost:4222"),
		LogLevel:               getEnv("LOG_LEVEL", "debug"),
		CloudWSURL:             getEnv("CLOUD_WS_URL", "wss://cloud.tablehub.com/ws"),
		ZitadelJWKSURL:         getEnv("ZITADEL_JWKS_URL", "http://localhost:8088/oauth/v2/keys"),
		AuthProvider:           authProvider,
		JWKSURL:                getEnv("JWKS_URL", defaultJWKSURL),
		ZitadelAPIURL:          getEnv("ZITADEL_API_URL", "http://localhost:8088"),
		ZitadelMachineKeyPath:  getEnv("ZITADEL_MACHINE_KEY_PATH", ""),
		ZitadelProjectID:       getEnv("ZITADEL_PROJECT_ID", ""),
		ZitadelOrgID:           getEnv("ZITADEL_ORG_ID", ""),
		LogtoAPIURL:            getEnv("LOGTO_API_URL", "http://localhost:3002"),
		LogtoApplicationID:     getEnv("LOGTO_APP_ID", ""),
		LogtoApplicationSecret: getEnv("LOGTO_APP_SECRET", ""),
		LogtoOrgID:             getEnv("LOGTO_ORG_ID", ""),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("PORT must be between 1 and 65535, got %d", c.Port)
	}
	if c.Env != "development" && c.Env != "production" {
		return fmt.Errorf("ENV must be 'development' or 'production', got '%s'", c.Env)
	}
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	// Validate Machine Key Path if specified
	if c.ZitadelMachineKeyPath != "" {
		if _, err := os.Stat(c.ZitadelMachineKeyPath); os.IsNotExist(err) {
			return fmt.Errorf("ZITADEL_MACHINE_KEY_PATH file not found: %s", c.ZitadelMachineKeyPath)
		}
	}
	return nil
}

// IsDev returns true if running in development mode.
func (c *Config) IsDev() bool {
	return c.Env == "development"
}

// Addr returns the address string for the HTTP server (e.g., ":9000").
func (c *Config) Addr() string {
	return fmt.Sprintf(":%d", c.Port)
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvInt64(key string, fallback int64) int64 {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvSlice(key string, fallback []string) []string {
	val, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return fallback
	}
	parts := strings.Split(val, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
