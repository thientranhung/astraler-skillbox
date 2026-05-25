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
