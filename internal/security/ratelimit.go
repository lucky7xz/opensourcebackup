// Package security provides security middleware for the HTTP API:
// rate limiting, brute-force protection, and IP-based throttling.
package security

import (
	"net"
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
			ip := clientIP(r)
			if !limiter.Allow(ip) {
				w.Header().Set("Retry-After", "5")
				http.Error(w, `{"error":"too many requests"}`, http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClientIP is the exported version for use outside this package.
func ClientIP(r *http.Request) string { return clientIP(r) }

// clientIP extracts the real client IP, honouring X-Forwarded-For
// only when the direct connection is from a private/loopback address
// (i.e. a trusted reverse proxy).
func clientIP(r *http.Request) string {
	remoteIP, _, _ := net.SplitHostPort(r.RemoteAddr)

	if isPrivate(remoteIP) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Take the first (leftmost) IP — that is the original client.
			if idx := len(xff); idx > 0 {
				for i := 0; i < len(xff); i++ {
					if xff[i] == ',' {
						return xff[:i]
					}
				}
				return xff
			}
		}
		if xri := r.Header.Get("X-Real-IP"); xri != "" {
			return xri
		}
	}
	return remoteIP
}

// isPrivate reports whether ip is RFC-1918 / loopback.
func isPrivate(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	private := []string{
		"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8", "::1/128",
	}
	for _, cidr := range private {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil && network.Contains(parsed) {
			return true
		}
	}
	return false
}
