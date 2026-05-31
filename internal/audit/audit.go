// Package audit records security-relevant events for GDPR compliance,
// incident investigation, and operational visibility.
//
// Every mutating API call (create/update/delete) and all authentication
// events are appended to the audit_log table. The table is append-only
// by convention — rows are never updated or deleted except by a privileged
// GDPR purge operation (which itself writes a purge record first).
package audit

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Action classifies the type of audit event.
type Action string

const (
	// Data-management actions
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionPurge  Action = "gdpr_purge"   // hard-delete by GDPR request
	ActionExport Action = "gdpr_export"  // data export by GDPR request

	// Authentication actions
	ActionLogin        Action = "auth_login"
	ActionLoginFail    Action = "auth_login_fail"
	ActionLogout       Action = "auth_logout"
	ActionTokenRevoke  Action = "token_revoke"

	// Backup/restore actions
	ActionBackupStart    Action = "backup_start"
	ActionBackupComplete Action = "backup_complete"
	ActionBackupFail     Action = "backup_fail"
	ActionRestoreStart   Action = "restore_start"
	ActionRestoreComplete Action = "restore_complete"
	ActionRestoreFail    Action = "restore_fail"
)

// ResourceType identifies the kind of object an event relates to.
type ResourceType string

const (
	ResourceSystem      ResourceType = "system"
	ResourceRepository  ResourceType = "repository"
	ResourcePolicy      ResourceType = "policy"
	ResourceJob         ResourceType = "job"
	ResourceSnapshot    ResourceType = "snapshot"
	ResourceRestoreTest ResourceType = "restore_test"
	ResourceAuth        ResourceType = "auth"
	ResourceAgent       ResourceType = "agent"
)

// Entry is a single immutable audit record.
type Entry struct {
	ID           int64
	Timestamp    time.Time
	Action       Action
	ResourceType ResourceType
	ResourceID   string // UUID or empty for auth events
	Actor        string // "admin", "agent:<system_id>", "system"
	IP           string // client IP address
	UserAgent    string
	Details      string // optional free-text JSON or description
	Success      bool
}

// Store persists audit entries.
type Store interface {
	// Append writes a new audit entry. It never fails silently —
	// if the DB write fails, the error is returned and the caller
	// should log it (but not block the main operation).
	Append(ctx context.Context, e Entry) error

	// List returns entries filtered by resource type and/or resource ID.
	// Pass empty strings to list all. Results are ordered newest-first.
	List(ctx context.Context, resourceType ResourceType, resourceID string, limit int) ([]Entry, error)
}

// PostgresStore implements Store against a PostgreSQL connection pool (pgxpool).
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a Store backed by a pgxpool connection pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Append inserts an audit entry. Intentionally fire-and-forget friendly —
// the caller should not block on an audit write failure.
func (s *PostgresStore) Append(ctx context.Context, e Entry) error {
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	const q = `
		INSERT INTO audit_log
			(timestamp, action, resource_type, resource_id, actor, ip, user_agent, details, success)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err := s.pool.Exec(ctx, q,
		e.Timestamp, string(e.Action), string(e.ResourceType),
		e.ResourceID, e.Actor, e.IP, e.UserAgent, e.Details, e.Success,
	)
	return err
}

// List returns audit entries, newest first, with an optional filter.
func (s *PostgresStore) List(ctx context.Context, rt ResourceType, rid string, limit int) ([]Entry, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	args := []any{limit}
	where := ""
	if rt != "" && rid != "" {
		where = "WHERE resource_type = $2 AND resource_id = $3"
		args = append(args, string(rt), rid)
	} else if rt != "" {
		where = "WHERE resource_type = $2"
		args = append(args, string(rt))
	}

	q := `SELECT id, timestamp, action, resource_type, resource_id,
	             actor, ip, user_agent, details, success
	      FROM audit_log ` + where + `
	      ORDER BY timestamp DESC LIMIT $1`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var resType, action string
		if err := rows.Scan(
			&e.ID, &e.Timestamp, &action, &resType,
			&e.ResourceID, &e.Actor, &e.IP, &e.UserAgent, &e.Details, &e.Success,
		); err != nil {
			return nil, err
		}
		e.Action = Action(action)
		e.ResourceType = ResourceType(resType)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// NoopStore discards all audit events. Used in tests and when audit is disabled.
type NoopStore struct{}

func (NoopStore) Append(_ context.Context, _ Entry) error                              { return nil }
func (NoopStore) List(_ context.Context, _ ResourceType, _ string, _ int) ([]Entry, error) {
	return nil, nil
}
