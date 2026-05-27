-- 000016_fix_provider_default_paths.up.sql
-- Fix default path candidates to match official provider documentation (May 2026).
--
-- Fixes:
--   opencode : project skills ".opencode/rules" -> ".opencode/skills"
--   codex    : remove wrong ".codex" paths, add correct ".agents/skills" native paths
--   gemini   : add global paths, add ".agents/skills" alias, set has_global_level=1

------------------------------------------------------------------------
-- 1. OpenCode: fix project skills path
------------------------------------------------------------------------
UPDATE provider_path_candidates
   SET relative_path = '.opencode/skills'
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'opencode')
   AND relative_path = '.opencode/rules'
   AND scope = 'project'
   AND purpose = 'skills';

------------------------------------------------------------------------
-- 2. Codex: remove wrong ".codex" paths
------------------------------------------------------------------------
DELETE FROM provider_path_candidates
 WHERE provider_definition_id = (SELECT id FROM provider_definitions WHERE key = 'codex')
   AND relative_path IN ('.codex', '.codex/skills', '.codex/config.toml', '~/.codex/config.toml');

-- Add correct Codex paths per official docs: native path is .agents/skills/
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.agents', 'project', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.agents/skills', 'project', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.agents', 'global', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.agents/skills', 'global', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'codex';

------------------------------------------------------------------------
-- 3. Gemini: add global paths + .agents/skills alias + has_global_level
------------------------------------------------------------------------
UPDATE provider_definitions
   SET has_global_level = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'gemini' AND has_global_level = 0;

-- Global detect + skills (native)
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.gemini', 'global', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'gemini';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.gemini/skills', 'global', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'gemini';

-- Project alias: .agents/skills (Gemini reads this as alias)
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.agents/skills', 'project', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'gemini';

-- Global alias: ~/.agents/skills
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.agents/skills', 'global', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'gemini';

------------------------------------------------------------------------
-- 4. OpenCode: add compat paths (reads .claude/skills + .agents/skills)
------------------------------------------------------------------------
-- Project compat
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.claude/skills', 'project', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.agents/skills', 'project', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

-- Global compat
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.claude/skills', 'global', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.agents/skills', 'global', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

------------------------------------------------------------------------
-- Bump database_version
------------------------------------------------------------------------
UPDATE app_settings
   SET database_version = 16, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
