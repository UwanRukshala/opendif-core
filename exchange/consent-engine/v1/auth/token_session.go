package auth

import (
	"sync"
	"time"
)

const tokenSessionTTL = 2 * time.Minute

// TokenSession holds tokens briefly between backend callback and portal fetch.
type TokenSession struct {
	AccessToken string
	IDToken     string
	TokenType   string
	ExpiresIn   int
	expiresAt   time.Time
}

// TokenSessionStore holds one-time token sessions for the portal to fetch.
type TokenSessionStore struct {
	mu       sync.Mutex
	sessions map[string]TokenSession
}

func NewTokenSessionStore() *TokenSessionStore {
	store := &TokenSessionStore{sessions: make(map[string]TokenSession)}
	go store.cleanupLoop()
	return store
}

func (s *TokenSessionStore) Save(sessionID string, session TokenSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session.expiresAt = time.Now().Add(tokenSessionTTL)
	s.sessions[sessionID] = session
}

// Get returns a token session without deleting it. Sessions expire via cleanup.
func (s *TokenSessionStore) Get(sessionID string) (TokenSession, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.sessions[sessionID]
	if !ok {
		return TokenSession{}, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.sessions, sessionID)
		return TokenSession{}, false
	}
	return entry, true
}

func (s *TokenSessionStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionID)
}

func (s *TokenSessionStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.mu.Lock()
		for id, entry := range s.sessions {
			if now.After(entry.expiresAt) {
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}
