// Package security provides security middleware for the HTTP API:
// rate limiting, brute-force protection, and IP-based throttling.
package security

import (
	"net/http"
	"sync"
	"time"
)

// bucket is a token-bucket rate limiter for a single key (IP address).
type bucket struct {
	tokens   float64
	lastSeen time.Time
	mu       sync.Mutex
}

// allow consumes one token and returns true if the request is permitted.
// Tokens refill at rate r per second up to burst capacity.
func (b *bucket) allow(rate, burst float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens += elapsed * rate
	if b.tokens > burst {
		b.tokens = burst
	}
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// IPRateLimiter enforces per-IP request limits.
// It periodically cleans stale entries to prevent memory growth.
type IPRateLimiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	burst    float64 // maximum burst size
	ttl      time.Duration
	stopOnce sync.Once
	stop     chan struct{}
}

// NewIPRateLimiter creates a limiter with the given fill rate and burst size.
// rate = sustained requests per second, burst = peak allowed per window.
func NewIPRateLimiter(rate, burst float64) *IPRateLimiter {
	l := &IPRateLimiter{
		buckets: make(map[string]*bucket),
		rate:    rate,
		burst:   burst,
		ttl:     5 * time.Minute,
		stop:    make(chan struct{}),
	}
	go l.cleanup()
	return l
}

// Allow returns false if the IP has exceeded its rate limit.
func (l *IPRateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: l.burst, lastSeen: time.Now()}
		l.buckets[ip] = b
	}
	l.mu.Unlock()
	return b.allow(l.rate, l.burst)
}

// cleanup removes stale buckets every minute to prevent unbounded memory growth.
func (l *IPRateLimiter) cleanup() {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			l.mu.Lock()
			for ip, b := range l.buckets {
				b.mu.Lock()
				stale := time.Since(b.lastSeen) > l.ttl
				b.mu.Unlock()
				if stale {
					delete(l.buckets, ip)
				}
			}
			l.mu.Unlock()
		case <-l.stop:
			return
		}
	}
}

// Stop releases background goroutine resources.
func (l *IPRateLimiter) Stop() {
	l.stopOnce.Do(func() { close(l.stop) })
}

// RateLimit returns HTTP middleware that rejects requests exceeding the
// configured rate. Responds 429 Too Many Requests with a Retry-After header.
func RateLimit(limiter *IPRateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIPFromRequest(r)
			if !limiter.Allow(ip) {
				w.Header().Set("Retry-After", "5")
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP extracts the real client IP from the request.
// Trusts X-Forwarded-For only from private/loopback addresses (reverse proxies).
// For audit logging use ClientIPHashed instead.
func ClientIP(r *http.Request) string {
	return clientIPFromRequest(r)
}
