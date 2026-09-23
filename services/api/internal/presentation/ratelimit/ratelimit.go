package ratelimit

import (
	"sync"
	"time"
)

type Limiter struct {
	mu          sync.Mutex
	perClient   int
	total       int
	window      time.Duration
	now         func() time.Time
	windowStart time.Time
	counts      map[string]int
	used        int
}

func New(perClient, total int, window time.Duration, now func() time.Time) *Limiter {
	return &Limiter{perClient: perClient, total: total, window: window, now: now, counts: map[string]int{}}
}

func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if now := l.now(); now.Sub(l.windowStart) >= l.window {
		l.windowStart, l.counts, l.used = now, map[string]int{}, 0
	}
	if l.used >= l.total || l.counts[key] >= l.perClient {
		return false
	}
	l.counts[key]++
	l.used++
	return true
}
