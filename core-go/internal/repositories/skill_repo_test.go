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

func TestSkillRepo_CountByHost(t *testing.T) {
	db := NewTestDB(t)
	skillRepo := NewSkillRepo(db)
	hostRepo := NewSkillHostFolderRepo(db)
	ctx := context.Background()

	host1ID := seedHost(t, hostRepo)
	host2ID, _, err := hostRepo.UpsertAndActivate(ctx, "host2", "/tmp/host2", "/tmp/host2/.agents/skills")
	if err != nil {
		t.Fatalf("seedHost2: %v", err)
	}

	skills1 := []domain.Skill{
		{Name: "a", RelativePath: ".agents/skills/a", AbsolutePath: "/tmp/host/.agents/skills/a", Status: domain.SkillStatusAvailable},
		{Name: "b", RelativePath: ".agents/skills/b", AbsolutePath: "/tmp/host/.agents/skills/b", Status: domain.SkillStatusAvailable},
	}
	if err := skillRepo.UpsertMany(ctx, host1ID, skills1); err != nil {
		t.Fatalf("UpsertMany host1: %v", err)
	}

	skills2 := []domain.Skill{
		{Name: "x", RelativePath: ".agents/skills/x", AbsolutePath: "/tmp/host2/.agents/skills/x", Status: domain.SkillStatusAvailable},
		{Name: "y", RelativePath: ".agents/skills/y", AbsolutePath: "/tmp/host2/.agents/skills/y", Status: domain.SkillStatusAvailable},
		{Name: "z", RelativePath: ".agents/skills/z", AbsolutePath: "/tmp/host2/.agents/skills/z", Status: domain.SkillStatusAvailable},
	}
	if err := skillRepo.UpsertMany(ctx, host2ID, skills2); err != nil {
		t.Fatalf("UpsertMany host2: %v", err)
	}

	count1, err := skillRepo.CountByHost(ctx, host1ID)
	if err != nil {
		t.Fatalf("CountByHost host1: %v", err)
	}
	if count1 != 2 {
		t.Errorf("host1 count: got %d want 2", count1)
	}

	count2, err := skillRepo.CountByHost(ctx, host2ID)
	if err != nil {
		t.Fatalf("CountByHost host2: %v", err)
	}
	if count2 != 3 {
		t.Errorf("host2 count: got %d want 3", count2)
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
