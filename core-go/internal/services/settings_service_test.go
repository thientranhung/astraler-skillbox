package services

import (
	"context"
	"testing"
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
