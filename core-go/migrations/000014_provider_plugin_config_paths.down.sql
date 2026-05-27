-- 000014_provider_plugin_config_paths.down.sql

DELETE FROM provider_path_candidates
 WHERE purpose = 'config'
   AND (
     (relative_path IN ('~/.claude/settings.json', '.claude/settings.json', '.claude/settings.local.json')
      AND provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'claude'))
     OR
     (relative_path IN ('~/.codex/config.toml', '.codex/config.toml')
      AND provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'codex'))
     OR
     (relative_path IN ('~/.gemini/antigravity-cli/settings.json', '.gemini/antigravity-cli/settings.json')
      AND provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'antigravity_cli'))
   );

UPDATE provider_definitions
   SET has_global_level = 0, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key IN ('codex', 'antigravity_cli');

UPDATE app_settings
   SET database_version = 13, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
