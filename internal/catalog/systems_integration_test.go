//go:build integration

package catalog_test

import (
	"context"
	"testing"

	"github.com/cerberus8484/opensourcebackup/internal/catalog"
)

func TestSystemStore_Create_AssignsIDAndCreatedAt(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSystemStore(db)

	s := &catalog.System{
		Hostname:  "web-01.example.com",
		RiskClass: "standard",
		Tags:      map[string]any{"env": "prod"},
	}
	if err := store.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s.ID.String() == "" {
		t.Fatal("expected ID to be set after Create")
	}
	if s.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set after Create")
	}
}

func TestSystemStore_GetByID_ReturnsSystem_WhenExists(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSystemStore(db)

	created := &catalog.System{Hostname: "db-01", RiskClass: "critical"}
	if err := store.Create(context.Background(), created); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := store.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Hostname != "db-01" {
		t.Errorf("Hostname: want db-01, got %s", got.Hostname)
	}
	if got.RiskClass != "critical" {
		t.Errorf("RiskClass: want critical, got %s", got.RiskClass)
	}
}

func TestSystemStore_GetByID_ReturnsErrNotFound_WhenMissing(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSystemStore(db)

	_, err := store.GetByID(context.Background(), [16]byte{0xFF})
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSystemStore_List_ReturnsAllSystems(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSystemStore(db)

	for _, h := range []string{"host-a", "host-b", "host-c"} {
		if err := store.Create(context.Background(), &catalog.System{Hostname: h, RiskClass: "standard"}); err != nil {
			t.Fatalf("Create %s: %v", h, err)
		}
	}

	systems, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(systems) != 3 {
		t.Errorf("want 3 systems, got %d", len(systems))
	}
}

func TestSystemStore_Update_PersistsChanges(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSystemStore(db)

	s := &catalog.System{Hostname: "original", RiskClass: "standard"}
	if err := store.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}

	s.Hostname = "updated"
	s.RiskClass = "critical"
	if err := store.Update(context.Background(), s); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := store.GetByID(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Hostname != "updated" {
		t.Errorf("Hostname: want updated, got %s", got.Hostname)
	}
	if got.RiskClass != "critical" {
		t.Errorf("RiskClass: want critical, got %s", got.RiskClass)
	}
}

func TestSystemStore_Update_ReturnsErrNotFound_WhenMissing(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSystemStore(db)

	err := store.Update(context.Background(), &catalog.System{ID: [16]byte{0xFF}, Hostname: "ghost", RiskClass: "standard"})
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestSystemStore_Delete_RemovesSystem(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSystemStore(db)

	s := &catalog.System{Hostname: "to-delete", RiskClass: "standard"}
	if err := store.Create(context.Background(), s); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := store.Delete(context.Background(), s.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := store.GetByID(context.Background(), s.ID)
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound after Delete, got %v", err)
	}
}

func TestSystemStore_Delete_ReturnsErrNotFound_WhenMissing(t *testing.T) {
	db := newTestDB(t)
	truncateTables(t, db)
	store := catalog.NewSystemStore(db)

	err := store.Delete(context.Background(), [16]byte{0xFF})
	if err != catalog.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
