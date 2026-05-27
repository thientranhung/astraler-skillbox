package repositories

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/providers"
)

func TestMigration000006_ProjectProviderSeeds(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		key       string
		status    string
		detectRel string
		skillsRel string
	}{
		{providers.CodexKey, "unsupported", providers.CodexDetectPath, providers.CodexSkillsPath},
		// gemini removed by migration 017
		{providers.AntigravityCLIKey, "unsupported", providers.AntigravityDetectPath, providers.AntigravitySkillsPath},
	}

	for _, c := range cases {
		var status string
		if err := db.QueryRow(`SELECT status FROM provider_definitions WHERE key = ?`, c.key).Scan(&status); err != nil {
			t.Fatalf("%s provider definition: %v", c.key, err)
		}
		if status != c.status {
			t.Errorf("%s status: got %q want %q", c.key, status, c.status)
		}

		for _, pathCase := range []struct {
			purpose string
			want    string
		}{
			{"detect", c.detectRel},
			{"skills", c.skillsRel},
		} {
			var got string
			err := db.QueryRow(`
				SELECT ppc.relative_path
				FROM provider_path_candidates ppc
				JOIN provider_definitions pd ON pd.id = ppc.provider_definition_id
				WHERE pd.key = ? AND ppc.purpose = ?`, c.key, pathCase.purpose).Scan(&got)
			if err != nil {
				t.Fatalf("%s %s candidate: %v", c.key, pathCase.purpose, err)
			}
			if got != pathCase.want {
				t.Errorf("%s %s path: got %q want %q", c.key, pathCase.purpose, got, pathCase.want)
			}
		}
	}
}

func TestMigration000006_DownHandlesExistingProjectProviders(t *testing.T) {
	db := NewTestDB(t)

	var codexDefID int64
	if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key = ?`, providers.CodexKey).Scan(&codexDefID); err != nil {
		t.Fatalf("codex provider definition: %v", err)
	}

	res, err := db.Exec(`INSERT INTO projects (name, path, status) VALUES ('demo', '/tmp/demo', 'active')`)
	if err != nil {
		t.Fatalf("insert project: %v", err)
	}
	projectID, _ := res.LastInsertId()

	res, err = db.Exec(`
		INSERT INTO project_providers
			(project_id, provider_definition_id, detected_path, skills_path, detection_status)
		VALUES (?, ?, '/tmp/demo/.codex', '/tmp/demo/.codex/skills', 'detected')`,
		projectID, codexDefID,
	)
	if err != nil {
		t.Fatalf("insert project_provider: %v", err)
	}
	ppID, _ := res.LastInsertId()

	if _, err := db.Exec(`
		INSERT INTO installs
			(project_provider_id, skill_name, install_mode, install_status, project_skill_path)
		VALUES (?, 'skill-a', 'direct', 'current', '/tmp/demo/.codex/skills/skill-a')`,
		ppID,
	); err != nil {
		t.Fatalf("insert install: %v", err)
	}
	var installID int64
	if err := db.QueryRow(`SELECT id FROM installs WHERE project_provider_id = ?`, ppID).Scan(&installID); err != nil {
		t.Fatalf("select install id: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO warnings (scope_type, scope_id, severity, code, message)
		VALUES
			('project_provider', ?, 'warning', 'invalid_structure', 'provider warning'),
			('install', ?, 'warning', 'broken_symlink', 'install warning')`,
		ppID, installID,
	); err != nil {
		t.Fatalf("insert warnings: %v", err)
	}

	downSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000006_add_project_provider_detection.down.sql"))
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if _, err := db.Exec(string(downSQL)); err != nil {
		t.Fatalf("down migration: %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM provider_definitions WHERE key = ?`, providers.CodexKey).Scan(&count); err != nil {
		t.Fatalf("count provider definition: %v", err)
	}
	if count != 0 {
		t.Errorf("codex provider definition count after down: got %d want 0", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM project_providers WHERE id = ?`, ppID).Scan(&count); err != nil {
		t.Fatalf("count project_provider: %v", err)
	}
	if count != 0 {
		t.Errorf("project_provider count after down: got %d want 0", count)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM installs WHERE project_provider_id = ?`, ppID).Scan(&count); err != nil {
		t.Fatalf("count install: %v", err)
	}
	if count != 0 {
		t.Errorf("install count after down: got %d want 0", count)
	}
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM warnings
		WHERE (scope_type = 'project_provider' AND scope_id = ?)
		   OR (scope_type = 'install' AND scope_id = ?)`,
		ppID, installID,
	).Scan(&count); err != nil {
		t.Fatalf("count warnings: %v", err)
	}
	if count != 0 {
		t.Errorf("warning count after down: got %d want 0", count)
	}
}
