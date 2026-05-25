-- 000003_claude_provider.up.sql
-- Slice 2D: seed claude provider definition and path candidates.

INSERT OR IGNORE INTO provider_definitions (key, display_name, provider_type, status, can_create_structure, has_global_level)
VALUES ('claude', 'Claude', 'claude', 'experimental', 0, 1);

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.claude', 'detect', 10 FROM provider_definitions WHERE key = 'claude';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.claude/skills', 'skills', 10 FROM provider_definitions WHERE key = 'claude';

UPDATE app_settings SET database_version = 3, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1;
