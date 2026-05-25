package repositories

import (
	"context"
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
