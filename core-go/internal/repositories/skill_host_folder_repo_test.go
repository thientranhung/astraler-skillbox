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
