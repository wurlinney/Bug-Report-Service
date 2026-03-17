package policy

import "testing"

func TestCanUserViewReport_OwnerAllowed(t *testing.T) {
	if ok := CanUserViewReport("user", "u1", "u1"); !ok {
		t.Fatalf("expected owner to be allowed")
	}
}

func TestCanUserViewReport_NonOwnerDenied(t *testing.T) {
	if ok := CanUserViewReport("user", "u1", "u2"); ok {
		t.Fatalf("expected non-owner to be denied")
	}
}

func TestCanUserViewReport_ModeratorAllowed(t *testing.T) {
	if ok := CanUserViewReport("moderator", "u1", "u2"); !ok {
		t.Fatalf("expected moderator to be allowed")
	}
}

func TestCanModeratorChangeStatus_OnlyModerator(t *testing.T) {
	if ok := CanModeratorChangeStatus("moderator"); !ok {
		t.Fatalf("expected moderator to be allowed")
	}
	if ok := CanModeratorChangeStatus("user"); ok {
		t.Fatalf("expected user to be denied")
	}
}
