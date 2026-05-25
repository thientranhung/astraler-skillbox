-- 000004_shared_agent_display_names.up.sql
UPDATE provider_definitions
SET display_name = 'Shared Agent Skills (.agents)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'generic_agents' AND display_name = 'Generic Agents';

UPDATE provider_definitions
SET display_name = 'Claude (.claude)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'claude' AND display_name = 'Claude';

UPDATE app_settings SET database_version = 4, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1;
