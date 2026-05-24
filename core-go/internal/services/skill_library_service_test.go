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
