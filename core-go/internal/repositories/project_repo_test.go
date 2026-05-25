package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func seedProject(t *testing.T, repo *ProjectRepo, name, path string) int64 {
	t.Helper()
	id, _, err := repo.UpsertByPath(context.Background(), name, path)
	if err != nil {
		t.Fatalf("seedProject: %v", err)
	}
	return id
}

func TestProjectRepo_UpsertByPath_New(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id, isNew, err := repo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a")
	if err != nil {
		t.Fatalf("UpsertByPath: %v", err)
	}
	if !isNew {
		t.Error("expected isNew=true on first insert")
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}
}

func TestProjectRepo_UpsertByPath_Idempotent(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id1, _, _ := repo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a")
	id2, isNew, err := repo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a")
	if err != nil {
		t.Fatalf("second UpsertByPath: %v", err)
	}
	if isNew {
		t.Error("expected isNew=false on second call with same path")
	}
	if id1 != id2 {
		t.Errorf("expected same id, got %d vs %d", id1, id2)
	}
}

func TestProjectRepo_GetByID(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id, _, _ := repo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a")
	p, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if p == nil {
		t.Fatal("expected project, got nil")
	}
	if p.Path != "/tmp/proj-a" {
		t.Errorf("path: got %q want %q", p.Path, "/tmp/proj-a")
	}
	if p.Status != domain.ProjectStatusActive {
		t.Errorf("status: got %q want active", p.Status)
	}
}

func TestProjectRepo_GetByID_Missing(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	p, err := repo.GetByID(context.Background(), 9999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if p != nil {
		t.Errorf("expected nil for missing id, got %v", p)
	}
}

func TestProjectRepo_List(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	seedProject(t, repo, "proj-a", "/tmp/proj-a")
	seedProject(t, repo, "proj-b", "/tmp/proj-b")

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 projects, got %d", len(list))
	}
}

func TestProjectRepo_List_Empty(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	list, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d", len(list))
	}
}

func TestProjectRepo_UpdateStatus(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id, _, _ := repo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a")
	if err := repo.UpdateStatus(ctx, id, domain.ProjectStatusMissing); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	p, _ := repo.GetByID(ctx, id)
	if p.Status != domain.ProjectStatusMissing {
		t.Errorf("status: got %q want missing", p.Status)
	}
}

func TestProjectRepo_UpdateLastScannedAt(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id, _, _ := repo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a")
	now := time.Now().UTC().Truncate(time.Second)
	if err := repo.UpdateLastScannedAt(ctx, id, now); err != nil {
		t.Fatalf("UpdateLastScannedAt: %v", err)
	}
	p, _ := repo.GetByID(ctx, id)
	if p.LastScannedAt == nil {
		t.Fatal("expected LastScannedAt to be set")
	}
	if !p.LastScannedAt.Equal(now) {
		t.Errorf("last_scanned_at: got %v want %v", p.LastScannedAt, now)
	}
}
