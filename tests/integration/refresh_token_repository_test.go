//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/application/ports"
)

func TestPostgresRefreshTokenRepository_SaveGetRevoke(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	users := postgres.NewUserRepository(db)
	tokens := postgres.NewRefreshTokenRepository(db)

	now := time.Unix(1_700_000_000, 0).UTC()
	if err := users.Create(context.Background(), ports.UserRecord{
		ID:           "u1",
		Email:        "a@example.com",
		PasswordHash: "x",
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	rt := ports.RefreshTokenRecord{
		ID:        "rt1",
		UserID:    "u1",
		Role:      "user",
		TokenHash: "h1",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}
	if err := tokens.Save(context.Background(), rt); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	got, found, err := tokens.GetActiveByID(context.Background(), "rt1")
	if err != nil {
		t.Fatalf("GetActiveByID error: %v", err)
	}
	if !found {
		t.Fatalf("expected found")
	}
	if got.UserID != "u1" || got.TokenHash != "h1" || got.Role != "user" {
		t.Fatalf("unexpected token: %+v", got)
	}

	revokeAt := now.Add(1 * time.Hour)
	if err := tokens.Revoke(context.Background(), "rt1", revokeAt); err != nil {
		t.Fatalf("Revoke error: %v", err)
	}

	_, found, err = tokens.GetActiveByID(context.Background(), "rt1")
	if err != nil {
		t.Fatalf("GetActiveByID after revoke error: %v", err)
	}
	if found {
		t.Fatalf("expected not found after revoke")
	}
}
