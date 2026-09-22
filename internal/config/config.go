package config

import (
	"errors"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// Config holds all configuration settings for the application.
type Config struct {
	Port                string
	DatabaseURL         string
	JWTSecret           string
	JWTAccessTTLMinutes int
	JWTRefreshTTLDays   int
	CORSAllowedOrigins  string
}

// LoadConfig loads configuration from environment variables, optionally reading from a .env file.
// It fails fast if required environment variables (DATABASE_URL, JWT_SECRET) are missing or empty.
func LoadConfig() (*Config, error) {
	// Optionally load .env file for local development if present
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is required and cannot be empty")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET is required and cannot be empty")
	}

	port := getEnvOrDefault("PORT", "8080")
	corsOrigins := getEnvOrDefault("CORS_ALLOWED_ORIGINS", "*")
	accessTTLMinutes := getEnvAsIntOrDefault("JWT_ACCESS_TTL_MINUTES", 15)
	refreshTTLDays := getEnvAsIntOrDefault("JWT_REFRESH_TTL_DAYS", 7)

	cfg := &Config{
		Port:                port,
		DatabaseURL:         dbURL,
		JWTSecret:           jwtSecret,
		JWTAccessTTLMinutes: accessTTLMinutes,
		JWTRefreshTTLDays:   refreshTTLDays,
		CORSAllowedOrigins:  corsOrigins,
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsIntOrDefault(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
