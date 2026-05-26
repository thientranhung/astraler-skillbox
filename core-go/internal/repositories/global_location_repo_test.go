package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestGlobalLocationRepo_ProviderDefByKey(t *testing.T) {
	db := NewTestDB(t)
	repo := NewGlobalLocationRepo(db)
	ctx := context.Background()

	id, displayName, status, err := repo.ProviderDefByKey(ctx, "generic_agents")
	if err != nil {
		t.Fatalf("ProviderDefByKey: %v", err)
	}
	if id == 0 {
		t.Error("id must be non-zero")
	}
	if displayName == "" {
		t.Error("displayName must not be empty")
	}
	if status == "" {
		t.Error("status must not be empty")
	}
}

func TestGlobalLocationRepo_ListForView(t *testing.T) {
	db := NewTestDB(t)
	scanRepo := NewGlobalScanRepo(db)
	listRepo := NewGlobalLocationRepo(db)
	ctx := context.Background()
	now := time.Now().UTC()

	var defID int64
	if err := db.QueryRow("SELECT id FROM provider_definitions WHERE key='generic_agents'").Scan(&defID); err != nil {
		t.Fatalf("get def id: %v", err)
	}

	rescan := "rescan"
	rootPath := "/home/.agents"
	skillsPath := "/home/.agents/skills"

	installs := []GlobalInstallScanResult{
		{
			SkillName:       "adr-helper",
			InstallMode:     domain.InstallModeSymlink,
			InstallStatus:   domain.InstallStatusCurrent,
			GlobalSkillPath: "/home/.agents/skills/adr-helper",
		},
		{
			SkillName:       "old-cmd",
			InstallMode:     domain.InstallModeSymlink,
			InstallStatus:   domain.InstallStatusBrokenSymlink,
			GlobalSkillPath: "/home/.agents/skills/old-cmd",
			Warning: &domain.Warning{
				ScopeType: domain.WarningScopeGlobalInstall,
				Severity:  domain.WarningSeverityWarning,
				Code:      "broken_symlink",
				Message:   "broken",
				ActionKey: &rescan,
			},
		},
	}

	if err := scanRepo.CommitGlobalScan(ctx, defID, &rootPath, &skillsPath,
		domain.GlobalLocationStatusActive, installs, nil, now); err != nil {
		t.Fatalf("CommitGlobalScan: %v", err)
	}

	locs, err := listRepo.ListForView(ctx)
	if err != nil {
		t.Fatalf("ListForView: %v", err)
	}

	if len(locs) != 1 {
		t.Fatalf("locations: got %d want 1", len(locs))
	}

	loc := locs[0]
	if loc.ProviderKey != "generic_agents" {
		t.Errorf("ProviderKey: got %q want %q", loc.ProviderKey, "generic_agents")
	}
	if loc.Status != domain.GlobalLocationStatusActive {
		t.Errorf("Status: got %q want active", loc.Status)
	}

	// Entries must be ordered by skill_name.
	if len(loc.Entries) != 2 {
		t.Fatalf("entries: got %d want 2", len(loc.Entries))
	}
	if loc.Entries[0].SkillName != "adr-helper" {
		t.Errorf("entries[0]: got %q want adr-helper", loc.Entries[0].SkillName)
	}
	if loc.Entries[1].SkillName != "old-cmd" {
		t.Errorf("entries[1]: got %q want old-cmd", loc.Entries[1].SkillName)
	}

	// One active warning for the broken symlink.
	if len(loc.Warnings) != 1 {
		t.Errorf("warnings: got %d want 1", len(loc.Warnings))
	} else if loc.Warnings[0].Code != "broken_symlink" {
		t.Errorf("warning code: got %q want broken_symlink", loc.Warnings[0].Code)
	}
}
