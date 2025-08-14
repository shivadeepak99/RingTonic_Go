package config

import (
	"os"
)

// the struct for Configurations !
type Config struct {
	Port             string
	DBPath           string
	StoragePath      string
	N8NWebhookURL    string
	N8NWebhookSecret string
	LogLevel         string
}

 
func Load() *Config {
	/* This contains the env setup , here it will return  the retrieved cred from .env or default value*/
	return &Config{
		Port:             getEnv("BACKEND_PORT", "8081"),
		DBPath:           getEnv("DB_PATH", "./data/ringtonic.db"),
		StoragePath:      getEnv("STORAGE_PATH", "./storage"),
		N8NWebhookURL:    getEnv("N8N_WEBHOOK_URL", "http://n8n:5678/webhook/ringtonic"),
		N8NWebhookSecret: getEnv("N8N_WEBHOOK_SECRET", "your-secure-secret-here"),
		LogLevel:         getEnv("LOG_LEVEL", "info"),
	}
}


func getEnv(key, defaultValue string) string {
	// getEnv returns the value of an environment variable or a default value
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
