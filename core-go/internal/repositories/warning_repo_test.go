package repositories

import (
	"context"
	"database/sql"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestWarningRepo_InsertAndList(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	ctx := context.Background()

	hostID := int64(1)
	w := domain.Warning{
		ScopeType: domain.WarningScopeSkillHostFolder,
		ScopeID:   &hostID,
		Severity:  domain.WarningSeverityWarning,
		Code:      "skill_host_missing",
		Message:   "Host folder not found",
	}
	id, err := repo.Insert(ctx, w)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	list, err := repo.ListByScope(ctx, domain.WarningScopeSkillHostFolder, hostID, false)
	if err != nil {
		t.Fatalf("ListByScope: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(list))
	}
	if list[0].Code != "skill_host_missing" {
		t.Errorf("code: %q", list[0].Code)
	}
}

func TestWarningRepo_ClearByScope(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	ctx := context.Background()

	hostID := int64(1)
	w := domain.Warning{
		ScopeType: domain.WarningScopeSkillHostFolder,
		ScopeID:   &hostID,
		Severity:  domain.WarningSeverityWarning,
		Code:      "test_warning",
		Message:   "Test",
	}
	_, _ = repo.Insert(ctx, w)

	if err := repo.ClearByScope(ctx, domain.WarningScopeSkillHostFolder, hostID); err != nil {
		t.Fatalf("ClearByScope: %v", err)
	}

	// Active warnings should be empty.
	list, _ := repo.ListByScope(ctx, domain.WarningScopeSkillHostFolder, hostID, false)
	if len(list) != 0 {
		t.Errorf("expected 0 active warnings after clear, got %d", len(list))
	}

	// Including resolved should return 1.
	all, _ := repo.ListByScope(ctx, domain.WarningScopeSkillHostFolder, hostID, true)
	if len(all) != 1 {
		t.Errorf("expected 1 resolved warning, got %d", len(all))
	}
}

func TestWarningRepo_CountActiveForProject_AcrossAllScopes(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	projRepo := NewProjectRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)
	installID := seedInstall(t, db, ppID, "skill-x", "/tmp/proj-a/.agents/skills/skill-x")

	// Insert one warning per scope.
	scopeProject := int64(pid)
	_, _ = repo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeProject,
		ScopeID:   &scopeProject,
		Severity:  domain.WarningSeverityWarning,
		Code:      "project_missing",
		Message:   "Project missing",
	})

	scopePP := ppID
	_, _ = repo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeProjectProvider,
		ScopeID:   &scopePP,
		Severity:  domain.WarningSeverityWarning,
		Code:      "invalid_structure",
		Message:   "Invalid structure",
	})

	scopeInstall := installID
	_, _ = repo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeInstall,
		ScopeID:   &scopeInstall,
		Severity:  domain.WarningSeverityWarning,
		Code:      "broken_symlink",
		Message:   "Broken symlink",
	})

	count, err := repo.CountActiveForProject(ctx, pid)
	if err != nil {
		t.Fatalf("CountActiveForProject: %v", err)
	}
	if count != 3 {
		t.Errorf("count: got %d want 3", count)
	}
}

func TestWarningRepo_CountActiveForProject_ExcludesOtherProjects(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	projRepo := NewProjectRepo(db)
	ctx := context.Background()

	pid1 := seedProject(t, projRepo, "proj-1", "/tmp/proj-1")
	pid2 := seedProject(t, projRepo, "proj-2", "/tmp/proj-2")

	scopeID := int64(pid1)
	_, _ = repo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeProject,
		ScopeID:   &scopeID,
		Severity:  domain.WarningSeverityWarning,
		Code:      "project_missing",
		Message:   "Project missing",
	})

	count, err := repo.CountActiveForProject(ctx, pid2)
	if err != nil {
		t.Fatalf("CountActiveForProject: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 for proj-2, got %d", count)
	}
}

func TestWarningRepo_ListActiveForProject_AcrossAllScopes(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	projRepo := NewProjectRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)
	installID := seedInstall(t, db, ppID, "skill-x", "/tmp/proj-a/.agents/skills/skill-x")

	scopeProject := int64(pid)
	_, _ = repo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeProject,
		ScopeID:   &scopeProject,
		Severity:  domain.WarningSeverityWarning,
		Code:      "project_missing",
		Message:   "Project missing",
	})

	scopePP := ppID
	_, _ = repo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeProjectProvider,
		ScopeID:   &scopePP,
		Severity:  domain.WarningSeverityWarning,
		Code:      "invalid_structure",
		Message:   "Invalid structure",
	})

	scopeInstall := installID
	_, _ = repo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeInstall,
		ScopeID:   &scopeInstall,
		Severity:  domain.WarningSeverityWarning,
		Code:      "broken_symlink",
		Message:   "Broken symlink",
	})

	list, err := repo.ListActiveForProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListActiveForProject: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("expected 3 warnings, got %d", len(list))
	}

	codes := make(map[string]bool)
	for _, w := range list {
		codes[w.Code] = true
	}
	for _, code := range []string{"project_missing", "invalid_structure", "broken_symlink"} {
		if !codes[code] {
			t.Errorf("expected warning code %q in list", code)
		}
	}
}

// seedWarningDirect inserts a warning row directly via SQL and returns its id.
func seedWarningDirect(t *testing.T, db *sql.DB, scopeType string, scopeID *int64, severity, code string, isResolved int) int64 {
	t.Helper()
	var res sql.Result
	var err error
	if scopeID == nil {
		res, err = db.ExecContext(context.Background(),
			`INSERT INTO warnings (scope_type, scope_id, severity, code, message, is_resolved)
			 VALUES (?, NULL, ?, ?, 'msg', ?)`, scopeType, severity, code, isResolved)
	} else {
		res, err = db.ExecContext(context.Background(),
			`INSERT INTO warnings (scope_type, scope_id, severity, code, message, is_resolved)
			 VALUES (?, ?, ?, ?, 'msg', ?)`, scopeType, *scopeID, severity, code, isResolved)
	}
	if err != nil {
		t.Fatalf("seedWarningDirect: %v", err)
	}
	id, _ := res.LastInsertId()
	return id
}

func TestWarningRepo_ExcludeRemovedProject(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	ctx := context.Background()

	// Seed projects.
	activeProj := int64(0)
	res, err := db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('active', '/tmp/active', 'active')`)
	if err != nil {
		t.Fatalf("insert active project: %v", err)
	}
	activeProj, _ = res.LastInsertId()

	res, err = db.ExecContext(ctx, `INSERT INTO projects (name, path, status) VALUES ('removed', '/tmp/removed', 'removed')`)
	if err != nil {
		t.Fatalf("insert removed project: %v", err)
	}
	removedProj, _ := res.LastInsertId()

	// Seed project_providers.
	defID := getGenericAgentsDefID(t, db)
	res, err = db.ExecContext(ctx, `INSERT INTO project_providers (project_id, provider_definition_id) VALUES (?, ?)`, activeProj, defID)
	if err != nil {
		t.Fatalf("insert active provider: %v", err)
	}
	activePP, _ := res.LastInsertId()

	res, err = db.ExecContext(ctx, `INSERT INTO project_providers (project_id, provider_definition_id) VALUES (?, ?)`, removedProj, defID)
	if err != nil {
		t.Fatalf("insert removed provider: %v", err)
	}
	removedPP, _ := res.LastInsertId()

	// Seed installs.
	res, err = db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES (?, 'sk', 'direct', 'current', '/tmp/active/.agents/skills/sk')`, activePP)
	if err != nil {
		t.Fatalf("insert active install: %v", err)
	}
	activeInstall, _ := res.LastInsertId()

	res, err = db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status, project_skill_path)
		 VALUES (?, 'sk', 'direct', 'current', '/tmp/removed/.agents/skills/sk')`, removedPP)
	if err != nil {
		t.Fatalf("insert removed install: %v", err)
	}
	removedInstall, _ := res.LastInsertId()

	// Seed warnings according to spec.
	sid1 := activeProj
	seedWarningDirect(t, db, "project", &sid1, "warning", "w1", 0)           // 1 KEEP
	sid2 := removedProj
	seedWarningDirect(t, db, "project", &sid2, "warning", "w2", 0)           // 2 EXCLUDE removed project
	sid3 := removedPP
	seedWarningDirect(t, db, "project_provider", &sid3, "warning", "w3", 0)  // 3 EXCLUDE provider of removed
	sid4 := removedInstall
	seedWarningDirect(t, db, "install", &sid4, "warning", "w4", 0)           // 4 EXCLUDE install of removed
	hostID := int64(1)
	seedWarningDirect(t, db, "skill_host_folder", &hostID, "error", "w5", 0) // 5 KEEP
	seedWarningDirect(t, db, "app", nil, "info", "w6", 0)                    // 6 KEEP (NULL scope_id)
	sid7 := activeInstall
	seedWarningDirect(t, db, "install", &sid7, "blocking", "w7", 0)          // 7 KEEP (install of active)
	seedWarningDirect(t, db, "app", nil, "critical", "w8", 0)                // 8 KEEP in ListActive, NOT bucketed in Count
	sid9 := activeProj
	seedWarningDirect(t, db, "project", &sid9, "warning", "w9", 1)           // 9 EXCLUDE (resolved)

	// CountActiveBySeverity: {Info:1, Warning:1, Error:1, Blocking:1} — critical not bucketed
	counts, err := repo.CountActiveBySeverity(ctx)
	if err != nil {
		t.Fatalf("CountActiveBySeverity: %v", err)
	}
	if counts.Info != 1 {
		t.Errorf("Info: got %d want 1", counts.Info)
	}
	if counts.Warning != 1 {
		t.Errorf("Warning: got %d want 1", counts.Warning)
	}
	if counts.Error != 1 {
		t.Errorf("Error: got %d want 1", counts.Error)
	}
	if counts.Blocking != 1 {
		t.Errorf("Blocking: got %d want 1", counts.Blocking)
	}
	if total := counts.Total(); total != 4 {
		t.Errorf("Total: got %d want 4", total)
	}

	// ListActive(50): 5 rows (1,5,6,7,8), id-DESC order
	list, err := repo.ListActive(ctx, 50)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("ListActive len: got %d want 5", len(list))
	}
	for i := 1; i < len(list); i++ {
		if list[i].ID >= list[i-1].ID {
			t.Errorf("ListActive not in DESC order at index %d: id %d >= %d", i, list[i].ID, list[i-1].ID)
		}
	}
}

func TestWarningRepo_ListActive_Limit(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	ctx := context.Background()

	// Insert 10 active warnings.
	for i := 0; i < 10; i++ {
		_, err := db.ExecContext(ctx,
			`INSERT INTO warnings (scope_type, scope_id, severity, code, message, is_resolved)
			 VALUES ('app', NULL, 'info', 'code', 'msg', 0)`)
		if err != nil {
			t.Fatalf("insert warning %d: %v", i, err)
		}
	}

	list, err := repo.ListActive(ctx, 5)
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(list) != 5 {
		t.Errorf("ListActive(5): got %d want 5", len(list))
	}
}

func TestWarningRepo_CountActiveForProject_ExcludesResolved(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	projRepo := NewProjectRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	scopeID := int64(pid)
	_, _ = repo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeProject,
		ScopeID:   &scopeID,
		Severity:  domain.WarningSeverityWarning,
		Code:      "project_missing",
		Message:   "Project missing",
	})
	_ = repo.ClearByScope(ctx, domain.WarningScopeProject, scopeID)

	count, err := repo.CountActiveForProject(ctx, pid)
	if err != nil {
		t.Fatalf("CountActiveForProject: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 after resolve, got %d", count)
	}
}
