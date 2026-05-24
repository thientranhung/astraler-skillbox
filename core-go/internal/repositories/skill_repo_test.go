package repositories

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func seedHost(t *testing.T, repo *SkillHostFolderRepo) int64 {
	t.Helper()
	id, _, err := repo.UpsertAndActivate(context.Background(), "host", "/tmp/host", "/tmp/host/.agents/skills")
	if err != nil {
		t.Fatalf("seedHost: %v", err)
	}
	return id
}

func TestSkillRepo_UpsertMany_And_List(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	hostID := seedHost(t, hostRepo)

	skills := []domain.Skill{
		{Name: "skill-a", RelativePath: ".agents/skills/skill-a", AbsolutePath: "/tmp/host/.agents/skills/skill-a", Status: domain.SkillStatusAvailable},
		{Name: "skill-b", RelativePath: ".agents/skills/skill-b", AbsolutePath: "/tmp/host/.agents/skills/skill-b", Status: domain.SkillStatusAvailable},
	}
	if err := skillRepo.UpsertMany(ctx, hostID, skills); err != nil {
		t.Fatalf("UpsertMany: %v", err)
	}

	listed, err := skillRepo.ListByHost(ctx, hostID)
	if err != nil {
		t.Fatalf("ListByHost: %v", err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(listed))
	}
}

func TestSkillRepo_UpsertMany_Updates(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()
	hostID := seedHost(t, hostRepo)

	skill := domain.Skill{Name: "skill-a", RelativePath: ".agents/skills/skill-a", AbsolutePath: "/tmp/host/.agents/skills/skill-a", Status: domain.SkillStatusAvailable}
	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{skill})

	skill.Status = domain.SkillStatusLocalModified
	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{skill})

	listed, _ := skillRepo.ListByHost(ctx, hostID)
	if len(listed) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(listed))
	}
	if listed[0].Status != domain.SkillStatusLocalModified {
		t.Errorf("status not updated: %q", listed[0].Status)
	}
}

func TestSkillRepo_MarkMissing(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()
	hostID := seedHost(t, hostRepo)

	skills := []domain.Skill{
		{Name: "a", RelativePath: ".agents/skills/a", AbsolutePath: "/tmp/a", Status: domain.SkillStatusAvailable},
		{Name: "b", RelativePath: ".agents/skills/b", AbsolutePath: "/tmp/b", Status: domain.SkillStatusAvailable},
	}
	_ = skillRepo.UpsertMany(ctx, hostID, skills)

	listed, _ := skillRepo.ListByHost(ctx, hostID)
	// Keep only the first skill's ID as "present".
	if err := skillRepo.MarkMissing(ctx, hostID, []int64{listed[0].ID}); err != nil {
		t.Fatalf("MarkMissing: %v", err)
	}

	after, _ := skillRepo.ListByHost(ctx, hostID)
	for _, s := range after {
		if s.Name == "a" && s.Status != domain.SkillStatusAvailable {
			t.Errorf("skill-a should remain available, got %q", s.Status)
		}
		if s.Name == "b" && s.Status != domain.SkillStatusMissing {
			t.Errorf("skill-b should be missing, got %q", s.Status)
		}
	}
}
