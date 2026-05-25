-- 000003_claude_provider.down.sql
-- Remove only the claude provider rows seeded by this migration.

DELETE FROM provider_path_candidates
WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'claude');

DELETE FROM provider_definitions WHERE key = 'claude';

UPDATE app_settings SET database_version = 2, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1;
