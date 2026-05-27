-- 000017_fix_antigravity_remove_gemini.up.sql
-- 1. Fix antigravity_cli default paths to match actual usage.
-- 2. Remove gemini provider entirely.

------------------------------------------------------------------------
-- 1. Antigravity CLI: fix project skills + add global skills
------------------------------------------------------------------------
-- Project skills: .antigravity-cli/skills -> .agents/skills
UPDATE provider_path_candidates
   SET relative_path = '.agents/skills'
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'antigravity_cli')
   AND relative_path = '.antigravity-cli/skills'
   AND scope = 'project'
   AND purpose = 'skills';

-- Global skills: add ~/.gemini/antigravity-cli/skills/
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.gemini/antigravity-cli/skills/', 'global', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'antigravity_cli';

------------------------------------------------------------------------
-- 2. Remove gemini provider: path candidates first (FK), then definition
------------------------------------------------------------------------
DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'gemini');

DELETE FROM provider_path_overrides
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'gemini');

DELETE FROM provider_user_settings
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'gemini');

DELETE FROM provider_definitions
 WHERE key = 'gemini';

------------------------------------------------------------------------
-- Bump database_version
------------------------------------------------------------------------
UPDATE app_settings
   SET database_version = 17, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
