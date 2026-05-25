package repositories

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// --- test helpers ---

func getProjectStatus(t *testing.T, db *sql.DB, projectID int64) domain.ProjectStatus {
	t.Helper()
	var status domain.ProjectStatus
	if err := db.QueryRowContext(context.Background(),
		`SELECT status FROM projects WHERE id=?`, projectID).Scan(&status); err != nil {
		t.Fatalf("getProjectStatus: %v", err)
	}
	return status
}

func getProjectLastScannedAt(t *testing.T, db *sql.DB, projectID int64) *string {
	t.Helper()
	var s sql.NullString
	if err := db.QueryRowContext(context.Background(),
		`SELECT last_scanned_at FROM projects WHERE id=?`, projectID).Scan(&s); err != nil {
		t.Fatalf("getProjectLastScannedAt: %v", err)
	}
	if s.Valid {
		return &s.String
	}
	return nil
}

func countProjectProviders(t *testing.T, db *sql.DB, projectID int64) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM project_providers WHERE project_id=?`, projectID).Scan(&n); err != nil {
		t.Fatalf("countProjectProviders: %v", err)
	}
	return n
}

func getProviderDetectionStatus(t *testing.T, db *sql.DB, projectID, defID int64) domain.DetectionStatus {
	t.Helper()
	var status domain.DetectionStatus
	if err := db.QueryRowContext(context.Background(),
		`SELECT detection_status FROM project_providers WHERE project_id=? AND provider_definition_id=?`,
		projectID, defID).Scan(&status); err != nil {
		t.Fatalf("getProviderDetectionStatus: %v", err)
	}
	return status
}

func countInstallsForProvider(t *testing.T, db *sql.DB, ppID int64) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM installs WHERE project_provider_id=?`, ppID).Scan(&n); err != nil {
		t.Fatalf("countInstallsForProvider: %v", err)
	}
	return n
}

func getInstallStatus(t *testing.T, db *sql.DB, ppID int64, path string) domain.InstallStatus {
	t.Helper()
	var status domain.InstallStatus
	if err := db.QueryRowContext(context.Background(),
		`SELECT install_status FROM installs WHERE project_provider_id=? AND project_skill_path=?`,
		ppID, path).Scan(&status); err != nil {
		t.Fatalf("getInstallStatus(%q): %v", path, err)
	}
	return status
}

func getProjectProviderID(t *testing.T, db *sql.DB, projectID, defID int64) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRowContext(context.Background(),
		`SELECT id FROM project_providers WHERE project_id=? AND provider_definition_id=?`,
		projectID, defID).Scan(&id); err != nil {
		t.Fatalf("getProjectProviderID: %v", err)
	}
	return id
}

func countActiveWarningsByScopeAndID(t *testing.T, db *sql.DB, scopeType domain.WarningScopeType, scopeID int64) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM warnings WHERE scope_type=? AND scope_id=? AND is_resolved=0`,
		string(scopeType), scopeID).Scan(&n); err != nil {
		t.Fatalf("countActiveWarningsByScopeAndID: %v", err)
	}
	return n
}

func strPtr(s string) *string { return &s }

// --- CommitProjectScan tests ---

func TestProjectScanRepo_CommitProjectScan_UpsertsProviderAndInstalls(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)

	err := repo.CommitProjectScan(ctx, pid, []ProviderScanResult{
		{
			ProviderDefinitionID: defID,
			DetectedPath:         strPtr("/tmp/proj-a/.agents"),
			SkillsPath:           strPtr("/tmp/proj-a/.agents/skills"),
			DetectionStatus:      domain.DetectionStatusDetected,
			Installs: []InstallScanResult{
				{SkillName: "skill-x", InstallMode: domain.InstallModeDirect, InstallStatus: domain.InstallStatusCurrent, ProjectSkillPath: "/tmp/proj-a/.agents/skills/skill-x"},
				{SkillName: "skill-y", InstallMode: domain.InstallModeSymlink, InstallStatus: domain.InstallStatusCurrent, ProjectSkillPath: "/tmp/proj-a/.agents/skills/skill-y"},
			},
		},
	}, nil, now)
	if err != nil {
		t.Fatalf("CommitProjectScan: %v", err)
	}

	if countProjectProviders(t, db, pid) != 1 {
		t.Errorf("expected 1 project_provider")
	}
	ppID := getProjectProviderID(t, db, pid, defID)
	if countInstallsForProvider(t, db, ppID) != 2 {
		t.Errorf("expected 2 installs")
	}
	if getProjectStatus(t, db, pid) != domain.ProjectStatusActive {
		t.Errorf("expected project status active")
	}
	if getProjectLastScannedAt(t, db, pid) == nil {
		t.Error("expected last_scanned_at to be set")
	}
}

func TestProjectScanRepo_CommitProjectScan_Idempotent(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)

	providers := []ProviderScanResult{{
		ProviderDefinitionID: defID,
		DetectedPath:         strPtr("/tmp/proj-a/.agents"),
		SkillsPath:           strPtr("/tmp/proj-a/.agents/skills"),
		DetectionStatus:      domain.DetectionStatusDetected,
		Installs: []InstallScanResult{
			{SkillName: "skill-x", InstallMode: domain.InstallModeDirect, InstallStatus: domain.InstallStatusCurrent, ProjectSkillPath: "/tmp/proj-a/.agents/skills/skill-x"},
		},
	}}
	_ = repo.CommitProjectScan(ctx, pid, providers, nil, now)
	_ = repo.CommitProjectScan(ctx, pid, providers, nil, now.Add(time.Minute))

	if countProjectProviders(t, db, pid) != 1 {
		t.Errorf("expected 1 project_provider after two scans")
	}
	ppID := getProjectProviderID(t, db, pid, defID)
	if countInstallsForProvider(t, db, ppID) != 1 {
		t.Errorf("expected 1 install after two scans")
	}
}

func TestProjectScanRepo_CommitProjectScan_MarksAbsentInstallsMissing(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)

	pathX := "/tmp/proj-a/.agents/skills/skill-x"
	pathY := "/tmp/proj-a/.agents/skills/skill-y"

	// First scan: two installs.
	_ = repo.CommitProjectScan(ctx, pid, []ProviderScanResult{{
		ProviderDefinitionID: defID,
		DetectedPath:         strPtr("/tmp/proj-a/.agents"),
		SkillsPath:           strPtr("/tmp/proj-a/.agents/skills"),
		DetectionStatus:      domain.DetectionStatusDetected,
		Installs: []InstallScanResult{
			{SkillName: "skill-x", InstallMode: domain.InstallModeDirect, InstallStatus: domain.InstallStatusCurrent, ProjectSkillPath: pathX},
			{SkillName: "skill-y", InstallMode: domain.InstallModeDirect, InstallStatus: domain.InstallStatusCurrent, ProjectSkillPath: pathY},
		},
	}}, nil, now)

	ppID := getProjectProviderID(t, db, pid, defID)

	// Second scan: only skill-x present.
	_ = repo.CommitProjectScan(ctx, pid, []ProviderScanResult{{
		ProviderDefinitionID: defID,
		DetectedPath:         strPtr("/tmp/proj-a/.agents"),
		SkillsPath:           strPtr("/tmp/proj-a/.agents/skills"),
		DetectionStatus:      domain.DetectionStatusDetected,
		Installs: []InstallScanResult{
			{SkillName: "skill-x", InstallMode: domain.InstallModeDirect, InstallStatus: domain.InstallStatusCurrent, ProjectSkillPath: pathX},
		},
	}}, nil, now.Add(time.Minute))

	// skill-y should be missing, not deleted.
	if countInstallsForProvider(t, db, ppID) != 2 {
		t.Errorf("expected 2 installs (no hard delete), got %d", countInstallsForProvider(t, db, ppID))
	}
	if getInstallStatus(t, db, ppID, pathY) != domain.InstallStatusMissing {
		t.Errorf("skill-y should be missing")
	}
	if getInstallStatus(t, db, ppID, pathX) != domain.InstallStatusCurrent {
		t.Errorf("skill-x should be current")
	}
}

func TestProjectScanRepo_CommitProjectScan_MarksAbsentProvidersMissing(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)

	// First scan: provider detected.
	_ = repo.CommitProjectScan(ctx, pid, []ProviderScanResult{{
		ProviderDefinitionID: defID,
		DetectedPath:         strPtr("/tmp/proj-a/.agents"),
		SkillsPath:           strPtr("/tmp/proj-a/.agents/skills"),
		DetectionStatus:      domain.DetectionStatusDetected,
	}}, nil, now)

	if getProviderDetectionStatus(t, db, pid, defID) != domain.DetectionStatusDetected {
		t.Errorf("expected detected after first scan")
	}

	// Second scan: no providers (provider disappeared).
	_ = repo.CommitProjectScan(ctx, pid, nil, nil, now.Add(time.Minute))

	// Provider row still exists but marked missing.
	if countProjectProviders(t, db, pid) != 1 {
		t.Errorf("expected provider row to remain (no hard delete)")
	}
	if getProviderDetectionStatus(t, db, pid, defID) != domain.DetectionStatusMissing {
		t.Errorf("expected detection_status=missing after provider disappears")
	}
}

func TestProjectScanRepo_CommitProjectScan_ClearsAndInsertsProjectWarnings(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")

	// Pre-insert a stale project warning.
	warningRepo := NewWarningRepo(db)
	scopeID := int64(pid)
	_, _ = warningRepo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeProject, ScopeID: &scopeID,
		Severity: domain.WarningSeverityWarning, Code: "stale_warning", Message: "old",
	})

	actionKey := "rescan"
	projectWarnings := []domain.Warning{{
		ScopeType: domain.WarningScopeProject,
		Severity:  domain.WarningSeverityWarning,
		Code:      "no_provider_detected",
		Message:   "No provider detected",
		ActionKey: &actionKey,
	}}

	if err := repo.CommitProjectScan(ctx, pid, nil, projectWarnings, now); err != nil {
		t.Fatalf("CommitProjectScan: %v", err)
	}

	// Old warning should be resolved.
	old, _ := warningRepo.ListByScope(ctx, domain.WarningScopeProject, int64(pid), true)
	resolved := 0
	for _, w := range old {
		if w.Code == "stale_warning" && w.IsResolved {
			resolved++
		}
	}
	if resolved != 1 {
		t.Errorf("expected stale_warning to be resolved, found %d resolved", resolved)
	}

	// New warning should be active.
	if countActiveWarningsByScopeAndID(t, db, domain.WarningScopeProject, int64(pid)) != 1 {
		t.Errorf("expected 1 active project-scope warning")
	}
}

func TestProjectScanRepo_CommitProjectScan_ProviderAndInstallWarningsScopedCorrectly(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)

	ppWarnCode := "invalid_structure"
	installWarnCode := "broken_symlink"
	rescan := "rescan"
	openFolder := "open_folder"

	err := repo.CommitProjectScan(ctx, pid, []ProviderScanResult{
		{
			ProviderDefinitionID: defID,
			DetectionStatus:      domain.DetectionStatusInvalidStructure,
			Warnings: []domain.Warning{{
				ScopeType: domain.WarningScopeProjectProvider,
				Severity:  domain.WarningSeverityWarning,
				Code:      ppWarnCode,
				Message:   "Invalid structure",
				ActionKey: &rescan,
			}},
			Installs: []InstallScanResult{
				{
					SkillName:        "ghost",
					InstallMode:      domain.InstallModeSymlink,
					InstallStatus:    domain.InstallStatusBrokenSymlink,
					ProjectSkillPath: "/tmp/proj-a/.agents/skills/ghost",
					Warning: &domain.Warning{
						ScopeType: domain.WarningScopeInstall,
						Severity:  domain.WarningSeverityWarning,
						Code:      installWarnCode,
						Message:   "Symlink broken",
						ActionKey: &openFolder,
					},
				},
			},
		},
	}, nil, now)
	if err != nil {
		t.Fatalf("CommitProjectScan: %v", err)
	}

	ppID := getProjectProviderID(t, db, pid, defID)
	if countActiveWarningsByScopeAndID(t, db, domain.WarningScopeProjectProvider, ppID) != 1 {
		t.Errorf("expected 1 active project_provider warning scoped to ppID=%d", ppID)
	}

	// Find install id.
	var installID int64
	if err := db.QueryRowContext(ctx,
		`SELECT id FROM installs WHERE project_provider_id=? AND project_skill_path=?`,
		ppID, "/tmp/proj-a/.agents/skills/ghost").Scan(&installID); err != nil {
		t.Fatalf("find install: %v", err)
	}
	if countActiveWarningsByScopeAndID(t, db, domain.WarningScopeInstall, installID) != 1 {
		t.Errorf("expected 1 active install warning scoped to installID=%d", installID)
	}
}

func TestProjectScanRepo_CommitProjectScan_UpdatesProjectStatus(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")

	if err := repo.CommitProjectScan(ctx, pid, nil, nil, now); err != nil {
		t.Fatalf("CommitProjectScan: %v", err)
	}
	if getProjectStatus(t, db, pid) != domain.ProjectStatusActive {
		t.Errorf("expected status=active")
	}
	if getProjectLastScannedAt(t, db, pid) == nil {
		t.Error("expected last_scanned_at to be set")
	}
}

// --- CommitProjectTerminal tests ---

func TestProjectScanRepo_CommitProjectTerminal_UpdatesStatusAndWarning(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	rescan := "rescan"
	w := &domain.Warning{
		ScopeType: domain.WarningScopeProject,
		Severity:  domain.WarningSeverityWarning,
		Code:      "project_missing",
		Message:   "Project missing",
		ActionKey: &rescan,
	}

	if err := repo.CommitProjectTerminal(ctx, pid, domain.ProjectStatusMissing, w, now); err != nil {
		t.Fatalf("CommitProjectTerminal: %v", err)
	}

	if getProjectStatus(t, db, pid) != domain.ProjectStatusMissing {
		t.Errorf("expected status=missing")
	}
	if countActiveWarningsByScopeAndID(t, db, domain.WarningScopeProject, int64(pid)) != 1 {
		t.Errorf("expected 1 active project warning")
	}
}

func TestProjectScanRepo_CommitProjectTerminal_NoProviderOrInstallMutation(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)

	// Pre-seed a provider and install.
	ppID := seedProjectProvider(t, db, pid, defID)
	seedInstall(t, db, ppID, "skill-x", "/tmp/proj-a/.agents/skills/skill-x")

	if err := repo.CommitProjectTerminal(ctx, pid, domain.ProjectStatusMissing, nil, now); err != nil {
		t.Fatalf("CommitProjectTerminal: %v", err)
	}

	// Provider and install rows untouched.
	if getProviderDetectionStatus(t, db, pid, defID) != domain.DetectionStatusDetected {
		t.Errorf("terminal must not mutate project_providers")
	}
	if getInstallStatus(t, db, ppID, "/tmp/proj-a/.agents/skills/skill-x") != domain.InstallStatusCurrent {
		t.Errorf("terminal must not mutate installs")
	}
}

func TestProjectScanRepo_CommitProjectTerminal_ClearsOldProjectWarning(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	warningRepo := NewWarningRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	scopeID := int64(pid)
	_, _ = warningRepo.Insert(ctx, domain.Warning{
		ScopeType: domain.WarningScopeProject, ScopeID: &scopeID,
		Severity: domain.WarningSeverityWarning, Code: "old_warning", Message: "old",
	})

	rescan := "rescan"
	_ = repo.CommitProjectTerminal(ctx, pid, domain.ProjectStatusMissing, &domain.Warning{
		ScopeType: domain.WarningScopeProject,
		Severity:  domain.WarningSeverityWarning,
		Code:      "project_missing",
		Message:   "missing",
		ActionKey: &rescan,
	}, now)

	all, _ := warningRepo.ListByScope(ctx, domain.WarningScopeProject, int64(pid), true)
	active := 0
	for _, w := range all {
		if !w.IsResolved {
			active++
		}
	}
	if active != 1 {
		t.Errorf("expected 1 active warning after terminal commit, got %d", active)
	}
}

func TestProjectScanRepo_CommitProjectTerminal_NilWarningIsOK(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewProjectScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	if err := repo.CommitProjectTerminal(ctx, pid, domain.ProjectStatusUnreadable, nil, now); err != nil {
		t.Fatalf("CommitProjectTerminal with nil warning: %v", err)
	}
	if getProjectStatus(t, db, pid) != domain.ProjectStatusUnreadable {
		t.Errorf("expected status=unreadable")
	}
}
