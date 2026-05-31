package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "osb_session"
	sessionTTL        = 8 * time.Hour
	bcryptCost        = 12
	sessionTokenBytes = 32
)

// ErrInvalidCredentials is returned on bad username or password.
var ErrInvalidCredentials = errors.New("invalid credentials")

// session holds the server-side session state.
type session struct {
	createdAt time.Time
	ip        string // IP address that created the session
}

// WebAuthenticator guards the web dashboard with username+password (bcrypt).
// Sessions are stored in-memory with a fixed TTL.
//
// Production note: for multi-instance deployments replace the in-memory
// session store with a Redis-backed one. Single-node is fine for v1.
type WebAuthenticator struct {
	passwordHash []byte
	sessions     map[string]*session
	mu           sync.RWMutex
	gcOnce       sync.Once
	stop         chan struct{}
}

// NewWebAuthenticator creates an authenticator with a bcrypt-hashed password.
// Call HashPassword to produce the hash at startup from the plain-text env var.
func NewWebAuthenticator(bcryptHash []byte) *WebAuthenticator {
	wa := &WebAuthenticator{
		passwordHash: bcryptHash,
		sessions:     make(map[string]*session),
		stop:         make(chan struct{}),
	}
	go wa.gcLoop()
	return wa
}

// HashPassword hashes a plain-text password with bcrypt.
// Call once at startup — store the result in memory, never on disk.
func HashPassword(plain string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
}

// Login validates credentials and returns a session token on success.
// The caller must set the session cookie on the HTTP response.
func (wa *WebAuthenticator) Login(password, ip string) (token string, err error) {
	if err := bcrypt.CompareHashAndPassword(wa.passwordHash, []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	raw := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token = hex.EncodeToString(raw)

	wa.mu.Lock()
	wa.sessions[token] = &session{createdAt: time.Now(), ip: ip}
	wa.mu.Unlock()
	return token, nil
}

// Validate returns true if the session token is valid and not expired.
func (wa *WebAuthenticator) Validate(token string) bool {
	if token == "" {
		return false
	}
	wa.mu.RLock()
	s, ok := wa.sessions[token]
	wa.mu.RUnlock()
	return ok && time.Since(s.createdAt) < sessionTTL
}

// Logout invalidates a session token.
func (wa *WebAuthenticator) Logout(token string) {
	wa.mu.Lock()
	delete(wa.sessions, token)
	wa.mu.Unlock()
}

// SetCookie writes the session cookie to the response.
// Secure flag is set when TLS is detected (X-Forwarded-Proto or r.TLS).
func SetCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(sessionTTL.Seconds()),
		HttpOnly: true,              // not accessible to JS
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearCookie removes the session cookie.
func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// SessionFromRequest extracts the session token from the cookie or
// the Authorization: Bearer <token> header.
func SessionFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		return cookie.Value
	}
	const prefix = "Bearer "
	if h := r.Header.Get("Authorization"); len(h) > len(prefix) {
		if subtle.ConstantTimeCompare([]byte(h[:len(prefix)]), []byte(prefix)) == 1 {
			return h[len(prefix):]
		}
	}
	return ""
}

// gcLoop removes expired sessions every 10 minutes.
func (wa *WebAuthenticator) gcLoop() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			wa.mu.Lock()
			for tok, s := range wa.sessions {
				if time.Since(s.createdAt) >= sessionTTL {
					delete(wa.sessions, tok)
				}
			}
			wa.mu.Unlock()
		case <-wa.stop:
			return
		}
	}
}

// Stop releases background goroutine resources.
func (wa *WebAuthenticator) Stop() { close(wa.stop) }
