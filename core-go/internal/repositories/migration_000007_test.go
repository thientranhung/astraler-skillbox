package repositories

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestMigration000007_RemovesIgnoredSkillEntries(t *testing.T) {
	db := NewTestDB(t)

	var hostID int64
	if err := db.QueryRow(`SELECT id FROM skill_host_folders LIMIT 1`).Scan(&hostID); err != nil {
		res, err := db.Exec(`
			INSERT INTO skill_host_folders (name, path, skills_path, status)
			VALUES ('host', '/tmp/host', '/tmp/host/skills', 'active')`)
		if err != nil {
			t.Fatalf("insert host folder: %v", err)
		}
		hostID, _ = res.LastInsertId()
	}

	res, err := db.Exec(`
		INSERT INTO skills (skill_host_folder_id, name, relative_path, absolute_path, status)
		VALUES (?, '.DS_Store', '.DS_Store', '/tmp/host/skills/.DS_Store', 'error')`,
		hostID,
	)
	if err != nil {
		t.Fatalf("insert ignored skill: %v", err)
	}
	skillID, _ := res.LastInsertId()

	var genericDefID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key = 'generic_agents'`).Scan(&genericDefID); err != nil {
		t.Fatalf("select generic provider: %v", err)
	}

	res, err = db.Exec(`
		INSERT OR IGNORE INTO global_provider_locations
			(provider_definition_id, name, path, skills_path, status)
		VALUES (?, 'agents', '/Users/me/.agents', '/Users/me/.agents/skills', 'active')`,
		genericDefID,
	)
	if err != nil {
		t.Fatalf("insert global location: %v", err)
	}
	globalLocID, _ := res.LastInsertId()
	if globalLocID == 0 {
		if err := db.QueryRow(`SELECT id FROM global_provider_locations WHERE provider_definition_id = ?`, genericDefID).Scan(&globalLocID); err != nil {
			t.Fatalf("select global location: %v", err)
		}
	}

	res, err = db.Exec(`
		INSERT INTO global_installs
			(global_provider_location_id, skill_name, install_mode, install_status, global_skill_path)
		VALUES (?, '.DS_Store', 'direct', 'error', '/Users/me/.agents/skills/.DS_Store')`,
		globalLocID,
	)
	if err != nil {
		t.Fatalf("insert ignored global install: %v", err)
	}
	globalInstallID, _ := res.LastInsertId()

	res, err = db.Exec(`
		INSERT INTO global_installs
			(global_provider_location_id, skill_id, skill_name, install_mode, install_status, global_skill_path)
		VALUES (?, ?, 'renamed-stale-entry', 'direct', 'error', '/Users/me/.agents/skills/renamed-stale-entry')`,
		globalLocID, skillID,
	)
	if err != nil {
		t.Fatalf("insert skill-linked ignored global install: %v", err)
	}
	linkedGlobalInstallID, _ := res.LastInsertId()

	res, err = db.Exec(`INSERT INTO projects (name, path, status) VALUES ('demo', '/tmp/demo', 'active')`)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	projectID, _ := res.LastInsertId()

	res, err = db.Exec(`
		INSERT INTO project_providers
			(project_id, provider_definition_id, detected_path, skills_path, detection_status)
		VALUES (?, ?, '/tmp/demo/.agents', '/tmp/demo/.agents/skills', 'detected')`,
		projectID, genericDefID,
	)
	if err != nil {
		t.Fatalf("insert project provider: %v", err)
	}
	projectProviderID, _ := res.LastInsertId()

	res, err = db.Exec(`
		INSERT INTO installs
			(project_provider_id, skill_name, install_mode, install_status, project_skill_path)
		VALUES (?, '.DS_Store', 'direct', 'error', '/tmp/demo/.agents/skills/.DS_Store')`,
		projectProviderID,
	)
	if err != nil {
		t.Fatalf("insert ignored project install: %v", err)
	}
	installID, _ := res.LastInsertId()

	res, err = db.Exec(`
		INSERT INTO installs
			(project_provider_id, skill_id, skill_name, install_mode, install_status, project_skill_path)
		VALUES (?, ?, 'renamed-stale-entry', 'direct', 'error', '/tmp/demo/.agents/skills/renamed-stale-entry')`,
		projectProviderID, skillID,
	)
	if err != nil {
		t.Fatalf("insert skill-linked ignored project install: %v", err)
	}
	linkedInstallID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO warnings (scope_type, scope_id, severity, code, message)
		VALUES
			('skill', ?, 'warning', 'metadata_entry', 'ignored skill warning'),
			('global_install', ?, 'warning', 'metadata_entry', 'ignored global warning'),
			('global_install', ?, 'warning', 'metadata_entry', 'ignored linked global warning'),
			('install', ?, 'warning', 'metadata_entry', 'ignored install warning'),
			('install', ?, 'warning', 'metadata_entry', 'ignored linked install warning')`,
		skillID, globalInstallID, linkedGlobalInstallID, installID, linkedInstallID,
	); err != nil {
		t.Fatalf("insert ignored warnings: %v", err)
	}

	upSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000007_remove_ignored_skill_entries.up.sql"))
	if err != nil {
		t.Fatalf("read cleanup migration: %v", err)
	}
	if _, err := db.Exec(string(upSQL)); err != nil {
		t.Fatalf("re-run cleanup migration: %v", err)
	}

	assertCount(t, db, `SELECT COUNT(*) FROM skills WHERE name = '.DS_Store'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM global_installs WHERE skill_name = '.DS_Store'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM installs WHERE skill_name = '.DS_Store'`, 0)
	assertCount(t, db, `SELECT COUNT(*) FROM global_installs WHERE skill_id = ?`, 0, skillID)
	assertCount(t, db, `SELECT COUNT(*) FROM installs WHERE skill_id = ?`, 0, skillID)
	assertCount(t, db, `
		SELECT COUNT(*) FROM warnings
		WHERE (scope_type = 'skill' AND scope_id = ?)
		   OR (scope_type = 'global_install' AND scope_id = ?)
		   OR (scope_type = 'global_install' AND scope_id = ?)
		   OR (scope_type = 'install' AND scope_id = ?)
		   OR (scope_type = 'install' AND scope_id = ?)`, 0, skillID, globalInstallID, linkedGlobalInstallID, installID, linkedInstallID)

	var dbVersion int
	if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id = 1`).Scan(&dbVersion); err != nil {
		t.Fatalf("select database version: %v", err)
	}
	if dbVersion != 7 {
		t.Errorf("database_version: got %d want 7", dbVersion)
	}
}

func assertCount(t *testing.T, db *sql.DB, query string, want int, args ...any) {
	t.Helper()

	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if got != want {
		t.Fatalf("count mismatch for %q: got %d want %d", query, got, want)
	}
}
