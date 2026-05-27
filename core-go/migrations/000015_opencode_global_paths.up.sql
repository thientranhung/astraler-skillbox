-- 000015_opencode_global_paths.up.sql
-- Enable global level for OpenCode and seed global detect/skills/config paths.

UPDATE provider_definitions
   SET has_global_level = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'opencode' AND has_global_level = 0;

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.config/opencode', 'global', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.config/opencode/skills', 'global', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.config/opencode/config.json', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

UPDATE app_settings
   SET database_version = 15, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
