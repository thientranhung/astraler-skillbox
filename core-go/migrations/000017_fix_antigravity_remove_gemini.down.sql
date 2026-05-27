-- 000017_fix_antigravity_remove_gemini.down.sql
-- Revert: restore gemini provider and revert antigravity_cli paths.

------------------------------------------------------------------------
-- 1. Restore gemini provider definition
------------------------------------------------------------------------
INSERT OR IGNORE INTO provider_definitions (key, display_name, status, icon_key, has_global_level)
VALUES ('gemini', 'Gemini', 'unsupported', 'gemini', 1);

-- Restore gemini path candidates (as set by migration 016)
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.gemini', 'project', 'detect', 10, 'assumed' FROM provider_definitions WHERE key = 'gemini';
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.gemini/skills', 'project', 'skills', 10, 'assumed' FROM provider_definitions WHERE key = 'gemini';
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.agents/skills', 'project', 'skills', 20, 'assumed' FROM provider_definitions WHERE key = 'gemini';
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.gemini', 'global', 'detect', 10, 'assumed' FROM provider_definitions WHERE key = 'gemini';
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.gemini/skills', 'global', 'skills', 10, 'assumed' FROM provider_definitions WHERE key = 'gemini';
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.agents/skills', 'global', 'skills', 20, 'assumed' FROM provider_definitions WHERE key = 'gemini';

------------------------------------------------------------------------
-- 2. Revert antigravity_cli paths
------------------------------------------------------------------------
-- Project skills: .agents/skills -> .antigravity-cli/skills
UPDATE provider_path_candidates
   SET relative_path = '.antigravity-cli/skills'
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'antigravity_cli')
   AND relative_path = '.agents/skills'
   AND scope = 'project'
   AND purpose = 'skills';

-- Remove global skills added in this migration
DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'antigravity_cli')
   AND relative_path = '~/.gemini/antigravity-cli/skills/'
   AND scope = 'global'
   AND purpose = 'skills';

UPDATE app_settings
   SET database_version = 16, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
