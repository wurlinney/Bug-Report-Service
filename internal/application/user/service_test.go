package user

import (
	"context"
	"testing"
	"time"

	"bug-report-service/internal/application/ports"
)

type memUsers struct {
	byID map[string]ports.UserRecord
}

func (m *memUsers) GetByEmail(_ context.Context, _ string) (ports.UserRecord, bool, error) {
	return ports.UserRecord{}, false, nil
}

func (m *memUsers) GetByID(_ context.Context, id string) (ports.UserRecord, bool, error) {
	u, ok := m.byID[id]
	return u, ok, nil
}

func (m *memUsers) Create(_ context.Context, _ ports.UserRecord) error { return nil }

func TestService_GetProfile(t *testing.T) {
	u := ports.UserRecord{
		ID:           "u1",
		Email:        "a@example.com",
		PasswordHash: "x",
		Role:         "user",
		CreatedAt:    time.Unix(100, 0).UTC(),
		UpdatedAt:    time.Unix(200, 0).UTC(),
	}
	svc := NewService(&memUsers{byID: map[string]ports.UserRecord{"u1": u}})

	p, err := svc.GetProfile(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ID != "u1" || p.Email != "a@example.com" || p.Role != "user" || p.CreatedAt != 100 || p.UpdatedAt != 200 {
		t.Fatalf("unexpected profile: %+v", p)
	}
}

func TestService_GetProfile_NotFound(t *testing.T) {
	svc := NewService(&memUsers{byID: map[string]ports.UserRecord{}})
	_, err := svc.GetProfile(context.Background(), "missing")
	if err != ErrUserNotFound {
		t.Fatalf("expected ErrUserNotFound, got %v", err)
	}
}
