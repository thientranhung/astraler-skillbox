-- 000004_shared_agent_display_names.down.sql
UPDATE provider_definitions
SET display_name = 'Generic Agents', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'generic_agents' AND display_name = 'Shared Agent Skills (.agents)';

UPDATE provider_definitions
SET display_name = 'Claude', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'claude' AND display_name = 'Claude (.claude)';

UPDATE app_settings SET database_version = 3, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1;
