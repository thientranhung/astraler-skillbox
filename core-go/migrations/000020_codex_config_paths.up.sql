-- 000020_codex_config_paths.up.sql
-- Restore codex config paths (removed incorrectly by migration 016).

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.codex/config.toml', 'project', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.codex/config.toml', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

UPDATE app_settings
   SET database_version = 20, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
