-- 000015_opencode_global_paths.down.sql
-- Revert OpenCode global paths and has_global_level.

DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'opencode')
   AND scope = 'global';

UPDATE provider_definitions
   SET has_global_level = 0, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'opencode';

UPDATE app_settings
   SET database_version = 14, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
