//go:build integration

package catalog_test

import (
	"context"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func TestPolicyStore_Create_AssignsIDAndCreatedAt(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewPolicyStore(db)

	schedule := "0 2 * * *"
	p := &catalog.BackupPolicy{
		Name:      "nightly-full",
		Engine:    "restic",
		Includes:  []string{"/home", "/etc"},
		Excludes:  []string{"/home/*/.cache"},
		Schedule:  &schedule,
		Retention: map[string]any{"daily": 7, "weekly": 4},
	}
	if err := store.Create(context.Background(), p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p.ID.String() == "" {
		t.Fatal("expected ID to be set after Create")
	}
	if p.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set after Create")
	}
}

func TestPolicyStore_GetByID_ReturnsPolicy_WhenExists(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewPolicyStore(db)

	created := &catalog.BackupPolicy{
		Name:      "daily-incremental",
		Engine:    "restic",
		Includes:  []string{"/var/data"},
		Retention: map[string]any{"daily": 14},
	}
	if err := store.Create(context.Background(), created); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "daily-incremental" {
		t.Errorf("Name: want daily-incremental, got %s", got.Name)
	}
	if got.Engine != "restic" {
		t.Errorf("Engine: want restic, got %s", got.Engine)
	}
	if len(got.Includes) != 1 || got.Includes[0] != "/var/data" {
		t.Errorf("Includes: want [/var/data], got %v", got.Includes)
	}
}

func TestPolicyStore_GetByID_ReturnsErrNotFound_WhenMissing(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewPolicyStore(db)

	_, err := store.GetByID(context.Background(), [16]byte{0xFF})
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestPolicyStore_List_ReturnsAllPolicies(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewPolicyStore(db)

	for _, name := range []string{"policy-a", "policy-b"} {
		if err := store.Create(context.Background(), &catalog.BackupPolicy{Name: name, Engine: "restic"}); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
	}

	policies, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(policies) != 2 {
		t.Errorf("want 2 policies, got %d", len(policies))
	}
}

func TestPolicyStore_Update_PersistsChanges(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewPolicyStore(db)

	p := &catalog.BackupPolicy{Name: "original", Engine: "restic"}
	if err := store.Create(context.Background(), p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	p.Name = "updated"
	p.Engine = "borg"
	if err := store.Update(context.Background(), p); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "updated" {
		t.Errorf("Name: want updated, got %s", got.Name)
	}
	if got.Engine != "borg" {
		t.Errorf("Engine: want borg, got %s", got.Engine)
	}
}

func TestPolicyStore_Delete_RemovesPolicy(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewPolicyStore(db)

	p := &catalog.BackupPolicy{Name: "to-delete", Engine: "restic"}
	if err := store.Create(context.Background(), p); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Delete(context.Background(), p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := store.GetByID(context.Background(), p.ID)
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound after Delete, got %v", err)
	}
}
