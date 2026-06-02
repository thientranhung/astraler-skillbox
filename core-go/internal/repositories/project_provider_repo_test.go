package repositories

import (
	"context"
	"database/sql"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// seedProjectProvider inserts a project_providers row and returns its id.
func seedProjectProvider(t *testing.T, db *sql.DB, projectID, providerDefinitionID int64) int64 {
	t.Helper()
	res, err := db.ExecContext(context.Background(),
		`INSERT INTO project_providers (project_id, provider_definition_id, detection_status)
		 VALUES (?, ?, 'detected')`, projectID, providerDefinitionID)
	if err != nil {
		t.Fatalf("seedProjectProvider: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func getGenericAgentsDefID(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	return getProviderDefID(t, db, "generic_agents")
}

func getProviderDefID(t *testing.T, db *sql.DB, key string) int64 {
	t.Helper()
	var id int64
	err := db.QueryRowContext(context.Background(),
		`SELECT id FROM provider_definitions WHERE key=?`, key).Scan(&id)
	if err != nil {
		t.Fatalf("getProviderDefID(%q): %v", key, err)
	}
	return id
}

func TestProjectProviderRepo_ListByProject_Empty(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	ppRepo := NewProjectProviderRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	rows, err := ppRepo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 providers, got %d", len(rows))
	}
}

func TestProjectProviderRepo_ListByProject_WithProvider(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	ppRepo := NewProjectProviderRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	seedProjectProvider(t, db, pid, defID)

	rows, err := ppRepo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(rows))
	}
	row := rows[0]
	if row.ProviderKey != "generic_agents" {
		t.Errorf("ProviderKey: got %q want generic_agents", row.ProviderKey)
	}
	if row.DetectionStatus != domain.DetectionStatusDetected {
		t.Errorf("DetectionStatus: got %q want detected", row.DetectionStatus)
	}
	if row.ProjectProviderID <= 0 {
		t.Errorf("ProjectProviderID: got %d, want positive", row.ProjectProviderID)
	}
}

func TestProjectProviderRepo_ListByProject_EntryCount(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	ppRepo := NewProjectProviderRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)

	// Insert two installs for this project_provider.
	for _, name := range []string{"skill-x", "skill-y"} {
		_, err := db.ExecContext(ctx,
			`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status, project_skill_path)
			 VALUES (?, ?, 'direct', 'current', ?)`, ppID, name, "/tmp/proj-a/.agents/skills/"+name)
		if err != nil {
			t.Fatalf("insert install: %v", err)
		}
	}

	rows, err := ppRepo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 provider row, got %d", len(rows))
	}
	if rows[0].EntryCount != 2 {
		t.Errorf("EntryCount: got %d want 2", rows[0].EntryCount)
	}
}

func TestProjectProviderRepo_ListByProject_EntryCountExcludesMissing(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	ppRepo := NewProjectProviderRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)

	_, err := db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES
		   (?, 'skill-current', 'direct', 'current', '/tmp/proj-a/.agents/skills/skill-current'),
		   (?, 'skill-missing', 'direct', 'missing', '/tmp/proj-a/.agents/skills/skill-missing')`,
		ppID, ppID)
	if err != nil {
		t.Fatalf("insert installs: %v", err)
	}

	rows, err := ppRepo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 provider row, got %d", len(rows))
	}
	if rows[0].EntryCount != 1 {
		t.Errorf("EntryCount: got %d want 1 (missing installs are historical)", rows[0].EntryCount)
	}
}

func TestProjectProviderRepo_ListByProject_OtherProjectIsolated(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	ppRepo := NewProjectProviderRepo(db)
	ctx := context.Background()

	pid1 := seedProject(t, projRepo, "proj-1", "/tmp/proj-1")
	pid2 := seedProject(t, projRepo, "proj-2", "/tmp/proj-2")
	defID := getGenericAgentsDefID(t, db)
	seedProjectProvider(t, db, pid1, defID)

	rows, err := ppRepo.ListByProject(ctx, pid2)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("expected 0 providers for proj-2, got %d", len(rows))
	}
}
