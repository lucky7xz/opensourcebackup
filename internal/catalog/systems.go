package catalog

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// SystemStore defines data access for the systems table.
type SystemStore interface {
	Create(ctx context.Context, s *System) error
	GetByID(ctx context.Context, id uuid.UUID) (*System, error)
	List(ctx context.Context) ([]System, error)
	Update(ctx context.Context, s *System) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type pgSystemStore struct {
	db *DB
}

// NewSystemStore returns a PostgreSQL-backed SystemStore.
func NewSystemStore(db *DB) SystemStore {
	return &pgSystemStore{db: db}
}

func (r *pgSystemStore) Create(ctx context.Context, s *System) error {
	tagsBytes, err := marshalJSONB(s.Tags)
	if err != nil {
		return err
	}
	row := r.db.pool.QueryRow(ctx, `
		INSERT INTO systems (hostname, os, agent_version, last_seen, tags, risk_class)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		s.Hostname, s.OS, s.AgentVersion, s.LastSeen, tagsBytes, s.RiskClass,
	)
	var rawID pgtype.UUID
	if err := row.Scan(&rawID, &s.CreatedAt); err != nil {
		return err
	}
	s.ID = uuid.UUID(rawID.Bytes)
	return nil
}

func (r *pgSystemStore) GetByID(ctx context.Context, id uuid.UUID) (*System, error) {
	row := r.db.pool.QueryRow(ctx, `
		SELECT id, hostname, os, agent_version, last_seen, tags, risk_class, created_at
		FROM systems WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	s, err := scanSystem(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return s, err
}

func (r *pgSystemStore) List(ctx context.Context) ([]System, error) {
	rows, err := r.db.pool.Query(ctx, `
		SELECT id, hostname, os, agent_version, last_seen, tags, risk_class, created_at
		FROM systems ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var systems []System
	for rows.Next() {
		s, err := scanSystem(rows)
		if err != nil {
			return nil, err
		}
		systems = append(systems, *s)
	}
	return systems, rows.Err()
}

func (r *pgSystemStore) Update(ctx context.Context, s *System) error {
	tagsBytes, err := marshalJSONB(s.Tags)
	if err != nil {
		return err
	}
	tag, err := r.db.pool.Exec(ctx, `
		UPDATE systems
		SET hostname=$1, os=$2, agent_version=$3, last_seen=$4, tags=$5, risk_class=$6
		WHERE id=$7`,
		s.Hostname, s.OS, s.AgentVersion, s.LastSeen, tagsBytes, s.RiskClass,
		pgtype.UUID{Bytes: s.ID, Valid: true},
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *pgSystemStore) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.pool.Exec(ctx, `DELETE FROM systems WHERE id = $1`,
		pgtype.UUID{Bytes: id, Valid: true},
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSystem(row rowScanner) (*System, error) {
	var (
		s       System
		rawID   pgtype.UUID
		tagsRaw []byte
	)
	if err := row.Scan(
		&rawID, &s.Hostname, &s.OS, &s.AgentVersion,
		&s.LastSeen, &tagsRaw, &s.RiskClass, &s.CreatedAt,
	); err != nil {
		return nil, err
	}
	s.ID = uuid.UUID(rawID.Bytes)
	if len(tagsRaw) > 0 {
		if err := json.Unmarshal(tagsRaw, &s.Tags); err != nil {
			return nil, err
		}
	}
	return &s, nil
}

func marshalJSONB(v any) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(v)
}
