package services

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestSettingsService_Get_NoActiveHost(t *testing.T) {
	svc := NewSettingsService(newMockSettings(nil), newMockHostRepo())
	view, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.ActiveSkillHostFolderID != nil {
		t.Errorf("expected nil activeHostID, got %v", *view.ActiveSkillHostFolderID)
	}
	if view.ActiveHost != nil {
		t.Errorf("expected nil activeHost, got %v", view.ActiveHost)
	}
}

func TestSettingsService_Get_WithActiveHost(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	svc := NewSettingsService(newMockSettings(&hostID), hostRepo)
	view, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.ActiveHost == nil {
		t.Fatal("expected activeHost to be populated")
	}
	if view.ActiveHost.Path != "/tmp/host" {
		t.Errorf("path: %q", view.ActiveHost.Path)
	}
}

func TestSettingsService_Get_MissingHostPath(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	const ghostPath = "/nonexistent/skillbox/test/host/does-not-exist-ever"
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "ghost", ghostPath, ghostPath+"/.agents/skills")

	svc := NewSettingsService(newMockSettings(&hostID), hostRepo)
	view, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.ActiveHost == nil {
		t.Fatal("expected activeHost to be populated even when path is missing")
	}
	if view.ActiveHost.Status != domain.SkillHostStatusMissing {
		t.Errorf("expected status %q, got %q", domain.SkillHostStatusMissing, view.ActiveHost.Status)
	}
	if view.ActiveHost.Path != ghostPath {
		t.Errorf("path should be preserved: got %q", view.ActiveHost.Path)
	}
}

func TestSettingsService_Get_ExistingHostPath(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	realPath := t.TempDir()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "real", realPath, realPath+"/.agents/skills")

	svc := NewSettingsService(newMockSettings(&hostID), hostRepo)
	view, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if view.ActiveHost == nil {
		t.Fatal("expected activeHost to be populated")
	}
	if view.ActiveHost.Status != domain.SkillHostStatusActive {
		t.Errorf("expected status %q for existing path, got %q", domain.SkillHostStatusActive, view.ActiveHost.Status)
	}
}

func TestSettingsService_Get_OrphanActiveID(t *testing.T) {
	// Active ID points to a non-existent host row.
	orphanID := int64(99999)
	svc := NewSettingsService(newMockSettings(&orphanID), newMockHostRepo())
	view, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Should gracefully return nil activeHost, not panic.
	if view.ActiveHost != nil {
		t.Errorf("expected nil activeHost for orphan id, got %v", view.ActiveHost)
	}
	if view.ActiveSkillHostFolderID == nil || *view.ActiveSkillHostFolderID != orphanID {
		t.Errorf("activeSkillHostFolderID should still be set: %v", view.ActiveSkillHostFolderID)
	}
}
