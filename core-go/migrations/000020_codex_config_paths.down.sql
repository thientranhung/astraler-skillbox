-- 000020_codex_config_paths.down.sql

DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'codex')
   AND relative_path IN ('.codex/config.toml', '~/.codex/config.toml')
   AND purpose = 'config';

UPDATE app_settings
   SET database_version = 19, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
