package repositories

import (
	"context"
	"testing"
)

func providerIDForSettingsTest(t *testing.T, db interface{ QueryRow(string, ...any) interface{ Scan(...any) error } }, key string) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key=?`, key).Scan(&id); err != nil {
		t.Fatalf("provider %q not found: %v", key, err)
	}
	return id
}

func TestProviderUserSettingsRepo_ListAll_EmptyByDefault(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderUserSettingsRepo(db)
	ctx := context.Background()

	all, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 0 {
		t.Errorf("expected 0 settings in fresh DB, got %d", len(all))
	}
}

func TestProviderUserSettingsRepo_Upsert_True(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderUserSettingsRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	if err := r.Upsert(ctx, provID, true); err != nil {
		t.Fatalf("Upsert(true): %v", err)
	}

	all, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(all))
	}
	if all[0].ProviderDefinitionID != provID {
		t.Errorf("ProviderDefinitionID: got %d want %d", all[0].ProviderDefinitionID, provID)
	}
	if !all[0].Enabled {
		t.Error("Enabled: got false want true")
	}
}

func TestProviderUserSettingsRepo_Upsert_False(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderUserSettingsRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='generic_agents'`).Scan(&provID); err != nil {
		t.Fatalf("generic_agents not found: %v", err)
	}

	if err := r.Upsert(ctx, provID, false); err != nil {
		t.Fatalf("Upsert(false): %v", err)
	}

	all, _ := r.ListAll(ctx)
	if len(all) != 1 {
		t.Fatalf("expected 1 setting, got %d", len(all))
	}
	if all[0].Enabled {
		t.Error("Enabled: got true want false")
	}
}

func TestProviderUserSettingsRepo_Upsert_OverwritesValue(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderUserSettingsRepo(db)
	ctx := context.Background()

	var provID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&provID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}

	_ = r.Upsert(ctx, provID, true)
	_ = r.Upsert(ctx, provID, false)

	all, _ := r.ListAll(ctx)
	if len(all) != 1 {
		t.Fatalf("expected 1 setting after two upserts, got %d", len(all))
	}
	if all[0].Enabled {
		t.Error("expected false after second upsert, got true")
	}
}

func TestProviderUserSettingsRepo_ListAll_MultipleProviders(t *testing.T) {
	db := NewTestDB(t)
	r := NewProviderUserSettingsRepo(db)
	ctx := context.Background()

	var claudeID, genericID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&claudeID); err != nil {
		t.Fatalf("claude not found: %v", err)
	}
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='generic_agents'`).Scan(&genericID); err != nil {
		t.Fatalf("generic_agents not found: %v", err)
	}

	_ = r.Upsert(ctx, claudeID, true)
	_ = r.Upsert(ctx, genericID, false)

	all, err := r.ListAll(ctx)
	if err != nil {
		t.Fatalf("ListAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 settings, got %d", len(all))
	}
}
