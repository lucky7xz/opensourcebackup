package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/api"
	"github.com/cerberus8484/opensourcebackup/internal/auth"
	"github.com/cerberus8484/opensourcebackup/internal/catalog"
	"github.com/cerberus8484/opensourcebackup/internal/scheduler"
)

const (
	serverReadTimeout       = 10 * time.Second
	serverReadHeaderTimeout = 5 * time.Second
	serverWriteTimeout      = 35 * time.Second
	serverIdleTimeout       = 60 * time.Second
	serverShutdownTimeout   = 10 * time.Second
	requestHandlerTimeout   = 30 * time.Second
	maxRequestBodyBytes     = 1 << 20 // 1 MB
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

	handler := api.New(
		catalog.NewSystemStore(db),
		catalog.NewRepositoryStore(db),
		catalog.NewPolicyStore(db),
		catalog.NewJobStore(db),
		catalog.NewSnapshotStore(db),
		catalog.NewRestoreTestStore(db),
		auth.NewEnrollmentTokenStore(db),
		auth.NewAgentTokenStore(db),
		logger,
	).WithPolicyNotifier(sched)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:5173"
	}

	httpHandler := api.Chain(mux,
		api.Recovery(logger),
		api.CORS(corsOrigin),
		api.SecurityHeaders,
		api.RequestBodyLimit(maxRequestBodyBytes),
		api.Logging(logger),
		api.Timeout(requestHandlerTimeout),
	)

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           httpHandler,
		ReadTimeout:       serverReadTimeout,
		ReadHeaderTimeout: serverReadHeaderTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
	}

	// TLS: wenn beide Dateien gesetzt sind → HTTPS, sonst HTTP (dev mode)
	tlsCert := os.Getenv("TLS_CERT_FILE")
	tlsKey := os.Getenv("TLS_KEY_FILE")
	tlsEnabled := tlsCert != "" && tlsKey != ""

	go func() {
		if tlsEnabled {
			logger.Info("control plane starting with HTTPS",
				"addr", addr,
				"cert", tlsCert,
			)
			if err := srv.ListenAndServeTLS(tlsCert, tlsKey); err != nil && err != http.ErrServerClosed {
				logger.Error("server error", "error", err)
			}
		} else {
			logger.Warn("control plane starting in HTTP dev mode — not for production",
				"addr", addr,
				"hint", "set TLS_CERT_FILE and TLS_KEY_FILE to enable HTTPS",
			)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("server error", "error", err)
			}
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("control plane stopped")
}
