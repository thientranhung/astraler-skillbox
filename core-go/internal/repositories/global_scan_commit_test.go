package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestGlobalScanRepo_CommitGlobalScan_Basic(t *testing.T) {
	db := NewTestDB(t)
	repo := NewGlobalScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC()

	var defID int64
	if err := db.QueryRow("SELECT id FROM provider_definitions WHERE key='generic_agents'").Scan(&defID); err != nil {
		t.Fatalf("get provider def: %v", err)
	}

	skillsPath := "/home/.agents/skills"
	rootPath := "/home/.agents"

	skillID := int64(999) // intentionally non-FK to test nullable handling when skill doesn't exist
	// Use nil skill_id to avoid FK constraint issues
	installs := []GlobalInstallScanResult{
		{
			SkillID:         nil,
			SkillName:       "adr-helper",
			InstallMode:     domain.InstallModeSymlink,
			InstallStatus:   domain.InstallStatusCurrent,
			GlobalSkillPath: "/home/.agents/skills/adr-helper",
		},
		{
			SkillID:         nil,
			SkillName:       "plain-dir",
			InstallMode:     domain.InstallModeDirect,
			InstallStatus:   domain.InstallStatusCurrent,
			GlobalSkillPath: "/home/.agents/skills/plain-dir",
		},
	}
	_ = skillID

	if err := repo.CommitGlobalScan(ctx, defID, &rootPath, &skillsPath,
		domain.GlobalLocationStatusActive, installs, nil, now); err != nil {
		t.Fatalf("CommitGlobalScan: %v", err)
	}

	// Verify one location row.
	var locCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM global_provider_locations WHERE provider_definition_id=?", defID).Scan(&locCount); err != nil {
		t.Fatalf("count locations: %v", err)
	}
	if locCount != 1 {
		t.Errorf("locations: got %d want 1", locCount)
	}

	// Verify two install rows.
	var instCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM global_installs gi
		JOIN global_provider_locations gl ON gl.id=gi.global_provider_location_id
		WHERE gl.provider_definition_id=?`, defID).Scan(&instCount); err != nil {
		t.Fatalf("count installs: %v", err)
	}
	if instCount != 2 {
		t.Errorf("installs: got %d want 2", instCount)
	}
}

func TestGlobalScanRepo_CommitGlobalScan_Reconcile(t *testing.T) {
	db := NewTestDB(t)
	repo := NewGlobalScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC()

	var defID int64
	if err := db.QueryRow("SELECT id FROM provider_definitions WHERE key='generic_agents'").Scan(&defID); err != nil {
		t.Fatalf("get provider def: %v", err)
	}

	skillsPath := "/home/.agents/skills"
	rootPath := "/home/.agents"

	rescan := "rescan"

	// First scan: two installs.
	first := []GlobalInstallScanResult{
		{SkillName: "adr-helper", InstallMode: domain.InstallModeSymlink, InstallStatus: domain.InstallStatusCurrent, GlobalSkillPath: "/home/.agents/skills/adr-helper"},
		{SkillName: "old-cmd", InstallMode: domain.InstallModeSymlink, InstallStatus: domain.InstallStatusBrokenSymlink, GlobalSkillPath: "/home/.agents/skills/old-cmd",
			Warning: &domain.Warning{
				ScopeType: domain.WarningScopeGlobalInstall,
				Severity:  domain.WarningSeverityWarning,
				Code:      "broken_symlink",
				Message:   "Global skill old-cmd has a broken symlink",
				ActionKey: &rescan,
			}},
	}
	if err := repo.CommitGlobalScan(ctx, defID, &rootPath, &skillsPath,
		domain.GlobalLocationStatusActive, first, nil, now); err != nil {
		t.Fatalf("first CommitGlobalScan: %v", err)
	}

	// Verify warning on old-cmd install.
	var warnCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM warnings WHERE code='broken_symlink' AND is_resolved=0`).Scan(&warnCount); err != nil {
		t.Fatalf("count warnings: %v", err)
	}
	if warnCount != 1 {
		t.Errorf("warnings after first scan: got %d want 1", warnCount)
	}

	// Second scan: only adr-helper remains (old-cmd removed from disk).
	second := []GlobalInstallScanResult{
		{SkillName: "adr-helper", InstallMode: domain.InstallModeSymlink, InstallStatus: domain.InstallStatusCurrent, GlobalSkillPath: "/home/.agents/skills/adr-helper"},
	}
	if err := repo.CommitGlobalScan(ctx, defID, &rootPath, &skillsPath,
		domain.GlobalLocationStatusActive, second, nil, now.Add(time.Minute)); err != nil {
		t.Fatalf("second CommitGlobalScan: %v", err)
	}

	// old-cmd install must be deleted.
	var oldCmdCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM global_installs WHERE skill_name='old-cmd'`).Scan(&oldCmdCount); err != nil {
		t.Fatalf("count old-cmd: %v", err)
	}
	if oldCmdCount != 0 {
		t.Errorf("old-cmd install after second scan: got %d want 0", oldCmdCount)
	}

	// Prior broken_symlink warning must be resolved.
	if err := db.QueryRow(`SELECT COUNT(*) FROM warnings WHERE code='broken_symlink' AND is_resolved=0`).Scan(&warnCount); err != nil {
		t.Fatalf("count warnings after second: %v", err)
	}
	if warnCount != 0 {
		t.Errorf("stale broken_symlink warning not cleared: got %d want 0", warnCount)
	}
}

func TestGlobalScanRepo_CommitGlobalScan_LocationWarningClears(t *testing.T) {
	db := NewTestDB(t)
	repo := NewGlobalScanRepo(db)
	ctx := context.Background()
	now := time.Now().UTC()

	var defID int64
	if err := db.QueryRow("SELECT id FROM provider_definitions WHERE key='generic_agents'").Scan(&defID); err != nil {
		t.Fatalf("get provider def: %v", err)
	}

	rescan := "rescan"
	locWarn := domain.Warning{
		ScopeType: domain.WarningScopeGlobalProviderLocation,
		Severity:  domain.WarningSeverityWarning,
		Code:      "global_provider_location_missing",
		Message:   "~/.agents is missing",
		ActionKey: &rescan,
	}

	// Scan 1: missing status + location warning.
	if err := repo.CommitGlobalScan(ctx, defID, nil, nil,
		domain.GlobalLocationStatusMissing, nil, []domain.Warning{locWarn}, now); err != nil {
		t.Fatalf("scan 1: %v", err)
	}

	var wCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM warnings WHERE code='global_provider_location_missing' AND is_resolved=0`).Scan(&wCount); err != nil {
		t.Fatalf("count: %v", err)
	}
	if wCount != 1 {
		t.Errorf("scan1 warning: got %d want 1", wCount)
	}

	// Scan 2: active, no warnings — prior location warning must clear.
	if err := repo.CommitGlobalScan(ctx, defID, ptr("/home/.agents"), ptr("/home/.agents/skills"),
		domain.GlobalLocationStatusActive, nil, nil, now.Add(time.Minute)); err != nil {
		t.Fatalf("scan 2: %v", err)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM warnings WHERE code='global_provider_location_missing' AND is_resolved=0`).Scan(&wCount); err != nil {
		t.Fatalf("count after scan2: %v", err)
	}
	if wCount != 0 {
		t.Errorf("stale location warning not cleared: got %d want 0", wCount)
	}
}

func ptr(s string) *string { return &s }
