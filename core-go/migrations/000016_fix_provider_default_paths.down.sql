-- 000016_fix_provider_default_paths.down.sql
-- Revert path candidate fixes.

-- Revert OpenCode project skills back to .opencode/rules
UPDATE provider_path_candidates
   SET relative_path = '.opencode/rules'
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'opencode')
   AND relative_path = '.opencode/skills'
   AND scope = 'project'
   AND purpose = 'skills';

-- Remove OpenCode compat paths added in this migration
DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'opencode')
   AND relative_path IN ('.claude/skills', '.agents/skills', '~/.claude/skills', '~/.agents/skills');

-- Remove Codex paths added in this migration and restore old ones
DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'codex')
   AND relative_path IN ('.agents', '.agents/skills', '~/.agents', '~/.agents/skills');

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.codex', 'project', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.codex/skills', 'project', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.codex/config.toml', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.codex/config.toml', 'project', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

-- Revert Gemini: remove global paths + alias, reset has_global_level
DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'gemini')
   AND relative_path IN ('~/.gemini', '~/.gemini/skills', '.agents/skills', '~/.agents/skills');

UPDATE provider_definitions
   SET has_global_level = 0, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'gemini';

UPDATE app_settings
   SET database_version = 15, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
