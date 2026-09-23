package http

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rateLimiter struct {
	mu          sync.Mutex
	perClient   int
	total       int
	window      time.Duration
	now         func() time.Time
	windowStart time.Time
	counts      map[string]int
	used        int
}

func newRateLimiter(perClient, total int, window time.Duration, now func() time.Time) *rateLimiter {
	return &rateLimiter{perClient: perClient, total: total, window: window, now: now, counts: map[string]int{}}
}

func (l *rateLimiter) allow(key string) bool {
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

func clientKey(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
