package repositories

import (
	"context"
	"database/sql"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func seedInstall(t *testing.T, db *sql.DB, projectProviderID int64, name, path string) int64 {
	t.Helper()
	res, err := db.ExecContext(context.Background(),
		`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES (?, ?, 'direct', 'current', ?)`, projectProviderID, name, path)
	if err != nil {
		t.Fatalf("seedInstall: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestInstallRepo_ListByProject_Empty(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewInstallRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	installs, err := repo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(installs) != 0 {
		t.Errorf("expected 0 installs, got %d", len(installs))
	}
}

func TestInstallRepo_ListByProject_ReturnsInstalls(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewInstallRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)

	seedInstall(t, db, ppID, "skill-x", "/tmp/proj-a/.agents/skills/skill-x")
	seedInstall(t, db, ppID, "skill-y", "/tmp/proj-a/.agents/skills/skill-y")

	installs, err := repo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(installs) != 2 {
		t.Fatalf("expected 2 installs, got %d", len(installs))
	}
}

func TestInstallRepo_ListByProject_InstallFields(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewInstallRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)
	seedInstall(t, db, ppID, "skill-x", "/tmp/proj-a/.agents/skills/skill-x")

	installs, err := repo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(installs) != 1 {
		t.Fatalf("expected 1 install, got %d", len(installs))
	}
	inst := installs[0]
	if inst.SkillName != "skill-x" {
		t.Errorf("SkillName: got %q want skill-x", inst.SkillName)
	}
	if inst.InstallMode != domain.InstallModeDirect {
		t.Errorf("InstallMode: got %q want direct", inst.InstallMode)
	}
	if inst.InstallStatus != domain.InstallStatusCurrent {
		t.Errorf("InstallStatus: got %q want current", inst.InstallStatus)
	}
	if inst.ProjectProviderID != ppID {
		t.Errorf("ProjectProviderID: got %d want %d", inst.ProjectProviderID, ppID)
	}
}

func TestInstallRepo_ListByProject_OtherProjectIsolated(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewInstallRepo(db)
	ctx := context.Background()

	pid1 := seedProject(t, projRepo, "proj-1", "/tmp/proj-1")
	pid2 := seedProject(t, projRepo, "proj-2", "/tmp/proj-2")
	defID := getGenericAgentsDefID(t, db)
	ppID1 := seedProjectProvider(t, db, pid1, defID)
	seedInstall(t, db, ppID1, "skill-x", "/tmp/proj-1/.agents/skills/skill-x")

	installs, err := repo.ListByProject(ctx, pid2)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(installs) != 0 {
		t.Errorf("expected 0 installs for proj-2, got %d", len(installs))
	}
}

func TestInstallRepo_DeleteByID_DeletesOneRow(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewInstallRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)
	idX := seedInstall(t, db, ppID, "skill-x", "/tmp/proj-a/.agents/skills/skill-x")
	seedInstall(t, db, ppID, "skill-y", "/tmp/proj-a/.agents/skills/skill-y")

	n, err := repo.DeleteByID(ctx, idX)
	if err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}
	if n != 1 {
		t.Errorf("rowsAffected: got %d want 1", n)
	}
	installs, err := repo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(installs) != 1 || installs[0].SkillName != "skill-y" {
		t.Errorf("expected only skill-y to remain, got %+v", installs)
	}
}

func TestInstallRepo_CountByModeActive(t *testing.T) {
	db := NewTestDB(t)
	repo := NewInstallRepo(db)
	ctx := context.Background()

	defID := getGenericAgentsDefID(t, db)

	// Active project (id auto-assigned).
	res, err := db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('active', '/tmp/act', 'active')`)
	if err != nil {
		t.Fatalf("insert active project: %v", err)
	}
	activeProjID, _ := res.LastInsertId()

	// Removed project.
	res, err = db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('removed', '/tmp/rem', 'removed')`)
	if err != nil {
		t.Fatalf("insert removed project: %v", err)
	}
	removedProjID, _ := res.LastInsertId()

	// Providers for each project.
	res, err = db.ExecContext(ctx, `INSERT INTO project_providers (project_id, provider_definition_id) VALUES (?, ?)`, activeProjID, defID)
	if err != nil {
		t.Fatalf("insert active provider: %v", err)
	}
	activePP, _ := res.LastInsertId()

	res, err = db.ExecContext(ctx, `INSERT INTO project_providers (project_id, provider_definition_id) VALUES (?, ?)`, removedProjID, defID)
	if err != nil {
		t.Fatalf("insert removed provider: %v", err)
	}
	removedPP, _ := res.LastInsertId()

	// Two symlink installs for active project (unique paths).
	for i, name := range []string{"skill-s1", "skill-s2"} {
		path := "/tmp/act/.agents/skills/" + name
		_, err = db.ExecContext(ctx,
			`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status, project_skill_path)
			 VALUES (?, ?, 'symlink', 'current', ?)`, activePP, name, path)
		if err != nil {
			t.Fatalf("insert symlink install %d: %v", i, err)
		}
	}
	// One direct install for active project.
	_, err = db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES (?, ?, 'direct', 'current', ?)`, activePP, "skill-d", "/tmp/act/.agents/skills/skill-d")
	if err != nil {
		t.Fatalf("insert direct install: %v", err)
	}

	// One symlink install for removed project (must not be counted).
	_, err = db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES (?, ?, 'symlink', 'current', ?)`, removedPP, "skill-r", "/tmp/rem/.agents/skills/skill-r")
	if err != nil {
		t.Fatalf("insert removed project install: %v", err)
	}

	counts, err := repo.CountByModeActive(ctx)
	if err != nil {
		t.Fatalf("CountByModeActive: %v", err)
	}
	if counts.Symlink != 2 {
		t.Errorf("Symlink: got %d want 2", counts.Symlink)
	}
	if counts.RsyncCopy != 0 {
		t.Errorf("RsyncCopy: got %d want 0", counts.RsyncCopy)
	}
	if counts.Direct != 1 {
		t.Errorf("Direct: got %d want 1", counts.Direct)
	}
}

func TestInstallRepo_DeleteByID_AbsentIsNoOp(t *testing.T) {
	db := NewTestDB(t)
	repo := NewInstallRepo(db)
	n, err := repo.DeleteByID(context.Background(), 99999)
	if err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}
	if n != 0 {
		t.Errorf("rowsAffected: got %d want 0", n)
	}
}
