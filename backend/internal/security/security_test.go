package security

import "testing"

func TestPasswordHash(t *testing.T) {
	hash, e := HashPassword("strong-password")
	if e != nil {
		t.Fatal(e)
	}
	if hash == "strong-password" {
		t.Fatal("password stored as plaintext")
	}
	if !CheckPassword(hash, "strong-password") {
		t.Fatal("valid password rejected")
	}
	if CheckPassword(hash, "wrong-password") {
		t.Fatal("invalid password accepted")
	}
}
func TestTokenIsRandom(t *testing.T) {
	a, b := Token(), Token()
	if len(a) != 64 || len(b) != 64 {
		t.Fatal("unexpected token length")
	}
	if a == b {
		t.Fatal("tokens must be unique")
	}
}
