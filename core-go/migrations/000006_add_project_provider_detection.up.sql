INSERT OR IGNORE INTO provider_definitions (key, display_name, provider_type, status, can_create_structure, has_global_level)
VALUES
    ('codex', 'Codex (.codex)', 'codex', 'unsupported', 0, 0),
    ('gemini', 'Gemini (.gemini)', 'gemini', 'unsupported', 0, 0),
    ('antigravity_cli', 'Antigravity CLI (.antigravity-cli)', 'antigravity_cli', 'unsupported', 0, 0);

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.codex', 'detect', 10 FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.codex/skills', 'skills', 10 FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.gemini', 'detect', 10 FROM provider_definitions WHERE key = 'gemini';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.gemini/skills', 'skills', 10 FROM provider_definitions WHERE key = 'gemini';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.antigravity-cli', 'detect', 10 FROM provider_definitions WHERE key = 'antigravity_cli';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.antigravity-cli/skills', 'skills', 10 FROM provider_definitions WHERE key = 'antigravity_cli';

UPDATE app_settings
   SET database_version = 6, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
