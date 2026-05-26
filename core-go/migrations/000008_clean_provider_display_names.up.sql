UPDATE provider_definitions
   SET display_name = 'Shared Agent Skills', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents'
   AND display_name IN ('Generic Agents', 'Shared Agent Skills (.agents)');

UPDATE provider_definitions
   SET display_name = 'Claude', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'claude'
   AND display_name = 'Claude (.claude)';

UPDATE provider_definitions
   SET display_name = 'Codex', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'codex'
   AND display_name = 'Codex (.codex)';

UPDATE provider_definitions
   SET display_name = 'Gemini', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'gemini'
   AND display_name = 'Gemini (.gemini)';

UPDATE provider_definitions
   SET display_name = 'Antigravity CLI', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'antigravity_cli'
   AND display_name = 'Antigravity CLI (.antigravity-cli)';

UPDATE app_settings
   SET database_version = 8, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
