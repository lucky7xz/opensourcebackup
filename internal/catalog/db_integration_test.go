//go:build integration

package catalog_test

import (
	"context"
	"os"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func TestOpen_ConnectsAndPings_WhenDatabaseIsReachable(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}

	ctx := context.Background()
	db, err := catalog.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if err := db.Ping(ctx); err != nil {
		t.Fatalf("Ping: %v", err)
	}
}

func TestOpen_ReturnsError_WhenDatabaseIsUnreachable(t *testing.T) {
	ctx := context.Background()
	_, err := catalog.Open(ctx, "postgres://invalid:invalid@localhost:19999/nonexistent?sslmode=disable&connect_timeout=1")
	if err == nil {
		t.Fatal("expected error for unreachable database, got nil")
	}
}
