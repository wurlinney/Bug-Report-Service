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

func (h fakeHasher) HashPassword(password string) (string, error) {
	return "hash:" + password, nil
}
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

func TestService_RegisterLoginRefresh_HappyPath(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	clk := fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}

	s := NewService(Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         clk,
		RefreshTTL:    30 * 24 * time.Hour,
	})

	ctx := context.Background()

	reg, err := s.Register(ctx, RegisterRequest{
		Email:    "a@example.com",
		Password: "P@ssw0rd!",
	})
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if reg.AccessToken == "" || reg.RefreshToken == "" || reg.RefreshTokenID == "" {
		t.Fatalf("expected tokens")
	}

	login, err := s.Login(ctx, LoginRequest{
		Email:    "a@example.com",
		Password: "P@ssw0rd!",
	})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if login.AccessToken == "" || login.RefreshToken == "" || login.RefreshTokenID == "" {
		t.Fatalf("expected tokens from login")
	}

	ref, err := s.Refresh(ctx, RefreshRequest{
		RefreshTokenID: login.RefreshTokenID,
		RefreshToken:   login.RefreshToken,
	})
	if err != nil {
		t.Fatalf("Refresh error: %v", err)
	}
	if ref.AccessToken == "" || ref.RefreshToken == "" || ref.RefreshTokenID == "" {
		t.Fatalf("expected rotated tokens from refresh")
	}
}

func TestService_Register_DuplicateEmail(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	_ = users.Create(context.Background(), ports.UserRecord{
		ID:           "u1",
		Email:        "a@example.com",
		PasswordHash: "x",
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	s := NewService(Deps{
		Users:         users,
		RefreshTokens: &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}},
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: time.Unix(1_700_000_000, 0).UTC()},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	_, err := s.Register(context.Background(), RegisterRequest{
		Email:    "a@example.com",
		Password: "P@ssw0rd!",
	})
	if !errors.Is(err, ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestService_Login_WrongPassword(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	_ = users.Create(context.Background(), ports.UserRecord{
		ID:           "u1",
		Email:        "a@example.com",
		PasswordHash: "hash:correct",
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	s := NewService(Deps{
		Users:         users,
		RefreshTokens: &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}},
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: time.Unix(1_700_000_000, 0).UTC()},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	_, err := s.Login(context.Background(), LoginRequest{
		Email:    "a@example.com",
		Password: "wrong",
	})
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_Refresh_InvalidToken(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}

	s := NewService(Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: time.Unix(1_700_000_000, 0).UTC()},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	_, err := s.Refresh(context.Background(), RefreshRequest{
		RefreshTokenID: "missing",
		RefreshToken:   "x",
	})
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken, got %v", err)
	}
}

func TestService_Refresh_PreservesRole(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	_ = users.Create(context.Background(), ports.UserRecord{
		ID:           "m1",
		Email:        "mod@example.com",
		PasswordHash: "hash:P@ssw0rd!",
		Role:         "moderator",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	})

	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	clk := fakeClock{t: time.Unix(1_700_000_000, 0).UTC()}

	s := NewService(Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         clk,
		RefreshTTL:    30 * 24 * time.Hour,
	})

	login, err := s.Login(context.Background(), LoginRequest{
		Email:    "mod@example.com",
		Password: "P@ssw0rd!",
	})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}

	ref, err := s.Refresh(context.Background(), RefreshRequest{
		RefreshTokenID: login.RefreshTokenID,
		RefreshToken:   login.RefreshToken,
	})
	if err != nil {
		t.Fatalf("Refresh error: %v", err)
	}
	if ref.AccessToken != "access:m1:moderator" {
		t.Fatalf("expected moderator role in access token, got %q", ref.AccessToken)
	}
}
