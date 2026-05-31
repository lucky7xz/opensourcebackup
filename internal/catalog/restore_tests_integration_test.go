//go:build integration

package catalog_test

import (
	"context"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func createFixtureForRestoreTest(t *testing.T, db *catalog.DB) (catalog.Snapshot, catalog.BackupJob) {
	t.Helper()
	ctx := context.Background()

	sys := &catalog.System{Hostname: "rt-test-host", RiskClass: "standard"}
	catalog.NewSystemStore(db).Create(ctx, sys)

	repo := &catalog.BackupRepository{Type: "restic", Location: "s3:rt-bucket"}
	catalog.NewRepositoryStore(db).Create(ctx, repo)

	pol := &catalog.BackupPolicy{Name: "rt-policy", Engine: "restic", RepositoryID: &repo.ID}
	catalog.NewPolicyStore(db).Create(ctx, pol)

	job := &catalog.BackupJob{SystemID: sys.ID, PolicyID: pol.ID, Status: "success"}
	catalog.NewJobStore(db).Create(ctx, job)

	snap := &catalog.Snapshot{
		JobID: job.ID, RepositoryID: repo.ID,
		EngineSnapshotID: "snap-rt-001", ChecksumStatus: "verified",
	}
	catalog.NewSnapshotStore(db).Create(ctx, snap)

	return *snap, *job
}

func TestRestoreTestStore_Create_AssignsID(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	snap, job := createFixtureForRestoreTest(t, db)
	store := catalog.NewRestoreTestStore(db)

	rt := &catalog.RestoreTest{
		SnapshotID:   snap.ID,
		SystemID:     job.SystemID,
		RepositoryID: snap.RepositoryID,
		Status:       "pending",
	}
	if err := store.Create(context.Background(), rt); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rt.ID.String() == "" {
		t.Fatal("expected ID after Create")
	}
}

func TestRestoreTestStore_Create_DerivesFromSnapshot(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	snap, job := createFixtureForRestoreTest(t, db)
	store := catalog.NewRestoreTestStore(db)

	rt := &catalog.RestoreTest{
		SnapshotID:   snap.ID,
		SystemID:     job.SystemID,
		RepositoryID: snap.RepositoryID,
		Status:       "pending",
	}
	store.Create(context.Background(), rt)

	got, err := store.GetByID(context.Background(), rt.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.SnapshotID != snap.ID {
		t.Error("SnapshotID mismatch")
	}
	if got.SystemID != job.SystemID {
		t.Error("SystemID mismatch")
	}
	if got.RepositoryID != snap.RepositoryID {
		t.Error("RepositoryID mismatch")
	}
}

func TestRestoreTestStore_Update_PersistsSuccess(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	snap, job := createFixtureForRestoreTest(t, db)
	store := catalog.NewRestoreTestStore(db)

	rt := &catalog.RestoreTest{
		SnapshotID: snap.ID, SystemID: job.SystemID,
		RepositoryID: snap.RepositoryID, Status: "pending",
	}
	store.Create(context.Background(), rt)

	files := 42
	bytes := int64(1234567)
	rt.Status = "success"
	rt.VerifiedFiles = &files
	rt.VerifiedBytes = &bytes
	if err := store.Update(context.Background(), rt); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := store.GetByID(context.Background(), rt.ID)
	if got.Status != "success" {
		t.Errorf("Status: want success, got %s", got.Status)
	}
	if got.VerifiedFiles == nil || *got.VerifiedFiles != 42 {
		t.Error("VerifiedFiles mismatch")
	}
}

func TestRestoreTestStore_HasSuccessfulTest(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	snap, job := createFixtureForRestoreTest(t, db)
	store := catalog.NewRestoreTestStore(db)

	// No test yet
	has, _ := store.HasSuccessfulTest(context.Background(), snap.ID)
	if has {
		t.Error("expected no successful test initially")
	}

	// Create successful test
	rt := &catalog.RestoreTest{
		SnapshotID: snap.ID, SystemID: job.SystemID,
		RepositoryID: snap.RepositoryID, Status: "success",
	}
	store.Create(context.Background(), rt)

	has, _ = store.HasSuccessfulTest(context.Background(), snap.ID)
	if !has {
		t.Error("expected successful test after creation")
	}
}

func TestRestoreTestStore_GetByID_ReturnsErrNotFound(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewRestoreTestStore(db)

	_, err := store.GetByID(context.Background(), [16]byte{0xFF})
	if err != catalog.ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestRestoreTestStore_Delete_RemovesTest(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	snap, job := createFixtureForRestoreTest(t, db)
	store := catalog.NewRestoreTestStore(db)

	rt := &catalog.RestoreTest{
		SnapshotID: snap.ID, SystemID: job.SystemID,
		RepositoryID: snap.RepositoryID, Status: "pending",
	}
	store.Create(context.Background(), rt)
	store.Delete(context.Background(), rt.ID)

	_, err := store.GetByID(context.Background(), rt.ID)
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound after Delete, got %v", err)
	}
}
