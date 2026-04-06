package mod_refresh

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"bug-report-service/internal/domain"
	"bug-report-service/internal/domain/user"
)

type mockRefreshTokenStore struct {
	token   user.RefreshToken
	found   bool
	getErr  error
	saveID  string
	saveErr error
}

func (m *mockRefreshTokenStore) GetActiveByID(_ context.Context, _ string) (user.RefreshToken, bool, error) {
	return m.token, m.found, m.getErr
}

func (m *mockRefreshTokenStore) Revoke(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (m *mockRefreshTokenStore) Save(_ context.Context, rt user.RefreshToken) (user.RefreshToken, error) {
	if m.saveErr != nil {
		return user.RefreshToken{}, m.saveErr
	}
	out := rt
	out.ID = m.saveID
	return out, nil
}

type mockTokenIssuerR struct {
	token string
	err   error
}

func (m *mockTokenIssuerR) IssueAccessToken(_ string) (string, error) {
	return m.token, m.err
}

type mockRandomR struct {
	token string
	err   error
}

func (m *mockRandomR) NewToken() (string, error) {
	return m.token, m.err
}

type mockClockR struct {
	now time.Time
}

func (m *mockClockR) Now() time.Time { return m.now }

func hash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestExecute_Success(t *testing.T) {
	now := time.Now()
	secret := "refresh-secret"

	store := &mockRefreshTokenStore{
		token: user.RefreshToken{
			ID:        "old-rt",
			UserID:    "u1",
			TokenHash: hash(secret),
			ExpiresAt: now.Add(time.Hour),
		},
		found:  true,
		saveID: "new-rt",
	}

	uc := New(
		store,
		&mockTokenIssuerR{token: "new-access"},
		&mockRandomR{token: "new-refresh-secret"},
		&mockClockR{now: now},
		24*time.Hour,
	)

	resp, err := uc.Execute(context.Background(), Request{
		RefreshTokenID: "old-rt",
		RefreshToken:   secret,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken != "new-access" {
		t.Errorf("expected new-access, got %s", resp.AccessToken)
	}
	if resp.RefreshTokenID != "new-rt" {
		t.Errorf("expected new-rt, got %s", resp.RefreshTokenID)
	}
	if resp.RefreshToken != "new-refresh-secret" {
		t.Errorf("expected new-refresh-secret, got %s", resp.RefreshToken)
	}
}

func TestExecute_Error_EmptyTokenID(t *testing.T) {
	uc := New(
		&mockRefreshTokenStore{},
		&mockTokenIssuerR{},
		&mockRandomR{},
		&mockClockR{now: time.Now()},
		24*time.Hour,
	)

	_, err := uc.Execute(context.Background(), Request{
		RefreshTokenID: "",
		RefreshToken:   "some-token",
	})
	if !errors.Is(err, domain.ErrInvalidRefresh) {
		t.Fatalf("expected ErrInvalidRefresh, got %v", err)
	}
}

func TestExecute_Error_TokenNotFound(t *testing.T) {
	uc := New(
		&mockRefreshTokenStore{found: false},
		&mockTokenIssuerR{},
		&mockRandomR{},
		&mockClockR{now: time.Now()},
		24*time.Hour,
	)

	_, err := uc.Execute(context.Background(), Request{
		RefreshTokenID: "missing",
		RefreshToken:   "tok",
	})
	if !errors.Is(err, domain.ErrInvalidRefresh) {
		t.Fatalf("expected ErrInvalidRefresh, got %v", err)
	}
}

func TestExecute_Error_ExpiredToken(t *testing.T) {
	now := time.Now()
	store := &mockRefreshTokenStore{
		token: user.RefreshToken{
			ID:        "rt-1",
			UserID:    "u1",
			TokenHash: hash("secret"),
			ExpiresAt: now.Add(-time.Hour), // expired
		},
		found: true,
	}

	uc := New(
		store,
		&mockTokenIssuerR{},
		&mockRandomR{},
		&mockClockR{now: now},
		24*time.Hour,
	)

	_, err := uc.Execute(context.Background(), Request{
		RefreshTokenID: "rt-1",
		RefreshToken:   "secret",
	})
	if !errors.Is(err, domain.ErrInvalidRefresh) {
		t.Fatalf("expected ErrInvalidRefresh, got %v", err)
	}
}
