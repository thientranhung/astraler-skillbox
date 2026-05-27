-- 000019_fix_opencode_pi_config_paths.up.sql
-- Add missing config paths for opencode and pi, fix pi global config.

------------------------------------------------------------------------
-- 1. OpenCode: add project config
------------------------------------------------------------------------
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.opencode/config.json', 'project', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

------------------------------------------------------------------------
-- 2. Pi: add project config
------------------------------------------------------------------------
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.pi/settings.json', 'project', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

------------------------------------------------------------------------
-- 3. Pi: fix global config (was ~/.config/opencode/config.json, should be ~/.pi/agent/settings.json)
------------------------------------------------------------------------
DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'pi')
   AND relative_path = '~/.config/opencode/config.json'
   AND scope = 'global'
   AND purpose = 'config';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.pi/agent/settings.json', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

------------------------------------------------------------------------
-- Bump database_version
------------------------------------------------------------------------
UPDATE app_settings
   SET database_version = 19, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
