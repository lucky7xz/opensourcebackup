package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	"github.com/cerberus8484/opensourcebackup/internal/agent"
	agentclient "github.com/cerberus8484/opensourcebackup/internal/agent/client"
)

const defaultPollInterval = 30 * time.Second

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	cp := agentclient.New(cfg.controlPlaneURL, cfg.apiKey)
	a := agent.New(cfg.agentConfig, cp, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := a.Run(ctx); err != nil {
		logger.Error("agent error", "error", err)
		os.Exit(1)
	}
}

type config struct {
	controlPlaneURL string
	apiKey          string
	agentConfig     agent.Config
}

func loadConfig() (config, error) {
	controlPlaneURL := requireEnv("CONTROL_PLANE_URL")
	systemIDStr := requireEnv("SYSTEM_ID")
	resticPassword := requireEnv("RESTIC_PASSWORD")
	resticRepo := requireEnv("RESTIC_REPO")

	systemID, err := uuid.Parse(systemIDStr)
	if err != nil {
		return config{}, fmt.Errorf("SYSTEM_ID is not a valid UUID: %w", err)
	}

	poll := defaultPollInterval
	if v := os.Getenv("AGENT_POLL_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			poll = d
		}
	}

	return config{
		controlPlaneURL: controlPlaneURL,
		apiKey:          os.Getenv("AGENT_API_KEY"),
		agentConfig: agent.Config{
			SystemID:       systemID,
			PollInterval:   poll,
			ResticBin:      os.Getenv("RESTIC_BIN"),
			ResticPassword: resticPassword,
			ResticRepo:     resticRepo,
		},
	}, nil
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("required environment variable not set", "key", key)
		os.Exit(1)
	}
	return v
}
