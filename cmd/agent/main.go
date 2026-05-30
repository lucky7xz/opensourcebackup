package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cerberus8484/opensourcebackup/internal/agent"
	agentclient "github.com/cerberus8484/opensourcebackup/internal/agent/client"
)

const (
	defaultPollInterval  = 30 * time.Second
	defaultTokenFilePath = "data/agent-token"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	controlPlaneURL := requireEnv(logger, "CONTROL_PLANE_URL")
	resticPassword := requireEnv(logger, "RESTIC_PASSWORD")
	resticRepo := requireEnv(logger, "RESTIC_REPO")

	token, err := resolveToken(logger, controlPlaneURL)
	if err != nil {
		logger.Error("failed to obtain agent token", "error", err)
		os.Exit(1)
	}

	poll := defaultPollInterval
	if v := os.Getenv("AGENT_POLL_INTERVAL"); v != "" {
		if d, parseErr := time.ParseDuration(v); parseErr == nil {
			poll = d
		}
	}

	cp := agentclient.New(controlPlaneURL, token)
	a := agent.New(agent.Config{
		PollInterval:   poll,
		ResticBin:      os.Getenv("RESTIC_BIN"),
		ResticPassword: resticPassword,
		ResticRepo:     resticRepo,
	}, cp, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		logger.Error("agent stopped with error", "error", err)
		os.Exit(1)
	}
}

// resolveToken returns an agent token using the following priority:
//  1. AGENT_TOKEN env var (direct)
//  2. Token file at AGENT_TOKEN_FILE (or default path)
//  3. Enroll using ENROLLMENT_TOKEN, save token to file
//  4. Abort if nothing works
func resolveToken(logger *slog.Logger, controlPlaneURL string) (string, error) {
	// 1. Direct env var (highest priority)
	if t := os.Getenv("AGENT_TOKEN"); t != "" {
		logger.Info("using agent token from AGENT_TOKEN env var")
		return t, nil
	}

	// 2. Token file
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

	// 3. Enroll with one-time enrollment token
	enrollmentToken := os.Getenv("ENROLLMENT_TOKEN")
	if enrollmentToken == "" {
		return "", fmt.Errorf(
			"no agent token available — set AGENT_TOKEN, create %s, or set ENROLLMENT_TOKEN",
			tokenFile,
		)
	}

	logger.Info("no agent token found — enrolling with enrollment token")
	// Use an unauthenticated client for enrollment (no bearer token yet)
	enrollClient := agentclient.New(controlPlaneURL, "")
	agentToken, err := enrollClient.Enroll(context.Background(), enrollmentToken)
	if err != nil {
		return "", fmt.Errorf("enrollment failed: %w", err)
	}

	if err := agent.SaveToken(tokenFile, agentToken); err != nil {
		// Non-fatal: token was obtained, just couldn't persist it
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
