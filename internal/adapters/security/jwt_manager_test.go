package security

import (
	"testing"
	"time"
)

func TestJWTManager_IssueAndVerifyAccessToken(t *testing.T) {
	m := NewJWTManager(JWTConfig{
		Issuer:        "bug-report-service",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    30 * 24 * time.Hour,
		HMACSecretKey: []byte("supersecret-supersecret-supersecret"),
		Now:           func() time.Time { return time.Unix(1_700_000_000, 0).UTC() },
	})

	access, err := m.IssueAccessToken("u1")
	if err != nil {
		t.Fatalf("IssueAccessToken error: %v", err)
	}
	if access == "" {
		t.Fatalf("expected non-empty token")
	}

	claims, err := m.VerifyAccessToken(access)
	if err != nil {
		t.Fatalf("VerifyAccessToken error: %v", err)
	}
	if claims.Subject != "u1" {
		t.Fatalf("expected sub=u1, got %q", claims.Subject)
	}
	if claims.Role != "moderator" {
		t.Fatalf("expected role=moderator, got %q", claims.Role)
	}
	if claims.Issuer != "bug-report-service" {
		t.Fatalf("expected issuer, got %q", claims.Issuer)
	}
	if !claims.ExpiresAt.After(time.Unix(1_700_000_000, 0).UTC()) {
		t.Fatalf("expected exp after now")
	}
}

func TestJWTManager_ExpiredTokenRejected(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()
	m := NewJWTManager(JWTConfig{
		Issuer:        "bug-report-service",
		AccessTTL:     1 * time.Second,
		RefreshTTL:    30 * 24 * time.Hour,
		HMACSecretKey: []byte("supersecret-supersecret-supersecret"),
		Now:           func() time.Time { return now },
	})

	access, err := m.IssueAccessToken("u1")
	if err != nil {
		t.Fatalf("IssueAccessToken error: %v", err)
	}

	m2 := NewJWTManager(JWTConfig{
		Issuer:        "bug-report-service",
		AccessTTL:     1 * time.Second,
		RefreshTTL:    30 * 24 * time.Hour,
		HMACSecretKey: []byte("supersecret-supersecret-supersecret"),
		Now:           func() time.Time { return now.Add(2 * time.Second) },
	})

	_, err = m2.VerifyAccessToken(access)
	if err == nil {
		t.Fatalf("expected expired token error")
	}
}
