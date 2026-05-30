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

type BackupRepository struct {
	ID                uuid.UUID
	Type              string
	Location          string
	EncryptionMode    *string
	ObjectLockEnabled bool
	RetentionPolicyID *uuid.UUID
	CreatedAt         time.Time
}

type BackupPolicy struct {
	ID        uuid.UUID
	Name      string
	Includes  []string
	Excludes  []string
	Schedule  *string
	Retention map[string]any
	Engine    string
	PreHooks  []string
	PostHooks []string
	CreatedAt time.Time
}

// BackupJob and Snapshot are models only — repositories added in B3.

type BackupJob struct {
	ID            uuid.UUID
	SystemID      uuid.UUID
	PolicyID      uuid.UUID
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
