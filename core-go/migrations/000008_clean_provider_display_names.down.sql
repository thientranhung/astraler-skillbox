UPDATE provider_definitions
   SET display_name = 'Shared Agent Skills (.agents)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents'
   AND display_name = 'Shared Agent Skills';

UPDATE provider_definitions
   SET display_name = 'Claude (.claude)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'claude'
   AND display_name = 'Claude';

UPDATE provider_definitions
   SET display_name = 'Codex (.codex)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'codex'
   AND display_name = 'Codex';

UPDATE provider_definitions
   SET display_name = 'Gemini (.gemini)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'gemini'
   AND display_name = 'Gemini';

UPDATE provider_definitions
   SET display_name = 'Antigravity CLI (.antigravity-cli)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'antigravity_cli'
   AND display_name = 'Antigravity CLI';

UPDATE app_settings
   SET database_version = 7, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
