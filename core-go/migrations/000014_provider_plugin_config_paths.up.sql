-- 000014_provider_plugin_config_paths.up.sql
-- Seed provider plugin configuration paths for global and project scopes.

UPDATE provider_definitions
   SET has_global_level = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key IN ('codex', 'antigravity_cli');

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.claude/settings.json', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'claude';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.claude/settings.json', 'project', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'claude';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.claude/settings.local.json', 'project', 'config', 9, 'assumed'
  FROM provider_definitions WHERE key = 'claude';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.codex/config.toml', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.codex/config.toml', 'project', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.gemini/antigravity-cli/settings.json', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'antigravity_cli';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.gemini/antigravity-cli/settings.json', 'project', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'antigravity_cli';

UPDATE app_settings
   SET database_version = 14, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
