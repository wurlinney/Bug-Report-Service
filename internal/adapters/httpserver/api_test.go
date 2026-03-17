package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bug-report-service/internal/application/auth"
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

type fakeVerifier struct{}

func (v fakeVerifier) VerifyAccessToken(token string) (Principal, error) {
	if token == "access:id-1:user" {
		return Principal{UserID: "id-1", Role: "user"}, nil
	}
	return Principal{}, ErrUnauthorized
}

func TestAPI_RegisterThenMe(t *testing.T) {
	users := &memUsers{byEmail: map[string]ports.UserRecord{}, byID: map[string]ports.UserRecord{}}
	refresh := &memRefresh{byTokenID: map[string]ports.RefreshTokenRecord{}}
	now := time.Unix(1_700_000_000, 0).UTC()

	authSvc := auth.NewService(auth.Deps{
		Users:         users,
		RefreshTokens: refresh,
		Hasher:        fakeHasher{},
		JWT:           fakeJWT{},
		Random:        fakeRandom{},
		Clock:         fakeClock{t: now},
		RefreshTTL:    30 * 24 * time.Hour,
	})

	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		AuthService:   authSvc,
		TokenVerifier: fakeVerifier{},
	})

	// register
	body, _ := json.Marshal(map[string]any{
		"email":    "a@example.com",
		"password": "P@ssw0rd!",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var reg struct {
		AccessToken    string `json:"access_token"`
		RefreshTokenID string `json:"refresh_token_id"`
		RefreshToken   string `json:"refresh_token"`
	}
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)
	if reg.AccessToken == "" || reg.RefreshTokenID == "" || reg.RefreshToken == "" {
		t.Fatalf("expected tokens, got %+v", reg)
	}

	// me
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req2.Header.Set("Authorization", "Bearer "+reg.AccessToken)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var me struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	_ = json.Unmarshal(rr2.Body.Bytes(), &me)
	if me.ID != "id-1" || me.Role != "user" {
		t.Fatalf("unexpected me: %+v", me)
	}
}

func TestAPI_Me_UnauthorizedWithoutToken(t *testing.T) {
	h := NewAPI(Deps{
		Ready:         NewReadiness(),
		AuthService:   nil,
		TokenVerifier: fakeVerifier{},
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}
