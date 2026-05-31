package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	rbacSessionTTL     = 8 * time.Hour
	rbacSessionBytes   = 32
	rbacCookieName     = "osb_session" // same cookie name for seamless upgrade
)

// ErrSessionExpired is returned when a session token exists but has expired.
var ErrSessionExpired = errors.New("auth: session expired")

// Session holds server-side state for an authenticated user.
type Session struct {
	Token     string
	UserID    uuid.UUID
	UserEmail string
	Role      Role
	CreatedAt time.Time
	IP        string
}

// IsExpired reports whether the session TTL has elapsed.
func (s *Session) IsExpired() bool {
	return time.Since(s.CreatedAt) >= rbacSessionTTL
}

// RBACSessionManager manages multi-user sessions with role information.
// Sessions are in-memory for MVP — acceptable because:
//   - Sessions are short-lived (8h TTL)
//   - On restart users simply log in again
//   - DB-backed sessions can be added later without API changes
type RBACSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	stop     chan struct{}
	once     sync.Once
}

// NewRBACSessionManager creates a session manager and starts the GC goroutine.
func NewRBACSessionManager() *RBACSessionManager {
	m := &RBACSessionManager{
		sessions: make(map[string]*Session),
		stop:     make(chan struct{}),
	}
	go m.gc()
	return m
}

// Create creates a new session for the given user and returns the token.
func (m *RBACSessionManager) Create(userID uuid.UUID, email string, role Role, ip string) (string, error) {
	raw := make([]byte, rbacSessionBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	token := hex.EncodeToString(raw)
	m.mu.Lock()
	m.sessions[token] = &Session{
		Token:     token,
		UserID:    userID,
		UserEmail: email,
		Role:      role,
		CreatedAt: time.Now(),
		IP:        ip,
	}
	m.mu.Unlock()
	return token, nil
}

// Get returns the session for token, or an error if not found/expired.
func (m *RBACSessionManager) Get(token string) (*Session, error) {
	if token == "" {
		return nil, ErrInvalidCredentials
	}
	m.mu.RLock()
	s, ok := m.sessions[token]
	m.mu.RUnlock()
	if !ok {
		return nil, ErrInvalidCredentials
	}
	if s.IsExpired() {
		m.Revoke(token)
		return nil, ErrSessionExpired
	}
	return s, nil
}

// Revoke deletes a session.
func (m *RBACSessionManager) Revoke(token string) {
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}

// RevokeByUser removes all sessions for a user (e.g. on disable/delete).
func (m *RBACSessionManager) RevokeByUser(userID uuid.UUID) {
	m.mu.Lock()
	for tok, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, tok)
		}
	}
	m.mu.Unlock()
}

// SetCookie writes the session cookie.
func (m *RBACSessionManager) SetCookie(w http.ResponseWriter, r *http.Request, token string) {
	secure := r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https"
	http.SetCookie(w, &http.Cookie{
		Name:     rbacCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(rbacSessionTTL.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ClearCookie removes the session cookie.
func (m *RBACSessionManager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     rbacCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

// TokenFromRequest extracts the session token from cookie or Authorization header.
func TokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(rbacCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	const prefix = "Bearer "
	if h := r.Header.Get("Authorization"); len(h) > len(prefix) {
		return h[len(prefix):]
	}
	return ""
}

// gc periodically removes expired sessions.
func (m *RBACSessionManager) gc() {
	t := time.NewTicker(10 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			m.mu.Lock()
			for tok, s := range m.sessions {
				if s.IsExpired() {
					delete(m.sessions, tok)
				}
			}
			m.mu.Unlock()
		case <-m.stop:
			return
		}
	}
}

// Stop releases background goroutine resources.
func (m *RBACSessionManager) Stop() { m.once.Do(func() { close(m.stop) }) }

// ── Context key ───────────────────────────────────────────────────────────────

type contextKeySession struct{}

// WithSession stores a session in the request context.
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, contextKeySession{}, s)
}

// SessionFromContext retrieves the session from ctx.
func SessionFromContext(ctx context.Context) (*Session, bool) {
	s, ok := ctx.Value(contextKeySession{}).(*Session)
	return s, ok && s != nil
}

// RoleFromContext returns the user's role or RoleViewer if no session.
func RoleFromContext(ctx context.Context) Role {
	if s, ok := SessionFromContext(ctx); ok {
		return s.Role
	}
	return RoleViewer
}
