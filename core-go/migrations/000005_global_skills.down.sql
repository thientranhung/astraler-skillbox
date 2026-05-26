DROP TABLE IF EXISTS global_installs;
DROP TABLE IF EXISTS global_provider_locations;

UPDATE provider_definitions
   SET has_global_level = 0, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents';

UPDATE app_settings
   SET database_version = 4, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
