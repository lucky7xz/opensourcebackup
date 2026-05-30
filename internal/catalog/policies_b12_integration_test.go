//go:build integration

package catalog_test

import (
	"context"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func TestPolicyStore_Create_PersistsRepositoryID(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)

	repo := &catalog.BackupRepository{Type: "restic", Location: "s3:bucket"}
	catalog.NewRepositoryStore(db).Create(context.Background(), repo)

	store := catalog.NewPolicyStore(db)
	p := &catalog.BackupPolicy{
		Name:         "with-repo",
		Engine:       "restic",
		RepositoryID: &repo.ID,
	}
	if err := store.Create(context.Background(), p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RepositoryID == nil {
		t.Fatal("expected RepositoryID to be set")
	}
	if *got.RepositoryID != repo.ID {
		t.Errorf("RepositoryID: want %s, got %s", repo.ID, *got.RepositoryID)
	}
}

func TestPolicyStore_Create_AllowsNilRepositoryID(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)

	store := catalog.NewPolicyStore(db)
	p := &catalog.BackupPolicy{Name: "no-repo", Engine: "restic"}
	if err := store.Create(context.Background(), p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RepositoryID != nil {
		t.Error("expected RepositoryID to be nil")
	}
}

func TestPolicyStore_Update_SetsRepositoryID(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)

	repo := &catalog.BackupRepository{Type: "restic", Location: "s3:bucket"}
	catalog.NewRepositoryStore(db).Create(context.Background(), repo)

	store := catalog.NewPolicyStore(db)
	p := &catalog.BackupPolicy{Name: "update-test", Engine: "restic"}
	store.Create(context.Background(), p)

	p.RepositoryID = &repo.ID
	if err := store.Update(context.Background(), p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, _ := store.GetByID(context.Background(), p.ID)
	if got.RepositoryID == nil || *got.RepositoryID != repo.ID {
		t.Error("expected RepositoryID after update")
	}
}
