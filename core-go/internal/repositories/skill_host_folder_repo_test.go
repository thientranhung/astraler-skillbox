package repositories

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestSkillHostFolderRepo_UpsertAndActivate_New(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	id, isNew, err := repo.UpsertAndActivate(ctx, "myhost", "/tmp/myhost", "/tmp/myhost/.agents/skills")
	if err != nil {
		t.Fatalf("UpsertAndActivate: %v", err)
	}
	if !isNew {
		t.Error("expected isNew=true on first insert")
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	// app_settings should point to the new host.
	settingsRepo := NewAppSettingsRepo(db)
	s, _ := settingsRepo.Get(ctx)
	if s.ActiveSkillHostFolderID == nil || *s.ActiveSkillHostFolderID != id {
		t.Errorf("app_settings.active_id: got %v want %d", s.ActiveSkillHostFolderID, id)
	}
}

func TestSkillHostFolderRepo_UpsertAndActivate_Idempotent(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	id1, _, _ := repo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")
	id2, isNew, err := repo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")
	if err != nil {
		t.Fatalf("second UpsertAndActivate: %v", err)
	}
	if isNew {
		t.Error("expected isNew=false on second call with same path")
	}
	if id1 != id2 {
		t.Errorf("expected same id, got %d vs %d", id1, id2)
	}
}

func TestSkillHostFolderRepo_UpsertAndActivate_SwitchHost(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	id1, _, _ := repo.UpsertAndActivate(ctx, "host1", "/tmp/host1", "/tmp/host1/.agents/skills")
	id2, _, err := repo.UpsertAndActivate(ctx, "host2", "/tmp/host2", "/tmp/host2/.agents/skills")
	if err != nil {
		t.Fatalf("switch host: %v", err)
	}

	// host1 should now be inactive.
	h1, _ := repo.GetByID(ctx, id1)
	if h1.Status != domain.SkillHostStatusInactive {
		t.Errorf("host1 status: got %q want inactive", h1.Status)
	}

	// host2 should be active.
	h2, _ := repo.GetByID(ctx, id2)
	if h2.Status != domain.SkillHostStatusActive {
		t.Errorf("host2 status: got %q want active", h2.Status)
	}
}

// TC-SETUP-003 regression: after switching host, installs sourced from the old
// host must be reclassified to old_host without requiring a project rescan.
func TestSkillHostFolderRepo_UpsertAndActivate_SwitchHost_MarksInstallsOldHost(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	// Establish host-a as active.
	hostAID, _, _ := repo.UpsertAndActivate(ctx, "host-a", "/tmp/host-a", "/tmp/host-a/.agents/skills")

	// Insert a project and provider to satisfy the FK chain.
	projRepo := NewProjectRepo(db)
	pid := seedProject(t, projRepo, "proj-x", "/tmp/proj-x")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)

	// Insert installs linked to host-a with various statuses.
	seedInstallFromHost := func(name, status string, hostID int64) int64 {
		res, err := db.ExecContext(ctx,
			`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status,
			                       project_skill_path, installed_from_host_folder_id)
			 VALUES (?, ?, 'symlink', ?, ?, ?)`, ppID, name, status,
			"/tmp/proj-x/.agents/skills/"+name, hostID)
		if err != nil {
			t.Fatalf("seedInstallFromHost %s: %v", name, err)
		}
		id, _ := res.LastInsertId()
		return id
	}

	currentID := seedInstallFromHost("skill-a", "current", hostAID)
	outdatedID := seedInstallFromHost("skill-b", "outdated", hostAID)
	missingID := seedInstallFromHost("skill-c", "missing", hostAID)   // missing must NOT be reclassified
	needsSyncID := seedInstallFromHost("skill-d", "needs_sync", hostAID)

	// Switch to host-b.
	_, _, err := repo.UpsertAndActivate(ctx, "host-b", "/tmp/host-b", "/tmp/host-b/.agents/skills")
	if err != nil {
		t.Fatalf("UpsertAndActivate host-b: %v", err)
	}

	installRepo := NewInstallRepo(db)
	installs, _ := installRepo.ListByProject(ctx, pid)
	statusByID := make(map[int64]domain.InstallStatus)
	for _, inst := range installs {
		statusByID[inst.ID] = inst.InstallStatus
	}

	if statusByID[currentID] != domain.InstallStatusOldHost {
		t.Errorf("skill-a (current): got %q want old_host", statusByID[currentID])
	}
	if statusByID[outdatedID] != domain.InstallStatusOldHost {
		t.Errorf("skill-b (outdated): got %q want old_host", statusByID[outdatedID])
	}
	if statusByID[missingID] != domain.InstallStatusMissing {
		t.Errorf("skill-c (missing): got %q want missing (must not be reclassified)", statusByID[missingID])
	}
	if statusByID[needsSyncID] != domain.InstallStatusOldHost {
		t.Errorf("skill-d (needs_sync): got %q want old_host", statusByID[needsSyncID])
	}
}

// After re-point, installs from the NEW active host are not affected.
func TestSkillHostFolderRepo_UpsertAndActivate_SwitchHost_PreservesNewHostInstalls(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	hostAID, _, _ := repo.UpsertAndActivate(ctx, "host-a", "/tmp/host-a", "/tmp/host-a/.agents/skills")
	hostBID, _, _ := repo.UpsertAndActivate(ctx, "host-b", "/tmp/host-b", "/tmp/host-b/.agents/skills")
	// Now re-point to host-a; host-b becomes inactive.

	projRepo := NewProjectRepo(db)
	pid := seedProject(t, projRepo, "proj-x", "/tmp/proj-x")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)

	// Install linked to host-b (currently active before next switch).
	res, err := db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status,
		                       project_skill_path, installed_from_host_folder_id)
		 VALUES (?, 'skill-b', 'symlink', 'current', '/tmp/proj-x/.agents/skills/skill-b', ?)`,
		ppID, hostBID)
	if err != nil {
		t.Fatalf("insert install for host-b: %v", err)
	}
	bInstallID, _ := res.LastInsertId()

	// Install linked to host-a.
	res, err = db.ExecContext(ctx,
		`INSERT INTO installs (project_provider_id, skill_name, install_mode, install_status,
		                       project_skill_path, installed_from_host_folder_id)
		 VALUES (?, 'skill-a', 'symlink', 'current', '/tmp/proj-x/.agents/skills/skill-a', ?)`,
		ppID, hostAID)
	if err != nil {
		t.Fatalf("insert install for host-a: %v", err)
	}
	aInstallID, _ := res.LastInsertId()

	// Switch back to host-a; host-b becomes inactive.
	_, _, err = repo.UpsertAndActivate(ctx, "host-a", "/tmp/host-a", "/tmp/host-a/.agents/skills")
	if err != nil {
		t.Fatalf("re-point to host-a: %v", err)
	}

	installRepo := NewInstallRepo(db)
	installs, _ := installRepo.ListByProject(ctx, pid)
	statusByID := make(map[int64]domain.InstallStatus)
	for _, inst := range installs {
		statusByID[inst.ID] = inst.InstallStatus
	}

	if statusByID[bInstallID] != domain.InstallStatusOldHost {
		t.Errorf("host-b install: got %q want old_host", statusByID[bInstallID])
	}
	if statusByID[aInstallID] != domain.InstallStatusCurrent {
		t.Errorf("host-a install: got %q want current (new active host, must be untouched)", statusByID[aInstallID])
	}
}

func TestSkillHostFolderRepo_GetByPath_Missing(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	h, err := repo.GetByPath(context.Background(), "/tmp/nonexistent")
	if err != nil {
		t.Fatalf("GetByPath: %v", err)
	}
	if h != nil {
		t.Errorf("expected nil, got %v", h)
	}
}

func TestSkillHostFolderRepo_UpdateStatus(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	id, _, _ := repo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")
	if err := repo.UpdateStatus(ctx, id, domain.SkillHostStatusMissing); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
	h, _ := repo.GetByID(ctx, id)
	if h.Status != domain.SkillHostStatusMissing {
		t.Errorf("status: got %q want missing", h.Status)
	}
}

func TestSkillHostFolderRepo_ListAll_IncludesActiveAndInactive(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	// Insert two hosts; second activation makes host1 inactive.
	_, _, _ = repo.UpsertAndActivate(ctx, "host1", "/tmp/host1", "/tmp/host1/.agents/skills")
	_, _, _ = repo.UpsertAndActivate(ctx, "host2", "/tmp/host2", "/tmp/host2/.agents/skills")

	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 hosts, got %d", len(all))
	}

	statusByPath := make(map[string]domain.SkillHostStatus)
	for _, h := range all {
		statusByPath[h.Path] = h.Status
	}
	if statusByPath["/tmp/host1"] != domain.SkillHostStatusInactive {
		t.Errorf("host1 status: got %q want inactive", statusByPath["/tmp/host1"])
	}
	if statusByPath["/tmp/host2"] != domain.SkillHostStatusActive {
		t.Errorf("host2 status: got %q want active", statusByPath["/tmp/host2"])
	}
}

func TestSkillHostFolderRepo_ListAll_Empty(t *testing.T) {
	db := NewTestDB(t)
	repo := NewSkillHostFolderRepo(db)
	all, err := repo.ListAll(context.Background())
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(all))
	}
}
