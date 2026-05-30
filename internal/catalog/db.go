package catalog

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a PostgreSQL connection pool.
type DB struct {
	pool *pgxpool.Pool
}

// Open parses dsn, creates a connection pool, and verifies connectivity.
func Open(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("catalog: parse dsn: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("catalog: ping: %w", err)
	}
	return &DB{pool: pool}, nil
}

// Ping checks that the database is reachable.
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// Close releases all connections in the pool.
func (db *DB) Close() {
	db.pool.Close()
}

// Pool exposes the underlying pool for queries in other catalog sub-packages.
func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}
