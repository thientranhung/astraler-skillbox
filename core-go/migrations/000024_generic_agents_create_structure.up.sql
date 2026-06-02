-- 000024_generic_agents_create_structure.up.sql
-- Shared Agent Skills may scaffold the project-local .agents/skills target.

UPDATE provider_definitions
   SET can_create_structure = 1,
       updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents';

UPDATE app_settings
   SET database_version = 24,
       updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
