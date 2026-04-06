package mod_profile

import (
	"context"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/user"
)

type mockUserGetter struct {
	user  user.User
	found bool
	err   error
}

func (m *mockUserGetter) GetByID(_ context.Context, _ string) (user.User, bool, error) {
	return m.user, m.found, m.err
}

func TestExecute_Success(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	uc := New(&mockUserGetter{
		user: user.User{
			ID:        "u1",
			Name:      "Mod",
			Email:     "mod@test.com",
			CreatedAt: now,
			UpdatedAt: now,
		},
		found: true,
	})

	profile, err := uc.Execute(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if profile.ID != "u1" {
		t.Errorf("expected ID u1, got %s", profile.ID)
	}
	if profile.Name != "Mod" {
		t.Errorf("expected Name Mod, got %s", profile.Name)
	}
	if profile.Email != "mod@test.com" {
		t.Errorf("expected Email mod@test.com, got %s", profile.Email)
	}
}

func TestExecute_Error_NotFound(t *testing.T) {
	uc := New(&mockUserGetter{found: false})

	_, err := uc.Execute(context.Background(), "missing")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
