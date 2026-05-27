package repositories

import (
	"context"
	"testing"
)

func TestAppSettingsRepo_Get_Default(t *testing.T) {
	db := NewTestDB(t)
	repo := NewAppSettingsRepo(db)

	s, err := repo.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if s.ActiveSkillHostFolderID != nil {
		t.Errorf("expected nil activeHostID, got %v", *s.ActiveSkillHostFolderID)
	}
	if s.DefaultInstallMode != "symlink" {
		t.Errorf("defaultInstallMode: got %q want %q", s.DefaultInstallMode, "symlink")
	}
	if s.DatabaseVersion != 17 {
		t.Errorf("databaseVersion: got %d want 17", s.DatabaseVersion)
	}
}

func TestAppSettingsRepo_UpdateActiveHost(t *testing.T) {
	db := NewTestDB(t)
	repo := NewAppSettingsRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	hostID, _, err := hostRepo.UpsertAndActivate(ctx, "test", "/tmp/myhost", "/tmp/myhost/.agents/skills")
	if err != nil {
		t.Fatalf("UpsertAndActivate: %v", err)
	}

	s, _ := repo.Get(ctx)
	if s.ActiveSkillHostFolderID == nil || *s.ActiveSkillHostFolderID != hostID {
		t.Errorf("activeHostID after UpsertAndActivate: got %v want %d", s.ActiveSkillHostFolderID, hostID)
	}

	if err := repo.UpdateActiveHost(ctx, nil); err != nil {
		t.Fatalf("UpdateActiveHost(nil): %v", err)
	}
	s2, _ := repo.Get(ctx)
	if s2.ActiveSkillHostFolderID != nil {
		t.Errorf("expected nil after clear, got %v", *s2.ActiveSkillHostFolderID)
	}
}
