package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/application/ports"
)

type memUsers struct {
	byEmail map[string]ports.UserRecord
	byID    map[string]ports.UserRecord
}

func (m *memUsers) GetByEmail(_ context.Context, email string) (ports.UserRecord, bool, error) {
	u, ok := m.byEmail[email]
	return u, ok, nil
}
func (m *memUsers) GetByID(_ context.Context, id string) (ports.UserRecord, bool, error) {
	u, ok := m.byID[id]
	return u, ok, nil
}
func (m *memUsers) Create(_ context.Context, u ports.UserRecord) error {
	if _, exists := m.byEmail[u.Email]; exists {
		return ports.ErrUniqueViolation
	}
	m.byEmail[u.Email] = u
	m.byID[u.ID] = u
	return nil
}

type memRefresh struct {
	byTokenID map[string]ports.RefreshTokenRecord
}

func (m *memRefresh) Save(_ context.Context, rt ports.RefreshTokenRecord) error {
	m.byTokenID[rt.ID] = rt
	return nil
}
func (m *memRefresh) GetActiveByID(_ context.Context, id string) (ports.RefreshTokenRecord, bool, error) {
	rt, ok := m.byTokenID[id]
	if !ok || rt.RevokedAt != nil {
		return ports.RefreshTokenRecord{}, false, nil
	}
	return rt, true, nil
}
func (m *memRefresh) Revoke(_ context.Context, id string, when time.Time) error {
	rt, ok := m.byTokenID[id]
	if !ok {
		return nil
	}
	rt.RevokedAt = &when
	m.byTokenID[id] = rt
	return nil
}

type fakeHasher struct{}

func (h fakeHasher) HashPassword(password string) (string, error) { return "hash:" + password, nil }
func (h fakeHasher) VerifyPassword(hash string, password string) (bool, error) {
	return hash == "hash:"+password, nil
}

type fakeJWT struct{}

func (j fakeJWT) IssueAccessToken(userID string, role string) (string, error) {
	return "access:" + userID + ":" + role, nil
}

type fakeRandom struct{}

func (r fakeRandom) NewID() string { return "id-1" }
func (r fakeRandom) NewToken() (string, error) {
	return "refresh-secret", nil
}

type fakeClock struct{ t time.Time }

func (c fakeClock) Now() time.Time { return c.t }

func TestService_LoginAndRefresh_ModeratorHappyPath(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	users := &memUsers{byEmail: map[string]ports.UserRecord{
		"mod@example.com": {
			ID:           "m1",
			Name:         "Alice Moderator",
			Email:        "mod@example.com",
			PasswordHash: "hash:P@ssw0rd!",
			Role:         "moderator",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	s := NewService(Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	login, err := s.Login(context.Background(), LoginRequest{
		Email:    "mod@example.com",
		Password: "P@ssw0rd!",
	})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if login.AccessToken != "access:m1:moderator" {
		t.Fatalf("unexpected access token: %s", login.AccessToken)
	}

	ref, err := s.Refresh(context.Background(), RefreshRequest{
		RefreshTokenID: login.RefreshTokenID,
		RefreshToken:   login.RefreshToken,
	})
	if err != nil {
		t.Fatalf("Refresh error: %v", err)
	}
	if ref.AccessToken != "access:m1:moderator" {
		t.Fatalf("unexpected refreshed token: %s", ref.AccessToken)
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	users := &memUsers{byEmail: map[string]ports.UserRecord{
		"mod@example.com": {
			ID:           "m1",
			Name:         "Alice Moderator",
			Email:        "mod@example.com",
			PasswordHash: "hash:correct",
			Role:         "moderator",
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	}, byID: map[string]ports.UserRecord{}}
	s := NewService(Deps{
		Users:         users,
		RefreshTokens: &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}},
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	_, err := s.Login(context.Background(), LoginRequest{Email: "mod@example.com", Password: "wrong"})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
