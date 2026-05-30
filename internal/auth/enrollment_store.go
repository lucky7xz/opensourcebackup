package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

// EnrollmentToken is a one-time token used by an agent to enroll with the control plane.
type EnrollmentToken struct {
	ID        uuid.UUID
	SystemID  uuid.UUID
	ExpiresAt time.Time
	UsedAt    *time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// EnrollmentTokenStore manages enrollment tokens.
type EnrollmentTokenStore interface {
	Create(ctx context.Context, systemID uuid.UUID, tokenHash string, expiresAt time.Time) (*EnrollmentToken, error)
	Consume(ctx context.Context, tokenHash string) (*EnrollmentToken, error)
	Revoke(ctx context.Context, id uuid.UUID) error
}

type pgEnrollmentTokenStore struct {
	db *catalog.DB
}

// NewEnrollmentTokenStore returns a PostgreSQL-backed EnrollmentTokenStore.
func NewEnrollmentTokenStore(db *catalog.DB) EnrollmentTokenStore {
	return &pgEnrollmentTokenStore{db: db}
}

func (s *pgEnrollmentTokenStore) Create(ctx context.Context, systemID uuid.UUID, tokenHash string, expiresAt time.Time) (*EnrollmentToken, error) {
	row := s.db.Pool().QueryRow(ctx, `
		INSERT INTO agent_enrollment_tokens (system_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
		RETURNING id, system_id, expires_at, used_at, revoked_at, created_at`,
		pgtype.UUID{Bytes: systemID, Valid: true}, tokenHash, expiresAt,
	)
	return scanEnrollmentToken(row)
}

// Consume validates the token hash, checks it is unused and unexpired, then marks it used.
// Returns ErrInvalidToken if not found/expired, ErrTokenAlreadyUsed if already consumed,
// ErrTokenRevoked if revoked.
func (s *pgEnrollmentTokenStore) Consume(ctx context.Context, tokenHash string) (*EnrollmentToken, error) {
	row := s.db.Pool().QueryRow(ctx, `
		SELECT id, system_id, expires_at, used_at, revoked_at, created_at
		FROM agent_enrollment_tokens
		WHERE token_hash = $1`,
		tokenHash,
	)
	t, err := scanEnrollmentToken(row)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if t.RevokedAt != nil {
		return nil, ErrTokenRevoked
	}
	if t.UsedAt != nil {
		return nil, ErrTokenAlreadyUsed
	}
	if time.Now().After(t.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	now := time.Now()
	if _, err := s.db.Pool().Exec(ctx,
		`UPDATE agent_enrollment_tokens SET used_at = $1 WHERE id = $2`,
		now, pgtype.UUID{Bytes: t.ID, Valid: true},
	); err != nil {
		return nil, fmt.Errorf("consume enrollment token: %w", err)
	}
	t.UsedAt = &now
	return t, nil
}

func (s *pgEnrollmentTokenStore) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool().Exec(ctx,
		`UPDATE agent_enrollment_tokens SET revoked_at = NOW() WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	return err
}

func scanEnrollmentToken(row interface{ Scan(...any) error }) (*EnrollmentToken, error) {
	var (
		t      EnrollmentToken
		rawID  pgtype.UUID
		rawSys pgtype.UUID
	)
	if err := row.Scan(&rawID, &rawSys, &t.ExpiresAt, &t.UsedAt, &t.RevokedAt, &t.CreatedAt); err != nil {
		return nil, err
	}
	t.ID = uuid.UUID(rawID.Bytes)
	t.SystemID = uuid.UUID(rawSys.Bytes)
	return &t, nil
}
