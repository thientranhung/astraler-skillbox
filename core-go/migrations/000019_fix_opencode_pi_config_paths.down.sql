-- 000019_fix_opencode_pi_config_paths.down.sql

DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'opencode')
   AND relative_path = '.opencode/config.json'
   AND scope = 'project'
   AND purpose = 'config';

DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'pi')
   AND relative_path = '.pi/settings.json'
   AND scope = 'project'
   AND purpose = 'config';

DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'pi')
   AND relative_path = '~/.pi/agent/settings.json'
   AND scope = 'global'
   AND purpose = 'config';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.config/opencode/config.json', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

UPDATE app_settings
   SET database_version = 18, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
