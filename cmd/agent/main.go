package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/kardianos/service"

	"github.com/cerberus8484/opensourcebackup/internal/agent"
	agentclient "github.com/cerberus8484/opensourcebackup/internal/agent/client"
)

const (
	defaultPollInterval  = 30 * time.Second
	defaultTokenFilePath = "data/agent-token"
	serviceName          = "OpenSourceBackupAgent"
	serviceDisplayName   = "OpenSourceBackup Agent"
	serviceDescription   = "Backup agent — polls the OpenSourceBackup Control Plane for jobs and executes Restic backups."
)

// program implements service.Interface so the agent can run as a
// Windows Service, systemd unit, or FreeBSD rc.d service.
type program struct {
	logger *slog.Logger
	cancel context.CancelFunc
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) run() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	p.logger = logger

	controlPlaneURL := requireEnv(logger, "CONTROL_PLANE_URL")
	resticPassword  := os.Getenv("RESTIC_PASSWORD")
	resticRepo      := os.Getenv("RESTIC_REPO")

	if resticPassword == "" || resticRepo == "" {
		logger.Warn("RESTIC_PASSWORD / RESTIC_REPO not set — backup/restore jobs will fail if triggered")
	}

	restoreRoot := os.Getenv("RESTORE_TEST_ROOT")
	if restoreRoot == "" {
		restoreRoot = "data/restore-tests"
	}

	poll := defaultPollInterval
	if v := os.Getenv("AGENT_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			poll = d
		}
	}

	token, err := resolveToken(logger, controlPlaneURL)
	if err != nil {
		logger.Error("failed to obtain agent token", "error", err)
		return
	}

	skipTLS := os.Getenv("AGENT_TLS_SKIP_VERIFY") == "true"
	if skipTLS {
		logger.Warn("TLS verification disabled — dev only")
	}

	cp := agentclient.New(controlPlaneURL, token, skipTLS)
	a := agent.New(agent.Config{
		PollInterval:    poll,
		ResticBin:       os.Getenv("RESTIC_BIN"),
		ResticPassword:  resticPassword,
		ResticRepo:      resticRepo,
		RestoreTestRoot: restoreRoot,
	}, cp, logger)

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	if err := a.Run(ctx); err != nil {
		logger.Error("agent stopped with error", "error", err)
	}
}

func (p *program) Stop(s service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	svcConfig := &service.Config{
		Name:        serviceName,
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		// Environment variables passed to the service
		EnvVars: buildEnvVars(),
	}

	prg := &program{}
	svc, err := service.New(prg, svcConfig)
	if err != nil {
		logger.Error("service setup failed", "error", err)
		os.Exit(1)
	}

	// Handle service control commands
	if len(os.Args) > 1 {
		cmd := strings.ToLower(os.Args[1])
		switch cmd {
		case "install":
			if err := svc.Install(); err != nil {
				fmt.Fprintf(os.Stderr, "Install failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Service installed successfully")
			fmt.Printf("  Start with: %s start\n", os.Args[0])
			return
		case "uninstall":
			if err := svc.Uninstall(); err != nil {
				fmt.Fprintf(os.Stderr, "Uninstall failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Service uninstalled")
			return
		case "start":
			if err := svc.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Start failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Service started")
			return
		case "stop":
			if err := svc.Stop(); err != nil {
				fmt.Fprintf(os.Stderr, "Stop failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Service stopped")
			return
		case "restart":
			if err := svc.Restart(); err != nil {
				fmt.Fprintf(os.Stderr, "Restart failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ Service restarted")
			return
		case "status":
			status, err := svc.Status()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Status error: %v\n", err)
				os.Exit(1)
			}
			switch status {
			case service.StatusRunning:
				fmt.Println("● Service is RUNNING")
			case service.StatusStopped:
				fmt.Println("○ Service is STOPPED")
			default:
				fmt.Println("? Service status unknown")
			}
			return
		case "run":
			// fall through to interactive run below
		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
			printUsage(os.Args[0])
			os.Exit(1)
		}
	}

	// Interactive mode (no args or "run") — run directly with signal handling
	if len(os.Args) <= 1 || strings.ToLower(os.Args[1]) == "run" {
		runInteractive(logger)
		return
	}

	// Run as service
	if err := svc.Run(); err != nil {
		logger.Error("service run error", "error", err)
		os.Exit(1)
	}
}

func runInteractive(logger *slog.Logger) {
	controlPlaneURL := requireEnv(logger, "CONTROL_PLANE_URL")
	resticPassword  := os.Getenv("RESTIC_PASSWORD")
	resticRepo      := os.Getenv("RESTIC_REPO")

	if resticPassword == "" || resticRepo == "" {
		logger.Warn("RESTIC_PASSWORD / RESTIC_REPO not set — backup/restore jobs will fail if triggered")
	}

	restoreRoot := os.Getenv("RESTORE_TEST_ROOT")
	if restoreRoot == "" {
		restoreRoot = "data/restore-tests"
	}

	poll := defaultPollInterval
	if v := os.Getenv("AGENT_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			poll = d
		}
	}

	token, err := resolveToken(logger, controlPlaneURL)
	if err != nil {
		logger.Error("failed to obtain agent token", "error", err)
		os.Exit(1)
	}

	skipTLS := os.Getenv("AGENT_TLS_SKIP_VERIFY") == "true"
	cp := agentclient.New(controlPlaneURL, token, skipTLS)
	a := agent.New(agent.Config{
		PollInterval:    poll,
		ResticBin:       os.Getenv("RESTIC_BIN"),
		ResticPassword:  resticPassword,
		ResticRepo:      resticRepo,
		RestoreTestRoot: restoreRoot,
	}, cp, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		logger.Error("agent stopped with error", "error", err)
		os.Exit(1)
	}
}

// buildEnvVars collects current env vars that should be passed to the service.
func buildEnvVars() map[string]string {
	vars := map[string]string{}
	keys := []string{
		"CONTROL_PLANE_URL", "RESTIC_PASSWORD", "RESTIC_REPO",
		"RESTIC_BIN", "AGENT_TOKEN_FILE", "AGENT_POLL_INTERVAL",
		"RESTORE_TEST_ROOT", "AGENT_TLS_SKIP_VERIFY",
	}
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			vars[k] = v
		}
	}
	return vars
}

func resolveToken(logger *slog.Logger, controlPlaneURL string) (string, error) {
	if t := os.Getenv("AGENT_TOKEN"); t != "" {
		logger.Info("using agent token from AGENT_TOKEN env var")
		return t, nil
	}
	tokenFile := os.Getenv("AGENT_TOKEN_FILE")
	if tokenFile == "" {
		tokenFile = defaultTokenFilePath
	}
	if t, err := agent.LoadToken(tokenFile); err != nil {
		return "", fmt.Errorf("load token file %s: %w", tokenFile, err)
	} else if t != "" {
		logger.Info("using agent token from file", "path", tokenFile)
		return t, nil
	}
	enrollmentToken := os.Getenv("ENROLLMENT_TOKEN")
	if enrollmentToken == "" {
		return "", fmt.Errorf(
			"no agent token available — set AGENT_TOKEN, create %s, or set ENROLLMENT_TOKEN",
			tokenFile,
		)
	}
	logger.Info("no agent token found — enrolling with enrollment token")
	enrollClient := agentclient.New(controlPlaneURL, "")
	agentToken, err := enrollClient.Enroll(context.Background(), enrollmentToken)
	if err != nil {
		return "", fmt.Errorf("enrollment failed: %w", err)
	}
	if err := agent.SaveToken(tokenFile, agentToken); err != nil {
		logger.Warn("could not save agent token to file", "path", tokenFile, "error", err)
	} else {
		logger.Info("agent token saved", "path", tokenFile)
	}
	return agentToken, nil
}

func requireEnv(logger *slog.Logger, key string) string {
	v := os.Getenv(key)
	if v == "" {
		logger.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}

func printUsage(bin string) {
	name := filepath.Base(bin)
	fmt.Printf(`
OpenSourceBackup Agent — Usage:

  %s                  Run interactively (foreground)
  %s run              Run interactively (foreground)
  %s install          Install as system service
  %s uninstall        Remove system service
  %s start            Start the service
  %s stop             Stop the service
  %s restart          Restart the service
  %s status           Show service status

Environment variables (required):
  CONTROL_PLANE_URL   URL of the Control Plane (e.g. http://192.168.1.100:8080)
  RESTIC_PASSWORD     Encryption password for backups
  RESTIC_REPO         Backup destination (e.g. /mnt/nas/backups or Z:\Backups)

Optional:
  ENROLLMENT_TOKEN    One-time token to enroll (first run only)
  AGENT_TOKEN_FILE    Path to saved token (default: data/agent-token)
  AGENT_POLL_INTERVAL Poll interval (default: 30s)
  RESTORE_TEST_ROOT   Sandbox directory for restore tests
  AGENT_TLS_SKIP_VERIFY=true  Skip TLS verification (dev only)
`, name, name, name, name, name, name, name, name)
}
