package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"sync"
	"time"
)

const loginStateTTL = 10 * time.Minute

// PendingLogin holds PKCE state consumed after eSignet redirects back.
type PendingLogin struct {
	CodeVerifier string
	ReturnTo     string
	expiresAt    time.Time
}

// LoginStateStore holds short-lived PKCE state between /auth/login and /auth/callback.
type LoginStateStore struct {
	mu     sync.Mutex
	states map[string]PendingLogin
}

func NewLoginStateStore() *LoginStateStore {
	store := &LoginStateStore{states: make(map[string]PendingLogin)}
	go store.cleanupLoop()
	return store
}

func (s *LoginStateStore) Save(state, codeVerifier, returnTo string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[state] = PendingLogin{
		CodeVerifier: codeVerifier,
		ReturnTo:     returnTo,
		expiresAt:    time.Now().Add(loginStateTTL),
	}
}

func (s *LoginStateStore) Consume(state string) (PendingLogin, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.states[state]
	if !ok {
		return PendingLogin{}, false
	}
	delete(s.states, state)
	if time.Now().After(entry.expiresAt) {
		return PendingLogin{}, false
	}
	return entry, true
}

func (s *LoginStateStore) cleanupLoop() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for state, entry := range s.states {
			if now.After(entry.expiresAt) {
				delete(s.states, state)
			}
		}
		s.mu.Unlock()
	}
}

func GenerateRandomString(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func GenerateCodeChallenge(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func BuildAuthorizeURL(baseURL string, params map[string]string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid eSignet base URL: %w", err)
	}
	u.Path = "/authorize"
	q := u.Query()
	for key, value := range params {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}
