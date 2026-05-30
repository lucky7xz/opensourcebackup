//go:build integration

package catalog_test

import (
	"context"
	"os"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func newTestDB(t *testing.T) *catalog.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL not set — skipping integration test")
	}
	db, err := catalog.Open(context.Background(), dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(db.Close)
	return db
}

func truncateTables(t *testing.T, db *catalog.DB) {
	t.Helper()
	_, err := db.Pool().Exec(context.Background(),
		"TRUNCATE snapshots, backup_jobs, backup_policies, repositories, systems RESTART IDENTITY CASCADE",
	)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}
}
