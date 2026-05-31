package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/api"
	"github.com/cerberus8484/opensourcebackup/internal/audit"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/cerberus8484/opensourcebackup/internal/metrics"
	"github.com/cerberus8484/opensourcebackup/internal/scheduler"
	"github.com/cerberus8484/opensourcebackup/internal/security"
)

const (
	// Minimum password length for bootstrap admin.
	minAdminPasswordLen = 10
)

const (
	serverReadTimeout       = 10 * time.Second
	serverReadHeaderTimeout = 5 * time.Second
	serverWriteTimeout      = 35 * time.Second
	serverIdleTimeout       = 60 * time.Second
	serverShutdownTimeout   = 10 * time.Second
	requestHandlerTimeout   = 30 * time.Second
	maxRequestBodyBytes     = 1 << 20 // 1 MB

	// Rate limiting: sustained requests per second (burst: 20 per IP)
	globalRatePerSec = 20.0
	globalBurst      = 20.0
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		logger.Error("DATABASE_URL not set")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := catalog.Open(ctx, dsn)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	// ── Audit store ──────────────────────────────────────────────────────────
	auditStore := audit.NewPostgresStore(db.Pool())

	// ── RBAC — multi-user authentication ────────────────────────────────────
	userStore := auth.NewUserStore(db.Pool())
	sessions  := auth.NewRBACSessionManager()

	// Bootstrap admin: if ADMIN_EMAIL + ADMIN_PASSWORD are set and no admin
	// user exists yet, create one automatically on first startup.
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPass  := os.Getenv("ADMIN_PASSWORD")
	if adminEmail != "" && adminPass != "" {
		if len(adminPass) < minAdminPasswordLen {
			logger.Error("ADMIN_PASSWORD too short — minimum 10 characters")
			os.Exit(1)
		}
		_, err := userStore.GetByEmail(ctx, adminEmail)
		if errors.Is(err, auth.ErrUserNotFound) {
			hash, err := auth.HashPassword(adminPass)
			if err != nil {
				logger.Error("failed to hash admin password", "error", err)
				os.Exit(1)
			}
			if _, err := userStore.Create(ctx, adminEmail, string(hash), auth.RoleAdmin, "Admin"); err != nil {
				logger.Error("failed to create bootstrap admin", "error", err)
				os.Exit(1)
			}
			logger.Info("bootstrap admin created", "email", adminEmail)
		} else if err == nil {
			logger.Info("admin user already exists — skipping bootstrap", "email", adminEmail)
		}
	} else if adminPass != "" {
		// Legacy single-password mode (no email set)
		logger.Warn("ADMIN_EMAIL not set — using legacy single-password mode",
			"hint", "set ADMIN_EMAIL to enable multi-user RBAC",
		)
	} else {
		logger.Warn("ADMIN_PASSWORD not set — dashboard accessible without login (dev only)")
	}

	// Legacy single-password fallback (only when ADMIN_EMAIL is not set)
	var webAuth *auth.WebAuthenticator
	if adminPass != "" && adminEmail == "" {
		hash, herr := auth.HashPassword(adminPass)
		if herr != nil {
			logger.Error("failed to hash admin password", "error", herr)
			os.Exit(1)
		}
		webAuth = auth.NewWebAuthenticator(hash)
		logger.Info("legacy single-password auth enabled (set ADMIN_EMAIL to upgrade to RBAC)")
	}

	// ── Scheduler ────────────────────────────────────────────────────────────
	sched := scheduler.New(
		catalog.NewPolicyStore(db),
		catalog.NewJobStore(db),
		logger,
	)
	go func() {
		if err := sched.Start(ctx); err != nil {
			logger.Error("scheduler error", "error", err)
		}
	}()

	// ── HTTP handler ─────────────────────────────────────────────────────────
	handler := api.New(
		catalog.NewSystemStore(db),
		catalog.NewRepositoryStore(db),
		catalog.NewPolicyStore(db),
		catalog.NewJobStore(db),
		catalog.NewSnapshotStore(db),
		catalog.NewRestoreTestStore(db),
		auth.NewEnrollmentTokenStore(db),
		auth.NewAgentTokenStore(db),
		auditStore,
		logger,
	).WithPolicyNotifier(sched).WithWebAuth(webAuth).WithRBAC(sessions, userStore)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// ── Prometheus /metrics ───────────────────────────────────────────────────
	// Served on the same port as the API — no separate metrics port.
	// The endpoint is unauthenticated by design: Prometheus scrapers typically
	// run inside the same network. Restrict access at the network/firewall level
	// if metrics should not be publicly accessible.
	// Same stores as the API — no shadow data, no duplication.
	metricsHandler := metrics.NewHandler(metrics.Stores{
		Systems:      catalog.NewSystemStore(db),
		Jobs:         catalog.NewJobStore(db),
		Snapshots:    catalog.NewSnapshotStore(db),
		RestoreTests: catalog.NewRestoreTestStore(db),
		Repositories: catalog.NewRepositoryStore(db),
		Policies:     catalog.NewPolicyStore(db),
	}, logger)
	mux.Handle("/metrics", metricsHandler)

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:5173"
	}

	// Global rate limiter — protects all endpoints
	globalLimiter := security.NewIPRateLimiter(globalRatePerSec, globalBurst)

	// Middleware chain (outer → inner):
	//   Timeout → Logging → RateLimit → WebAuth → CSRF → SecurityHeaders → CORS → BodyLimit → Recovery
	httpHandler := api.Chain(mux,
		api.Recovery(logger),
		api.RequestBodyLimit(maxRequestBodyBytes),
		api.CORS(corsOrigin),
		api.SecurityHeadersCSP,
		security.CSRFProtect,
		api.RBACMiddleware(sessions, auditStore), // replaces old WebAuth
		security.RateLimit(globalLimiter),
		api.Logging(logger),
		api.Timeout(requestHandlerTimeout),
	)

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	tlsCert     := os.Getenv("TLS_CERT_FILE")
	tlsKey      := os.Getenv("TLS_KEY_FILE")
	tlsEnabled  := tlsCert != "" && tlsKey != ""
	tlsRequired := os.Getenv("TLS_REQUIRED") == "true"

	// When TLS is required but not configured, refuse to start.
	// This prevents accidental plaintext deployments in production.
	if tlsRequired && !tlsEnabled {
		logger.Error("TLS_REQUIRED=true but TLS_CERT_FILE/TLS_KEY_FILE not set — refusing to start in HTTP mode")
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpHandler,
		ReadTimeout:       serverReadTimeout,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	// Optional HTTP→HTTPS redirect server.
	// When TLS is enabled and HTTP_REDIRECT_ADDR is set, a minimal redirect
	// server listens on that address and redirects all traffic to HTTPS.
	// Example: LISTEN_ADDR=:8443 HTTP_REDIRECT_ADDR=:8080
	var redirectSrv *http.Server
	if tlsEnabled {
		if redirectAddr := os.Getenv("HTTP_REDIRECT_ADDR"); redirectAddr != "" {
			redirectSrv = &http.Server{
				Addr:              redirectAddr,
				ReadTimeout:       serverReadTimeout,
				ReadHeaderTimeout: serverReadHeaderTimeout,
				WriteTimeout:      serverWriteTimeout,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					target := "https://" + r.Host + r.RequestURI
					http.Redirect(w, r, target, http.StatusMovedPermanently)
				}),
			}
			go func() {
				logger.Info("HTTP redirect server started", "addr", redirectAddr, "redirects_to", "https")
				if err := redirectSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("redirect server error", "error", err)
				}
			}()
		}
	}

	go func() {
		if tlsEnabled {
			logger.Info("control plane starting with HTTPS",
				"addr", addr,
				"cert", tlsCert,
				"tls_required", tlsRequired,
			)
			if err := srv.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
				logger.Error("server error", "error", err)
			}
		} else {
			logger.Warn("HTTP mode — tokens and data are transmitted unencrypted",
				"addr", addr,
				"hint", "set TLS_CERT_FILE + TLS_KEY_FILE for production use",
			)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("server error", "error", err)
			}
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()
	if redirectSrv != nil {
		if err := redirectSrv.Shutdown(shutdownCtx); err != nil {
			logger.Error("redirect server shutdown error", "error", err)
		}
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	globalLimiter.Stop()
	sessions.Stop()
	if webAuth != nil {
		webAuth.Stop()
	}
	logger.Info("control plane stopped")
}
