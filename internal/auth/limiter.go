package auth

import (
	"sync"
	"time"
)

type LoginLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	failures map[string]loginFailure
}

type loginFailure struct {
	Count      int
	FirstSeen  time.Time
	LockedTill time.Time
}

func NewLoginLimiter(limit int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		limit:    limit,
		window:   window,
		failures: map[string]loginFailure{},
	}
}

func (l *LoginLimiter) Allow(key string) bool {
	now := time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	failure, ok := l.failures[key]
	if !ok {
		return true
	}
	if failure.LockedTill.After(now) {
		return false
	}
	if now.Sub(failure.FirstSeen) > l.window {
		delete(l.failures, key)
		return true
	}
	return true
}

func (l *LoginLimiter) RecordFailure(key string) {
	now := time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	failure := l.failures[key]
	if failure.FirstSeen.IsZero() || now.Sub(failure.FirstSeen) > l.window {
		failure = loginFailure{FirstSeen: now}
	}
	failure.Count++
	if failure.Count >= l.limit {
		failure.LockedTill = now.Add(l.window)
	}
	l.failures[key] = failure
}

func (l *LoginLimiter) Clear(key string) {
	l.mu.Lock()
	delete(l.failures, key)
	l.mu.Unlock()
}
