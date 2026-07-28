package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

const iterations = 310000

func derive(password string, salt []byte, rounds int) []byte {
	mac := hmac.New(sha256.New, []byte(password))
	mac.Write(salt)
	mac.Write([]byte{0, 0, 0, 1})
	u := mac.Sum(nil)
	out := append([]byte(nil), u...)
	for i := 1; i < rounds; i++ {
		mac.Reset()
		mac.Write(u)
		u = mac.Sum(nil)
		for j := range out {
			out[j] ^= u[j]
		}
	}
	return out
}

func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", fmt.Errorf("password must be at least 12 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := derive(password, salt, iterations)
	enc := base64.RawStdEncoding
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iterations, enc.EncodeToString(salt), enc.EncodeToString(hash)), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	rounds, err := strconv.Atoi(parts[1])
	if err != nil || rounds < 100000 || rounds > 1000000 {
		return false
	}
	enc := base64.RawStdEncoding
	salt, err := enc.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := enc.DecodeString(parts[3])
	if err != nil {
		return false
	}
	actual := derive(password, salt, rounds)
	return len(actual) == len(expected) && subtle.ConstantTimeCompare(actual, expected) == 1
}

type Sessions struct {
	mu     sync.Mutex
	tokens map[string]time.Time
	ttl    time.Duration
}

func NewSessions() *Sessions { return &Sessions{tokens: map[string]time.Time{}, ttl: 24 * time.Hour} }
func (s *Sessions) Create() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	s.mu.Lock()
	s.tokens[token] = time.Now().Add(s.ttl)
	s.mu.Unlock()
	return token, nil
}
func (s *Sessions) Valid(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expiry, ok := s.tokens[token]
	if !ok {
		return false
	}
	if time.Now().After(expiry) {
		delete(s.tokens, token)
		return false
	}
	return true
}
func (s *Sessions) Delete(token string) { s.mu.Lock(); delete(s.tokens, token); s.mu.Unlock() }
