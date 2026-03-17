package bootstrap

import "testing"

func TestNewApp_DoesNotRequireDBOrJWTSecretYet(t *testing.T) {
	// At this stage NewApp should still work in a minimal mode
	// (health endpoints only) even without DATABASE_URL/JWT_SECRET.
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "")

	_, err := NewApp()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
