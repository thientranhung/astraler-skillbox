-- 000024_generic_agents_create_structure.down.sql

UPDATE provider_definitions
   SET can_create_structure = 0,
       updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents';

UPDATE app_settings
   SET database_version = 23,
       updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
