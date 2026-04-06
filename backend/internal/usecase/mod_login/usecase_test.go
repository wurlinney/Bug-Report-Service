package mod_login

import (
	"context"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/user"
)

type mockUserFinder struct {
	user  user.User
	found bool
	err   error
}

func (m *mockUserFinder) GetByEmail(_ context.Context, _ string) (user.User, bool, error) {
	return m.user, m.found, m.err
}

type mockPasswordVerifier struct {
	ok  bool
	err error
}

func (m *mockPasswordVerifier) VerifyPassword(_ string, _ string) (bool, error) {
	return m.ok, m.err
}

type mockTokenIssuer struct {
	token string
	err   error
}

func (m *mockTokenIssuer) IssueAccessToken(_ string) (string, error) {
	return m.token, m.err
}

type mockRefreshTokenSaver struct {
	saved user.RefreshToken
	err   error
}

func (m *mockRefreshTokenSaver) Save(_ context.Context, rt user.RefreshToken) (user.RefreshToken, error) {
	if m.err != nil {
		return user.RefreshToken{}, m.err
	}
	out := rt
	out.ID = m.saved.ID
	return out, nil
}

type mockRandomGenerator struct {
	token string
	err   error
}

func (m *mockRandomGenerator) NewToken() (string, error) {
	return m.token, m.err
}

type mockClock struct {
	now time.Time
}

func (m *mockClock) Now() time.Time { return m.now }

func newTestUseCase(
	users *mockUserFinder,
	hasher *mockPasswordVerifier,
) *UseCase {
	return New(
		users,
		hasher,
		&mockTokenIssuer{token: "access-tok"},
		&mockRefreshTokenSaver{saved: user.RefreshToken{ID: "rt-1"}},
		&mockRandomGenerator{token: "refresh-secret"},
		&mockClock{now: time.Now()},
		24*time.Hour,
	)
}

func TestExecute_Success(t *testing.T) {
	uc := newTestUseCase(
		&mockUserFinder{
			user:  user.User{ID: "u1", Email: "mod@test.com", PasswordHash: "hash"},
			found: true,
		},
		&mockPasswordVerifier{ok: true},
	)

	resp, err := uc.Execute(context.Background(), Request{
		Email:    "mod@test.com",
		Password: "pass123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "access-tok" {
		t.Errorf("expected access-tok, got %s", resp.AccessToken)
	}
	if resp.RefreshTokenID != "rt-1" {
		t.Errorf("expected rt-1, got %s", resp.RefreshTokenID)
	}
	if resp.RefreshToken != "refresh-secret" {
		t.Errorf("expected refresh-secret, got %s", resp.RefreshToken)
	}
}

func TestExecute_Error_UserNotFound(t *testing.T) {
	uc := newTestUseCase(
		&mockUserFinder{found: false},
		&mockPasswordVerifier{ok: true},
	)

	_, err := uc.Execute(context.Background(), Request{
		Email:    "nobody@test.com",
		Password: "pass",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestExecute_Error_WrongPassword(t *testing.T) {
	uc := newTestUseCase(
		&mockUserFinder{
			user:  user.User{ID: "u1", PasswordHash: "hash"},
			found: true,
		},
		&mockPasswordVerifier{ok: false},
	)

	_, err := uc.Execute(context.Background(), Request{
		Email:    "mod@test.com",
		Password: "wrong",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
