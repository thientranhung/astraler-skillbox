package repositories

import (
	"context"
	"testing"
)

func TestResetAllData_EmptyDB(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Should not error on empty DB.
	if err := ResetAllData(ctx, db); err != nil {
		t.Fatalf("ResetAllData on empty DB: %v", err)
	}
}

func TestResetAllData_ClearsUserData(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Insert a project.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO projects (name, path, created_at, updated_at)
		 VALUES ('p1', '/tmp/p1', strftime('%Y-%m-%dT%H:%M:%SZ','now'), strftime('%Y-%m-%dT%H:%M:%SZ','now'))`); err != nil {
		t.Fatalf("insert project: %v", err)
	}

	// Insert a skill_host_folder + skill.
	var hostID int64
	if err := db.QueryRowContext(ctx,
		`INSERT INTO skill_host_folders (name, path, skills_path, created_at, updated_at)
		 VALUES ('h1', '/tmp/h1', '/tmp/h1/.agents/skills', strftime('%Y-%m-%dT%H:%M:%SZ','now'), strftime('%Y-%m-%dT%H:%M:%SZ','now'))
		 RETURNING id`).Scan(&hostID); err != nil {
		t.Fatalf("insert skill_host_folder: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO skills (name, relative_path, absolute_path, skill_host_folder_id)
		 VALUES ('s1', 's1', '/tmp/h1/.agents/skills/s1', ?)`,
		hostID); err != nil {
		t.Fatalf("insert skill: %v", err)
	}

	// Point app_settings at the host.
	if _, err := db.ExecContext(ctx,
		`UPDATE app_settings SET active_skill_host_folder_id = ? WHERE id = 1`, hostID); err != nil {
		t.Fatalf("update app_settings: %v", err)
	}

	if err := ResetAllData(ctx, db); err != nil {
		t.Fatalf("ResetAllData: %v", err)
	}

	tables := []string{"projects", "skill_host_folders", "skills", "skill_sources",
		"operations", "warnings", "provider_plugin_entries",
		"provider_plugin_layer_scans", "provider_plugin_marketplaces",
		"plugin_update_check_cache", "global_provider_locations",
		"global_installs", "installs", "project_providers",
		"provider_user_settings", "provider_path_overrides"}
	for _, tbl := range tables {
		var n int
		if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+tbl).Scan(&n); err != nil {
			t.Errorf("count %s: %v", tbl, err)
			continue
		}
		if n != 0 {
			t.Errorf("table %s: expected 0 rows, got %d", tbl, n)
		}
	}
}

func TestResetAllData_ResetsSettings(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	// Set non-default values.
	if _, err := db.ExecContext(ctx,
		`UPDATE app_settings SET default_install_mode = 'copy' WHERE id = 1`); err != nil {
		t.Fatalf("update app_settings: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE network_settings SET update_check_enabled = 1, cache_ttl_hours = 99 WHERE id = 1`); err != nil {
		t.Fatalf("update network_settings: %v", err)
	}

	if err := ResetAllData(ctx, db); err != nil {
		t.Fatalf("ResetAllData: %v", err)
	}

	var installMode string
	var activeID interface{}
	if err := db.QueryRowContext(ctx,
		`SELECT active_skill_host_folder_id, default_install_mode FROM app_settings WHERE id = 1`).
		Scan(&activeID, &installMode); err != nil {
		t.Fatalf("read app_settings: %v", err)
	}
	if activeID != nil {
		t.Errorf("active_skill_host_folder_id: expected NULL, got %v", activeID)
	}
	if installMode != "symlink" {
		t.Errorf("default_install_mode: expected 'symlink', got %q", installMode)
	}

	var enabled int
	var ttl int
	if err := db.QueryRowContext(ctx,
		`SELECT update_check_enabled, cache_ttl_hours FROM network_settings WHERE id = 1`).
		Scan(&enabled, &ttl); err != nil {
		t.Fatalf("read network_settings: %v", err)
	}
	if enabled != 0 {
		t.Errorf("update_check_enabled: expected 0, got %d", enabled)
	}
	if ttl != 6 {
		t.Errorf("cache_ttl_hours: expected 6, got %d", ttl)
	}
}

func TestResetAllData_SingletonRowsPreserved(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	if err := ResetAllData(ctx, db); err != nil {
		t.Fatalf("ResetAllData: %v", err)
	}

	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_settings`).Scan(&n); err != nil {
		t.Fatalf("count app_settings: %v", err)
	}
	if n != 1 {
		t.Errorf("app_settings: expected 1 row, got %d", n)
	}

	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM network_settings`).Scan(&n); err != nil {
		t.Fatalf("count network_settings: %v", err)
	}
	if n != 1 {
		t.Errorf("network_settings: expected 1 row, got %d", n)
	}
}
