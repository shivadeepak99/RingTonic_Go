package config

import (
	"os"
)

// Config holds all configuration for the application
type Config struct {
	Port             string
	DBPath           string
	StoragePath      string
	N8NWebhookURL    string
	N8NWebhookSecret string
	LogLevel         string
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		Port:             getEnv("BACKEND_PORT", "8080"),
		DBPath:           getEnv("DB_PATH", "./data/ringtonic.db"),
		StoragePath:      getEnv("STORAGE_PATH", "./storage"),
		N8NWebhookURL:    getEnv("N8N_WEBHOOK_URL", "http://n8n:5678/webhook/ringtonic"),
		N8NWebhookSecret: getEnv("N8N_WEBHOOK_SECRET", "your-secure-secret-here"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
	}
}

// getEnv returns the value of an environment variable or a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
