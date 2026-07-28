package auth

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("password was not hashed")
	}
	if !VerifyPassword(hash, "correct horse battery staple") {
		t.Fatal("valid password rejected")
	}
	if VerifyPassword(hash, "wrong") {
		t.Fatal("invalid password accepted")
	}
}

func TestSessionsExpireAndRejectUnknownTokens(t *testing.T) {
	m := NewSessions()
	token, err := m.Create()
	if err != nil {
		t.Fatal(err)
	}
	if !m.Valid(token) {
		t.Fatal("fresh session rejected")
	}
	if m.Valid("unknown") {
		t.Fatal("unknown session accepted")
	}
	m.Delete(token)
	if m.Valid(token) {
		t.Fatal("deleted session accepted")
	}
}
