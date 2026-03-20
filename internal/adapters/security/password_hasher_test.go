package security

import "testing"

func TestPasswordHasher_HashAndVerify(t *testing.T) {
	h := NewBCryptPasswordHasher(12)

	hash, err := h.HashPassword("P@ssw0rd!")
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" || hash == "P@ssw0rd!" {
		t.Fatalf("expected non-empty hash different from password")
	}

	ok, err := h.VerifyPassword(hash, "P@ssw0rd!")
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if !ok {
		t.Fatalf("expected password to match hash")
	}

	ok, err = h.VerifyPassword(hash, "wrong")
	if err != nil {
		t.Fatalf("VerifyPassword error: %v", err)
	}
	if ok {
		t.Fatalf("expected wrong password to not match hash")
	}
}
