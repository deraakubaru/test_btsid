package config

import (
	"os"
	"testing"
)

func TestLoadConfig_MissingDatabaseURL(t *testing.T) {
	os.Unsetenv("DATABASE_URL")
	os.Setenv("JWT_SECRET", "testsecret")
	defer os.Unsetenv("JWT_SECRET")

	cfg, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected error when DATABASE_URL is missing, got nil (cfg: %+v)", cfg)
	}
}

func TestLoadConfig_MissingJWTSecret(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost:5432/db")
	os.Unsetenv("JWT_SECRET")
	defer os.Unsetenv("DATABASE_URL")

	cfg, err := LoadConfig()
	if err == nil {
		t.Fatalf("expected error when JWT_SECRET is missing, got nil (cfg: %+v)", cfg)
	}
}

func TestLoadConfig_ValidConfig(t *testing.T) {
	os.Setenv("DATABASE_URL", "postgres://localhost:5432/testdb")
	os.Setenv("JWT_SECRET", "validsecret")
	os.Setenv("PORT", "9090")
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://example.com")
	os.Setenv("JWT_ACCESS_TTL_MINUTES", "30")
	os.Setenv("JWT_REFRESH_TTL_DAYS", "14")
	defer func() {
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("JWT_SECRET")
		os.Unsetenv("PORT")
		os.Unsetenv("CORS_ALLOWED_ORIGINS")
		os.Unsetenv("JWT_ACCESS_TTL_MINUTES")
		os.Unsetenv("JWT_REFRESH_TTL_DAYS")
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DatabaseURL != "postgres://localhost:5432/testdb" {
		t.Errorf("expected DatabaseURL 'postgres://localhost:5432/testdb', got '%s'", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "validsecret" {
		t.Errorf("expected JWTSecret 'validsecret', got '%s'", cfg.JWTSecret)
	}
	if cfg.Port != "9090" {
		t.Errorf("expected Port '9090', got '%s'", cfg.Port)
	}
	if cfg.CORSAllowedOrigins != "http://example.com" {
		t.Errorf("expected CORSAllowedOrigins 'http://example.com', got '%s'", cfg.CORSAllowedOrigins)
	}
	if cfg.JWTAccessTTLMinutes != 30 {
		t.Errorf("expected JWTAccessTTLMinutes 30, got %d", cfg.JWTAccessTTLMinutes)
	}
	if cfg.JWTRefreshTTLDays != 14 {
		t.Errorf("expected JWTRefreshTTLDays 14, got %d", cfg.JWTRefreshTTLDays)
	}
}
