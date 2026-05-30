package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// AgentToken is a long-lived bearer token bound to a specific system.
type AgentToken struct {
	ID         uuid.UUID
	SystemID   uuid.UUID
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// AgentTokenStore manages agent bearer tokens.
type AgentTokenStore interface {
	Create(ctx context.Context, systemID uuid.UUID, tokenHash string) (*AgentToken, error)
	ValidateAndTouch(ctx context.Context, tokenHash string) (uuid.UUID, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type pgAgentTokenStore struct {
	db *catalog.DB
}

// NewAgentTokenStore returns a PostgreSQL-backed AgentTokenStore.
func NewAgentTokenStore(db *catalog.DB) AgentTokenStore {
	return &pgAgentTokenStore{db: db}
}

func (s *pgAgentTokenStore) Create(ctx context.Context, systemID uuid.UUID, tokenHash string) (*AgentToken, error) {
	row := s.db.Pool().QueryRow(ctx, `
		INSERT INTO agent_tokens (system_id, token_hash)
		VALUES ($1, $2)
		RETURNING id, system_id, last_used_at, revoked_at, created_at`,
		pgtype.UUID{Bytes: systemID, Valid: true}, tokenHash,
	)
	return scanAgentToken(row)
}

// ValidateAndTouch looks up the token by hash, checks it is not revoked,
// updates last_used_at, and returns the bound system_id.
func (s *pgAgentTokenStore) ValidateAndTouch(ctx context.Context, tokenHash string) (uuid.UUID, error) {
	row := s.db.Pool().QueryRow(ctx, `
		UPDATE agent_tokens
		SET last_used_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
		RETURNING id, system_id, last_used_at, revoked_at, created_at`,
		tokenHash,
	)
	t, err := scanAgentToken(row)
	if err != nil {
		return uuid.Nil, ErrInvalidToken
	}
	return t.SystemID, nil
}

func (s *pgAgentTokenStore) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool().Exec(ctx,
		`UPDATE agent_tokens SET revoked_at = NOW() WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	return err
}

// RevokeBySystemID revokes all tokens for a system. Used when a system is deleted or compromised.
func RevokeBySystemID(ctx context.Context, db *catalog.DB, systemID uuid.UUID) error {
	_, err := db.Pool().Exec(ctx,
		`UPDATE agent_tokens SET revoked_at = NOW() WHERE system_id = $1 AND revoked_at IS NULL`,
		pgtype.UUID{Bytes: systemID, Valid: true},
	)
	return fmt.Errorf("revoke tokens for system %s: %w", systemID, err)
}

func scanAgentToken(row interface{ Scan(...any) error }) (*AgentToken, error) {
	var (
		t      AgentToken
		rawID  pgtype.UUID
		rawSys pgtype.UUID
	)
	if err := row.Scan(&rawID, &rawSys, &t.LastUsedAt, &t.RevokedAt, &t.CreatedAt); err != nil {
		return nil, err
	}
	t.ID = uuid.UUID(rawID.Bytes)
	t.SystemID = uuid.UUID(rawSys.Bytes)
	return &t, nil
}
