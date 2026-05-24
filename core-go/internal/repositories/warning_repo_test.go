package repositories

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestWarningRepo_InsertAndList(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	ctx := context.Background()

	hostID := int64(1)
	w := domain.Warning{
		ScopeType: domain.WarningScopeSkillHostFolder,
		ScopeID:   &hostID,
		Severity:  domain.WarningSeverityWarning,
		Code:      "skill_host_missing",
		Message:   "Host folder not found",
	}
	id, err := repo.Insert(ctx, w)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	list, err := repo.ListByScope(ctx, domain.WarningScopeSkillHostFolder, hostID, false)
	if err != nil {
		t.Fatalf("ListByScope: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(list))
	}
	if list[0].Code != "skill_host_missing" {
		t.Errorf("code: %q", list[0].Code)
	}
}

func TestWarningRepo_ClearByScope(t *testing.T) {
	db := NewTestDB(t)
	repo := NewWarningRepo(db)
	ctx := context.Background()

	hostID := int64(1)
	w := domain.Warning{
		ScopeType: domain.WarningScopeSkillHostFolder,
		ScopeID:   &hostID,
		Severity:  domain.WarningSeverityWarning,
		Code:      "test_warning",
		Message:   "Test",
	}
	_, _ = repo.Insert(ctx, w)

	if err := repo.ClearByScope(ctx, domain.WarningScopeSkillHostFolder, hostID); err != nil {
		t.Fatalf("ClearByScope: %v", err)
	}

	// Active warnings should be empty.
	list, _ := repo.ListByScope(ctx, domain.WarningScopeSkillHostFolder, hostID, false)
	if len(list) != 0 {
		t.Errorf("expected 0 active warnings after clear, got %d", len(list))
	}

	// Including resolved should return 1.
	all, _ := repo.ListByScope(ctx, domain.WarningScopeSkillHostFolder, hostID, true)
	if len(all) != 1 {
		t.Errorf("expected 1 resolved warning, got %d", len(all))
	}
}
