-- 000009_provider_registry_seed.down.sql

-- Remove opencode path candidates before deleting the definition (no CASCADE on FK).
DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'opencode');

DELETE FROM provider_definitions WHERE key = 'opencode';

-- Remove global-scope candidates added by this migration.
DELETE FROM provider_path_candidates
 WHERE scope = 'global'
   AND provider_definition_id IN (
         SELECT id FROM provider_definitions WHERE key IN ('generic_agents', 'claude')
       );

UPDATE provider_definitions
   SET has_global_level = 0, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents';

UPDATE provider_definitions
   SET icon_key = NULL, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key IN ('generic_agents', 'claude', 'codex', 'gemini', 'antigravity_cli');

-- Note: ALTER TABLE ADD COLUMN cannot be reversed in SQLite without table recreation.
-- The scope and verification_status columns remain but are unused after rollback.

UPDATE app_settings
   SET database_version = 8, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
