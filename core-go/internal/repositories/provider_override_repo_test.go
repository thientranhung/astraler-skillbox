package repositories

import (
	"context"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func providerIDByKey(t *testing.T, r *ProviderOverrideRepo, ctx context.Context, key string) int64 {
	t.Helper()
	id, err := r.GetProviderIDByKey(ctx, key)
	if err != nil {
		t.Fatalf("GetProviderIDByKey(%q): %v", key, err)
	}
	if id == 0 {
		t.Fatalf("GetProviderIDByKey(%q): not found", key)
	}
	return id
}

func TestProviderOverrideRepo_Upsert_And_ListAll(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	provID := providerIDByKey(t, r, ctx, "claude")

	err := r.Upsert(ctx, domain.ProviderPathOverride{
		ProviderDefinitionID: provID,
		Scope:                "project",
		Purpose:              "detect",
		Paths:                []string{".custom-claude", ".claude-alt"},
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	all, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("ListAll len: got %d want 1", len(all))
	}
	got := all[0]
	if got.ProviderDefinitionID != provID {
		t.Errorf("ProviderDefinitionID: got %d want %d", got.ProviderDefinitionID, provID)
	}
	if got.Scope != "project" {
		t.Errorf("Scope: got %q want project", got.Scope)
	}
	if got.Purpose != "detect" {
		t.Errorf("Purpose: got %q want detect", got.Purpose)
	}
	if len(got.Paths) != 2 || got.Paths[0] != ".custom-claude" || got.Paths[1] != ".claude-alt" {
		t.Errorf("Paths: got %v want [.custom-claude .claude-alt]", got.Paths)
	}
}

func TestProviderOverrideRepo_Upsert_ReplacesPaths(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	provID := providerIDByKey(t, r, ctx, "claude")

	_ = r.Upsert(ctx, domain.ProviderPathOverride{
		ProviderDefinitionID: provID, Scope: "project", Purpose: "detect",
		Paths: []string{".first"},
	})
	_ = r.Upsert(ctx, domain.ProviderPathOverride{
		ProviderDefinitionID: provID, Scope: "project", Purpose: "detect",
		Paths: []string{".second"},
	})

	all, _ := r.ListAll(ctx)
	if len(all) != 1 {
		t.Fatalf("expected 1 override after upsert, got %d", len(all))
	}
	if len(all[0].Paths) != 1 || all[0].Paths[0] != ".second" {
		t.Errorf("expected .second after upsert, got %v", all[0].Paths)
	}
}

func TestProviderOverrideRepo_Delete_ExistingRow(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	provID := providerIDByKey(t, r, ctx, "claude")

	_ = r.Upsert(ctx, domain.ProviderPathOverride{
		ProviderDefinitionID: provID, Scope: "project", Purpose: "detect",
		Paths: []string{".custom"},
	})

	deleted, err := r.Delete(ctx, provID, "project", "detect")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Error("Delete: expected true (row existed), got false")
	}

	all, _ := r.ListAll(ctx)
	if len(all) != 0 {
		t.Errorf("after delete: expected 0 overrides, got %d", len(all))
	}
}

func TestProviderOverrideRepo_Delete_NonExistent(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	provID := providerIDByKey(t, r, ctx, "claude")

	deleted, err := r.Delete(ctx, provID, "project", "detect")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if deleted {
		t.Error("Delete: expected false (no row), got true")
	}
}

func TestProviderOverrideRepo_GetProviderIDByKey_KnownKey(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	id, err := r.GetProviderIDByKey(ctx, "claude")
	if err != nil {
		t.Fatalf("GetProviderIDByKey: %v", err)
	}
	if id == 0 {
		t.Error("expected non-zero ID for claude")
	}
}

func TestProviderOverrideRepo_GetProviderIDByKey_UnknownKey(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	id, err := r.GetProviderIDByKey(ctx, "no_such_provider")
	if err != nil {
		t.Fatalf("GetProviderIDByKey returned error for unknown: %v", err)
	}
	if id != 0 {
		t.Errorf("expected 0 for unknown key, got %d", id)
	}
}

func TestProviderOverrideRepo_ListAll_EmptyWhenNoOverrides(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	all, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 overrides in fresh DB, got %d", len(all))
	}
}

func TestProviderOverrideRepo_Upsert_InvalidScope(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	provID := providerIDByKey(t, r, ctx, "claude")

	err := r.Upsert(ctx, domain.ProviderPathOverride{
		ProviderDefinitionID: provID,
		Scope:                "invalid_scope",
		Purpose:              "detect",
		Paths:                []string{".custom"},
	})
	if err == nil {
		t.Error("expected CHECK constraint error for invalid scope, got nil")
	}
}

func TestProviderOverrideRepo_Upsert_InvalidPurpose(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderOverrideRepo(db)
	ctx := context.Background()

	provID := providerIDByKey(t, r, ctx, "claude")

	err := r.Upsert(ctx, domain.ProviderPathOverride{
		ProviderDefinitionID: provID,
		Scope:                "project",
		Purpose:              "not_a_purpose",
		Paths:                []string{".custom"},
	})
	if err == nil {
		t.Error("expected CHECK constraint error for invalid purpose, got nil")
	}
}
