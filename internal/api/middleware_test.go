package api_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/api"
)

var testLog = slog.New(slog.NewTextHandler(os.Stderr, nil))

func TestLogging_PassesRequestToNext(t *testing.T) {
	called := false
	handler := api.Logging(testLog)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/test", nil))
	if !called {
		t.Error("expected next handler to be called")
	}
}

func TestRecovery_Returns500_OnPanic(t *testing.T) {
	handler := api.Recovery(testLog)(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/panic", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", rec.Code)
	}
}

func TestTimeout_Returns503_WhenHandlerExceedsDeadline(t *testing.T) {
	handler := api.Timeout(10 * time.Millisecond)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(500 * time.Millisecond):
		case <-r.Context().Done():
		}
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/slow", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("want 503, got %d", rec.Code)
	}
}

func TestSecurityHeaders_SetsAllRequiredHeaders(t *testing.T) {
	handler := api.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))

	required := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-Xss-Protection":          "0", // modern browsers: CSP is the real defence
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Content-Security-Policy":   "default-src 'none'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}
	for header, want := range required {
		if got := rec.Header().Get(header); got != want {
			t.Errorf("header %s: want %q, got %q", header, want, got)
		}
	}
}

func TestSecurityHeadersCSP_IsHardened(t *testing.T) {
	handler := api.SecurityHeadersCSP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/ui/", nil))

	csp := rec.Header().Get("Content-Security-Policy")

	// script-src must be locked to 'self' — no inline scripts allowed (R-01).
	if !strings.Contains(csp, "script-src 'self';") {
		t.Errorf("script-src must be 'self', got CSP: %q", csp)
	}
	if strings.Contains(csp, "script-src 'self' 'unsafe-inline'") {
		t.Error("script-src must NOT allow 'unsafe-inline'")
	}
	// Dead font-CDN origins must be gone.
	for _, dead := range []string{"fonts.googleapis.com", "fonts.gstatic.com"} {
		if strings.Contains(csp, dead) {
			t.Errorf("CSP must not reference unused origin %q", dead)
		}
	}
	// style-src retains 'unsafe-inline' by design (React inline styles).
	if !strings.Contains(csp, "style-src 'self' 'unsafe-inline'") {
		t.Errorf("style-src must keep 'unsafe-inline' for inline styles, got: %q", csp)
	}
	// Framing still denied.
	if !strings.Contains(csp, "frame-ancestors 'none'") {
		t.Error("frame-ancestors 'none' must be present")
	}
}

func TestRequestBodyLimit_Returns413_WhenBodyExceedsLimit(t *testing.T) {
	handler := api.RequestBodyLimit(10)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 100)
		if _, err := r.Body.Read(buf); err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		}
	}))

	body := strings.NewReader("this body is definitely longer than ten bytes")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("POST", "/", body))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("want 413, got %d", rec.Code)
	}
}

func TestRequestBodyLimit_PassesThrough_WhenBodyWithinLimit(t *testing.T) {
	called := false
	handler := api.RequestBodyLimit(1024)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("POST", "/", strings.NewReader(`{"key":"val"}`)))
	if !called {
		t.Error("expected handler to be called for small body")
	}
}

func TestChain_AppliesMiddlewareInOrder(t *testing.T) {
	var order []string
	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1-before")
			next.ServeHTTP(w, r)
			order = append(order, "m1-after")
		})
	}
	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2-before")
			next.ServeHTTP(w, r)
			order = append(order, "m2-after")
		})
	}
	inner := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
	})

	api.Chain(inner, m1, m2).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	want := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("order[%d]: want %s, got %s", i, v, order[i])
		}
	}
}
