package config

import (
	"os"
	"testing"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("RATE_LIMIT_RPS", "")
	t.Setenv("RATE_LIMIT_BURST", "")
	t.Setenv("HTTP_READ_TIMEOUT", "")
	t.Setenv("HTTP_WRITE_TIMEOUT", "")
	t.Setenv("HTTP_IDLE_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HTTP.Addr == "" {
		t.Fatalf("expected default addr, got empty")
	}
	if cfg.RateLimit.RPS <= 0 || cfg.RateLimit.Burst <= 0 {
		t.Fatalf("expected positive rate limit defaults")
	}
}

func TestLoad_ValidatesRateLimit(t *testing.T) {
	_ = os.Setenv("RATE_LIMIT_RPS", "0")
	_ = os.Setenv("RATE_LIMIT_BURST", "0")
	t.Cleanup(func() {
		_ = os.Unsetenv("RATE_LIMIT_RPS")
		_ = os.Unsetenv("RATE_LIMIT_BURST")
	})

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for non-positive rate limit")
	}
}

func TestLoad_ProdRequiresSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("S3_ACCESS_KEY", "")
	t.Setenv("S3_SECRET_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error in prod when required vars are empty")
	}
}
