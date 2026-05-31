package catalog

import (
	"time"

	"github.com/google/uuid"
)

type System struct {
	ID           uuid.UUID
	Hostname     string
	OS           *string
	AgentVersion *string
	LastSeen     *time.Time
	Tags         map[string]any
	RiskClass    string
	CreatedAt    time.Time
}

// ImmutableMode documents the write-protection mechanism of a repository.
// This is a declared property — not automatically verified against the storage backend.
type ImmutableMode string

const (
	ImmutableNone       ImmutableMode = "none"        // no write protection
	ImmutableObjectLock ImmutableMode = "object_lock" // S3/MinIO Object Lock
	ImmutableWORM       ImmutableMode = "worm"        // hardware/NAS WORM
	ImmutableAppendOnly ImmutableMode = "append_only" // restic --append-only or similar
	ImmutableUnknown    ImmutableMode = "unknown"      // not verified
)

// IsProtected reports whether this mode provides any form of write protection.
func (m ImmutableMode) IsProtected() bool {
	return m != ImmutableNone && m != ImmutableUnknown && m != ""
}

type BackupRepository struct {
	ID                uuid.UUID
	Type              string
	Location          string
	EncryptionMode    *string
	ObjectLockEnabled bool
	ImmutableMode     ImmutableMode // preferred over ObjectLockEnabled for new code
	RetentionPolicyID *uuid.UUID
	CreatedAt         time.Time
}

// RepositoryHealth holds derived health indicators for a repository.
// Computed on demand — never stored. Honest: no fake "healthy" without evidence.
type RepositoryHealth struct {
	RepositoryID       uuid.UUID
	EncryptionEnabled  bool
	ImmutableMode      ImmutableMode
	SnapshotCount      int
	VerifiedCount      int     // snapshots with successful restore test
	LastBackupAt       *time.Time
	LastRestoreTestAt  *time.Time
	LastRetentionAt    *time.Time
}

// RetentionPlan holds the restic-compatible keep rules for a policy.
// All zero values mean "no limit" for that dimension.
// The safety rule (never delete the last restore-tested snapshot) is enforced
// by the control plane during retention job validation — independent of these values.
type RetentionPlan struct {
	KeepLast    int `json:"keep_last,omitempty"`
	KeepDaily   int `json:"keep_daily,omitempty"`
	KeepWeekly  int `json:"keep_weekly,omitempty"`
	KeepMonthly int `json:"keep_monthly,omitempty"`
	KeepYearly  int `json:"keep_yearly,omitempty"`
}

// HasRules reports whether any keep rule is configured.
func (r RetentionPlan) HasRules() bool {
	return r.KeepLast > 0 || r.KeepDaily > 0 || r.KeepWeekly > 0 ||
		r.KeepMonthly > 0 || r.KeepYearly > 0
}

type BackupPolicy struct {
	ID           uuid.UUID
	Name         string
	Includes     []string
	Excludes     []string
	Schedule     *string
	Retention    map[string]any
	RetentionPlan RetentionPlan  // typed keep rules — sourced from dedicated columns
	Engine       string
	PreHooks     []string
	PostHooks    []string
	RepositoryID *uuid.UUID // which repository this policy backs up to
	CreatedAt    time.Time
}

const (
	JobTypeBackup    = "backup"
	JobTypeRetention = "retention"
)

// RestoreTest records the result of verifying a snapshot can be restored.
type RestoreTest struct {
	ID            uuid.UUID
	SnapshotID    uuid.UUID
	SystemID      uuid.UUID
	RepositoryID  uuid.UUID
	Status        string
	TargetPath    *string
	StartedAt     *time.Time
	FinishedAt    *time.Time
	VerifiedFiles *int
	VerifiedBytes *int64
	ErrorSummary  *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// BackupJob and Snapshot are models only — repositories added in B3.

type BackupJob struct {
	ID            uuid.UUID
	SystemID      uuid.UUID
	PolicyID      uuid.UUID
	Type          string     // "backup" | "retention" — see JobTypeBackup / JobTypeRetention
	StartedAt     *time.Time
	FinishedAt    *time.Time
	Status        string
	BytesScanned  *int64
	BytesUploaded *int64
	ErrorSummary  *string
	RawOutput     map[string]any
	CreatedAt     time.Time
}

type Snapshot struct {
	ID               uuid.UUID
	JobID            uuid.UUID
	EngineSnapshotID string
	RepositoryID     uuid.UUID
	CreatedAt        time.Time
	Hostname         *string
	Paths            []string
	ChecksumStatus   string
}
