package config

import "testing"

func TestLoad_AuthAndDBDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_ISSUER", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_ACCESS_TTL", "")
	t.Setenv("JWT_REFRESH_TTL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.DB.URL != "" {
		t.Fatalf("expected empty DB URL default")
	}
	if cfg.JWT.Issuer == "" {
		t.Fatalf("expected default issuer")
	}
	if cfg.JWT.AccessTTL <= 0 || cfg.JWT.RefreshTTL <= 0 {
		t.Fatalf("expected positive JWT ttls")
	}
}
