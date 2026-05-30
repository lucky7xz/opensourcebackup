//go:build integration

package catalog_test

import (
	"context"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func createFixtureJobAndRepo(t *testing.T, db *catalog.DB) (*catalog.BackupJob, *catalog.BackupRepository) {
	t.Helper()
	ctx := context.Background()

	sys, pol := createFixtureSystemAndPolicy(t, db)

	job := &catalog.BackupJob{SystemID: sys.ID, PolicyID: pol.ID, Status: "success"}
	if err := catalog.NewJobStore(db).Create(ctx, job); err != nil {
		t.Fatalf("fixture job: %v", err)
	}

	repo := &catalog.BackupRepository{Type: "restic", Location: "s3:fixture-bucket"}
	if err := catalog.NewRepositoryStore(db).Create(ctx, repo); err != nil {
		t.Fatalf("fixture repo: %v", err)
	}
	return job, repo
}

func TestSnapshotStore_Create_AssignsIDAndCreatedAt(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	job, repo := createFixtureJobAndRepo(t, db)
	store := catalog.NewSnapshotStore(db)

	snap := &catalog.Snapshot{
		JobID:            job.ID,
		EngineSnapshotID: "abc123",
		RepositoryID:     repo.ID,
		Paths:            []string{"/home", "/etc"},
		ChecksumStatus:   "unverified",
	}
	if err := store.Create(context.Background(), snap); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if snap.ID.String() == "" {
		t.Fatal("expected ID after Create")
	}
	if snap.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt after Create")
	}
}

func TestSnapshotStore_GetByID_ReturnsSnapshot_WhenExists(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	job, repo := createFixtureJobAndRepo(t, db)
	store := catalog.NewSnapshotStore(db)

	created := &catalog.Snapshot{
		JobID: job.ID, RepositoryID: repo.ID,
		EngineSnapshotID: "snap-xyz",
		Paths:            []string{"/data"},
		ChecksumStatus:   "verified",
	}
	store.Create(context.Background(), created)

	got, err := store.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.EngineSnapshotID != "snap-xyz" {
		t.Errorf("EngineSnapshotID: want snap-xyz, got %s", got.EngineSnapshotID)
	}
	if got.ChecksumStatus != "verified" {
		t.Errorf("ChecksumStatus: want verified, got %s", got.ChecksumStatus)
	}
	if len(got.Paths) != 1 || got.Paths[0] != "/data" {
		t.Errorf("Paths: want [/data], got %v", got.Paths)
	}
}

func TestSnapshotStore_GetByID_ReturnsErrNotFound_WhenMissing(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSnapshotStore(db)

	_, err := store.GetByID(context.Background(), [16]byte{0xFF})
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSnapshotStore_List_ReturnsAllSnapshots(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	job, repo := createFixtureJobAndRepo(t, db)
	store := catalog.NewSnapshotStore(db)

	for _, id := range []string{"snap-a", "snap-b"} {
		store.Create(context.Background(), &catalog.Snapshot{
			JobID: job.ID, RepositoryID: repo.ID,
			EngineSnapshotID: id, ChecksumStatus: "unverified",
		})
	}

	snaps, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(snaps) != 2 {
		t.Errorf("want 2 snapshots, got %d", len(snaps))
	}
}

func TestSnapshotStore_ListByJobID_ReturnsOnlyMatchingSnapshots(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	job, repo := createFixtureJobAndRepo(t, db)

	sys2 := &catalog.System{Hostname: "other", RiskClass: "standard"}
	catalog.NewSystemStore(db).Create(context.Background(), sys2)
	pol := &catalog.BackupPolicy{Name: "p2", Engine: "restic"}
	catalog.NewPolicyStore(db).Create(context.Background(), pol)
	job2 := &catalog.BackupJob{SystemID: sys2.ID, PolicyID: pol.ID, Status: "success"}
	catalog.NewJobStore(db).Create(context.Background(), job2)

	store := catalog.NewSnapshotStore(db)
	store.Create(context.Background(), &catalog.Snapshot{JobID: job.ID, RepositoryID: repo.ID, EngineSnapshotID: "s1", ChecksumStatus: "unverified"})
	store.Create(context.Background(), &catalog.Snapshot{JobID: job.ID, RepositoryID: repo.ID, EngineSnapshotID: "s2", ChecksumStatus: "unverified"})
	store.Create(context.Background(), &catalog.Snapshot{JobID: job2.ID, RepositoryID: repo.ID, EngineSnapshotID: "s3", ChecksumStatus: "unverified"})

	snaps, err := store.ListByJobID(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("ListByJobID: %v", err)
	}
	if len(snaps) != 2 {
		t.Errorf("want 2 snapshots for job, got %d", len(snaps))
	}
}

func TestSnapshotStore_Delete_RemovesSnapshot(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	job, repo := createFixtureJobAndRepo(t, db)
	store := catalog.NewSnapshotStore(db)

	snap := &catalog.Snapshot{JobID: job.ID, RepositoryID: repo.ID, EngineSnapshotID: "del", ChecksumStatus: "unverified"}
	store.Create(context.Background(), snap)
	store.Delete(context.Background(), snap.ID)

	_, err := store.GetByID(context.Background(), snap.ID)
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound after Delete, got %v", err)
	}
}
