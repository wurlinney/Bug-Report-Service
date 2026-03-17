//go:build integration

package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/adapters/persistence/postgres"
	"bug-report-service/internal/application/ports"
)

func TestPostgresUserRepository_CreateAndGetByEmail(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)

	repo := postgres.NewUserRepository(db)
	now := time.Unix(1_700_000_000, 0).UTC()

	u := ports.UserRecord{
		ID:           "u1",
		Email:        "a@example.com",
		PasswordHash: "hash",
		Role:         "user",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := repo.Create(context.Background(), u); err != nil {
		t.Fatalf("Create error: %v", err)
	}

	got, found, err := repo.GetByEmail(context.Background(), "a@example.com")
	if err != nil {
		t.Fatalf("GetByEmail error: %v", err)
	}
	if !found {
		t.Fatalf("expected found")
	}
	if got.ID != "u1" || got.Email != "a@example.com" || got.Role != "user" {
		t.Fatalf("unexpected user: %+v", got)
	}
}

func TestPostgresUserRepository_UniqueEmail(t *testing.T) {
	db := mustDB(t)
	ensureSchema(t, db)
	repo := postgres.NewUserRepository(db)

	now := time.Now().UTC()
	u1 := ports.UserRecord{ID: "u1", Email: "a@example.com", PasswordHash: "x", Role: "user", CreatedAt: now, UpdatedAt: now}
	u2 := ports.UserRecord{ID: "u2", Email: "a@example.com", PasswordHash: "y", Role: "user", CreatedAt: now, UpdatedAt: now}

	if err := repo.Create(context.Background(), u1); err != nil {
		t.Fatalf("Create#1 error: %v", err)
	}
	err := repo.Create(context.Background(), u2)
	if !errors.Is(err, ports.ErrUniqueViolation) {
		t.Fatalf("expected ErrUniqueViolation, got %v", err)
	}
}
