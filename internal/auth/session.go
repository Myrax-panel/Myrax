package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sync"
	"time"
)

type Session struct {
	Username  string
	ExpiresAt time.Time
}

type SessionManager struct {
	mu       sync.Mutex
	secret   []byte
	ttl      time.Duration
	sessions map[string]Session
}

func NewSessionManager(encodedSecret string, ttl time.Duration) (*SessionManager, error) {
	secret, err := base64.RawStdEncoding.DecodeString(encodedSecret)
	if err != nil || len(secret) < 32 {
		return nil, fmt.Errorf("invalid session secret")
	}
	return &SessionManager{
		secret:   secret,
		ttl:      ttl,
		sessions: map[string]Session{},
	}, nil
}

func GenerateSecret() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base64.RawStdEncoding.EncodeToString(secret), nil
}

func (m *SessionManager) Create(username string) (string, Session, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", Session{}, err
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	session := Session{
		Username:  username,
		ExpiresAt: time.Now().UTC().Add(m.ttl),
	}
	m.mu.Lock()
	m.sessions[m.tokenKey(token)] = session
	m.mu.Unlock()
	return token, session, nil
}

func (m *SessionManager) Get(token string) (Session, bool) {
	if token == "" {
		return Session{}, false
	}
	key := m.tokenKey(token)
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[key]
	if !ok {
		return Session{}, false
	}
	if !session.ExpiresAt.After(now) {
		delete(m.sessions, key)
		return Session{}, false
	}
	return session, true
}

func (m *SessionManager) Delete(token string) {
	if token == "" {
		return
	}
	m.mu.Lock()
	delete(m.sessions, m.tokenKey(token))
	m.mu.Unlock()
}

func (m *SessionManager) tokenKey(token string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(token))
	return base64.RawStdEncoding.EncodeToString(mac.Sum(nil))
}
