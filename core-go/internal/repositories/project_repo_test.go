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

func TestProjectRepo_List_HidesRemovedProjects(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	seedProject(t, repo, "proj-a", "/tmp/proj-a")
	idB := seedProject(t, repo, "proj-b", "/tmp/proj-b")
	if ok, err := repo.MarkRemoved(ctx, idB); err != nil || !ok {
		t.Fatalf("MarkRemoved: ok=%v err=%v", ok, err)
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 project (removed hidden), got %d", len(list))
	}
	if list[0].Path != "/tmp/proj-a" {
		t.Errorf("expected proj-a, got %q", list[0].Path)
	}
}

func TestProjectRepo_GetByID_ReturnsNilForRemoved(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id := seedProject(t, repo, "proj-a", "/tmp/proj-a")
	if ok, err := repo.MarkRemoved(ctx, id); err != nil || !ok {
		t.Fatalf("MarkRemoved: ok=%v err=%v", ok, err)
	}

	p, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if p != nil {
		t.Errorf("expected nil for removed project, got %+v", p)
	}
}

func TestProjectRepo_UpsertByPath_RevivesRemoved(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id := seedProject(t, repo, "proj-a", "/tmp/proj-a")
	if ok, err := repo.MarkRemoved(ctx, id); err != nil || !ok {
		t.Fatalf("MarkRemoved: ok=%v err=%v", ok, err)
	}

	id2, isNew, err := repo.UpsertByPath(ctx, "proj-a-renamed", "/tmp/proj-a")
	if err != nil {
		t.Fatalf("UpsertByPath revive: %v", err)
	}
	if isNew {
		t.Error("expected isNew=false on revival")
	}
	if id2 != id {
		t.Errorf("expected same id %d, got %d", id, id2)
	}

	p, _ := repo.GetByID(ctx, id2)
	if p == nil {
		t.Fatal("expected project to be visible after revival")
	}
	if p.Status != domain.ProjectStatusActive {
		t.Errorf("status: got %q want active", p.Status)
	}
	if p.Name != "proj-a-renamed" {
		t.Errorf("name: got %q want proj-a-renamed", p.Name)
	}
}

func TestProjectRepo_MarkRemoved_Success(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id := seedProject(t, repo, "proj-a", "/tmp/proj-a")
	ok, err := repo.MarkRemoved(ctx, id)
	if err != nil {
		t.Fatalf("MarkRemoved: %v", err)
	}
	if !ok {
		t.Error("expected ok=true for existing active project")
	}

	// Confirm via direct query since GetByID hides removed rows
	var status string
	if err := db.QueryRowContext(ctx, `SELECT status FROM projects WHERE id = ?`, id).Scan(&status); err != nil {
		t.Fatalf("query: %v", err)
	}
	if status != "removed" {
		t.Errorf("status: got %q want removed", status)
	}
}

func TestProjectRepo_MarkRemoved_Missing(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	ok, err := repo.MarkRemoved(ctx, 9999)
	if err != nil {
		t.Fatalf("expected no DB error for missing id, got: %v", err)
	}
	if ok {
		t.Error("expected ok=false for missing project id")
	}
}

func TestProjectRepo_CountActive(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	// Insert one active and one removed project via direct SQL.
	_, err := db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('active-proj', '/tmp/active', 'active')`)
	if err != nil {
		t.Fatalf("insert active project: %v", err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('removed-proj', '/tmp/removed', 'removed')`)
	if err != nil {
		t.Fatalf("insert removed project: %v", err)
	}

	count, err := repo.CountActive(ctx)
	if err != nil {
		t.Fatalf("CountActive: %v", err)
	}
	if count != 1 {
		t.Errorf("CountActive: got %d want 1 (removed excluded)", count)
	}
}

func TestProjectRepo_MarkRemoved_AlreadyRemoved(t *testing.T) {
	db := NewTestDB(t)
	repo := NewProjectRepo(db)
	ctx := context.Background()

	id := seedProject(t, repo, "proj-a", "/tmp/proj-a")
	if ok, err := repo.MarkRemoved(ctx, id); err != nil || !ok {
		t.Fatalf("first MarkRemoved: ok=%v err=%v", ok, err)
	}

	ok, err := repo.MarkRemoved(ctx, id)
	if err != nil {
		t.Fatalf("expected no DB error for already-removed, got: %v", err)
	}
	if ok {
		t.Error("expected ok=false for already-removed project")
	}
}
