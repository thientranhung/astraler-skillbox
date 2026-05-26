package services

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestSkillLibraryService_List_EmptyHost(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	svc := NewSkillLibraryService(newMockSkillRepo(), hostRepo, &mockWarningRepo{})
	view, err := svc.List(ctx, hostID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(view.Skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(view.Skills))
	}
	if view.HostPath != "/tmp/host" {
		t.Errorf("hostPath: %q", view.HostPath)
	}
}

func TestSkillLibraryService_List_WithSkills(t *testing.T) {
	hostRepo := newMockHostRepo()
	skillRepo := newMockSkillRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{
		{Name: "skill-a", RelativePath: ".agents/skills/skill-a", AbsolutePath: "/tmp/a", Status: domain.SkillStatusAvailable},
		{Name: "skill-b", RelativePath: ".agents/skills/skill-b", AbsolutePath: "/tmp/b", Status: domain.SkillStatusMissing},
	})

	svc := NewSkillLibraryService(skillRepo, hostRepo, &mockWarningRepo{})
	view, err := svc.List(ctx, hostID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(view.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(view.Skills))
	}
	if view.Totals.Available != 1 {
		t.Errorf("totals.available: got %d want 1", view.Totals.Available)
	}
	if view.Totals.Missing != 1 {
		t.Errorf("totals.missing: got %d want 1", view.Totals.Missing)
	}
}

func TestSkillLibraryService_List_UnknownHost(t *testing.T) {
	svc := NewSkillLibraryService(newMockSkillRepo(), newMockHostRepo(), &mockWarningRepo{})
	_, err := svc.List(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for unknown host")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestSkillLibraryService_List_ProjectsUsingCount(t *testing.T) {
	hostRepo := newMockHostRepo()
	skillRepo := newMockSkillRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{
		{Name: "skill-a", RelativePath: ".agents/skills/skill-a", AbsolutePath: "/tmp/a", Status: domain.SkillStatusAvailable},
		{Name: "skill-b", RelativePath: ".agents/skills/skill-b", AbsolutePath: "/tmp/b", Status: domain.SkillStatusAvailable},
	})
	listed, _ := skillRepo.ListByHost(ctx, hostID)
	var idA, idB int64
	for _, s := range listed {
		if s.Name == "skill-a" {
			idA = s.ID
		} else {
			idB = s.ID
		}
	}
	skillRepo.projectCounts = map[int64]int{idA: 3, idB: 0}

	svc := NewSkillLibraryService(skillRepo, hostRepo, &mockWarningRepo{})
	view, err := svc.List(ctx, hostID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, item := range view.Skills {
		if item.Name == "skill-a" && item.ProjectsUsingCount != 3 {
			t.Errorf("skill-a ProjectsUsingCount: got %d want 3", item.ProjectsUsingCount)
		}
		if item.Name == "skill-b" && item.ProjectsUsingCount != 0 {
			t.Errorf("skill-b ProjectsUsingCount: got %d want 0", item.ProjectsUsingCount)
		}
	}
}

func TestSkillLibraryService_GetSkillDetail_Found(t *testing.T) {
	hostRepo := newMockHostRepo()
	skillRepo := newMockSkillRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	_ = skillRepo.UpsertMany(ctx, hostID, []domain.Skill{
		{SkillHostFolderID: hostID, Name: "skill-a", RelativePath: ".agents/skills/skill-a", AbsolutePath: "/tmp/a", Status: domain.SkillStatusAvailable},
	})
	listed, _ := skillRepo.ListByHost(ctx, hostID)
	skillID := listed[0].ID

	skillRepo.usages = []domain.SkillProjectUsage{
		{ProjectID: 1, ProjectName: "my-proj", ProjectProviderID: 1, ProviderKey: "generic_agents",
			ProviderDisplayName: "Shared Agent Skills (.agents)", Mode: "symlink", Status: "current", ProjectSkillPath: "/tmp/proj/.agents/skills/skill-a"},
	}

	svc := NewSkillLibraryService(skillRepo, hostRepo, &mockWarningRepo{})
	view, err := svc.GetSkillDetail(ctx, skillID)
	if err != nil {
		t.Fatalf("GetSkillDetail: %v", err)
	}
	if view.Skill.Name != "skill-a" {
		t.Errorf("Skill.Name: got %q want skill-a", view.Skill.Name)
	}
	if view.Skill.HostPath != "/tmp/host" {
		t.Errorf("Skill.HostPath: got %q want /tmp/host", view.Skill.HostPath)
	}
	if len(view.Projects) != 1 {
		t.Errorf("Projects: got %d want 1", len(view.Projects))
	}
	if view.Projects[0].ProviderDisplayName != "Shared Agent Skills (.agents)" {
		t.Errorf("ProviderDisplayName: got %q", view.Projects[0].ProviderDisplayName)
	}
}

func TestSkillLibraryService_GetSkillDetail_NotFound(t *testing.T) {
	svc := NewSkillLibraryService(newMockSkillRepo(), newMockHostRepo(), &mockWarningRepo{})
	_, err := svc.GetSkillDetail(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for unknown skill")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestSkillLibraryService_GetSkillDetail_NonPositiveID(t *testing.T) {
	svc := NewSkillLibraryService(newMockSkillRepo(), newMockHostRepo(), &mockWarningRepo{})
	_, err := svc.GetSkillDetail(context.Background(), 0)
	if err == nil {
		t.Fatal("expected error for id=0")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestSkillLibraryService_List_WithWarnings(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	warnRepo := &mockWarningRepo{
		warnings: []domain.Warning{
			{ScopeType: domain.WarningScopeSkillHostFolder, ScopeID: &hostID, Code: "test_warn", Message: "test", Severity: domain.WarningSeverityWarning},
		},
	}

	svc := NewSkillLibraryService(newMockSkillRepo(), hostRepo, warnRepo)
	view, err := svc.List(ctx, hostID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(view.Warnings) != 1 {
		t.Errorf("expected 1 warning, got %d", len(view.Warnings))
	}
}
