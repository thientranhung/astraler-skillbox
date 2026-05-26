-- 000009_provider_registry_seed.up.sql
-- PR-1 Provider Registry: add scope + verification_status to path candidates,
-- seed icon_key for all built-in providers, add global-scope candidates,
-- fix generic_agents has_global_level, and add OpenCode as unsupported.

-- Add scope and verification_status columns to provider_path_candidates.
-- DEFAULT 'project' is safe: all existing rows are project-relative paths.
ALTER TABLE provider_path_candidates ADD COLUMN scope TEXT NOT NULL DEFAULT 'project';
ALTER TABLE provider_path_candidates ADD COLUMN verification_status TEXT NOT NULL DEFAULT 'assumed';

-- Seed icon_key for built-in providers.
UPDATE provider_definitions
   SET icon_key = 'generic_agents', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents' AND icon_key IS NULL;

UPDATE provider_definitions
   SET icon_key = 'claude', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'claude' AND icon_key IS NULL;

UPDATE provider_definitions
   SET icon_key = 'codex', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'codex' AND icon_key IS NULL;

UPDATE provider_definitions
   SET icon_key = 'gemini', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'gemini' AND icon_key IS NULL;

UPDATE provider_definitions
   SET icon_key = 'antigravity', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'antigravity_cli' AND icon_key IS NULL;

-- Fix generic_agents: it does support global-level detection (~/.agents).
UPDATE provider_definitions
   SET has_global_level = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents' AND has_global_level = 0;

-- Add global-scope path candidates for providers with has_global_level = 1.
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.agents', 'global', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'generic_agents';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.agents/skills', 'global', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'generic_agents';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.claude', 'global', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'claude';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.claude/skills', 'global', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'claude';

-- Add OpenCode as unsupported/experimental built-in.
INSERT OR IGNORE INTO provider_definitions (key, display_name, provider_type, icon_key, status, can_create_structure, has_global_level)
VALUES ('opencode', 'OpenCode', 'opencode', 'opencode', 'unsupported', 0, 0);

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.opencode', 'project', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.opencode/rules', 'project', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'opencode';

UPDATE app_settings
   SET database_version = 9, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
