package security

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

const (
	csrfCookieName = "osb_csrf"
	csrfHeaderName = "X-CSRF-Token"
	csrfTokenBytes = 16
)

// CSRFProtect returns middleware that enforces CSRF protection on state-mutating
// requests (POST, PUT, DELETE, PATCH).
//
// Strategy: Double-Submit Cookie pattern.
//   1. On GET requests a CSRF cookie is set (if not already present).
//   2. On mutating requests the X-CSRF-Token header must match the cookie value.
//
// This is sufficient for same-origin SPA calls. It prevents CSRF attacks because
// a malicious third-party site cannot read or set cookies on the dashboard origin.
//
// Agent API routes (/v1/agent/*) are exempt — they use Bearer token auth and
// are not called from a browser context.
func CSRFProtect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only enforce on mutating methods
		if isMutating(r.Method) && !isCSRFExempt(r.URL.Path) {
			cookie, err := r.Cookie(csrfCookieName)
			if err != nil || cookie.Value == "" {
				http.Error(w, `{"error":"missing CSRF cookie — reload the page"}`, http.StatusForbidden)
				return
			}
			header := r.Header.Get(csrfHeaderName)
			if !secureEqual(cookie.Value, header) {
				http.Error(w, `{"error":"CSRF token mismatch"}`, http.StatusForbidden)
				return
			}
		}

		// Ensure CSRF cookie exists on every response
		if _, err := r.Cookie(csrfCookieName); err != nil {
			token := newCSRFToken()
			http.SetCookie(w, &http.Cookie{
				Name:     csrfCookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: false, // Must be readable by JS to set the header
				SameSite: http.SameSiteStrictMode,
				// Secure is set by the outer HSTS/TLS layer
			})
		}

		next.ServeHTTP(w, r)
	})
}

func isMutating(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch:
		return true
	}
	return false
}

// isCSRFExempt lists paths that skip CSRF — agent API uses Bearer tokens,
// enrollment uses one-time tokens, login itself is bootstrapping auth.
func isCSRFExempt(path string) bool {
	exempt := []string{
		"/v1/agent/",    // Bearer token auth
		"/auth/login",   // bootstrapping — no session yet
		"/v1/agent/enroll",
	}
	for _, e := range exempt {
		if len(path) >= len(e) && path[:len(e)] == e {
			return true
		}
	}
	return false
}

func newCSRFToken() string {
	b := make([]byte, csrfTokenBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// secureEqual compares two strings in constant time to prevent timing attacks.
func secureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
