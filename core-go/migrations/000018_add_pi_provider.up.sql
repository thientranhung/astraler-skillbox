-- 000018_add_pi_provider.up.sql
-- Add pi.dev provider with OpenCode-compatible paths.

INSERT OR IGNORE INTO provider_definitions (key, display_name, provider_type, icon_key, status, can_create_structure, has_global_level)
VALUES ('pi', 'Pi', 'pi', 'pi', 'unsupported', 0, 1);

-- Project detect
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.opencode', 'project', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

-- Project skills (native)
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.opencode/skills', 'project', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

-- Project skills (compat)
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.claude/skills', 'project', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '.agents/skills', 'project', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

-- Global detect
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.config/opencode', 'global', 'detect', 10, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

-- Global skills (native)
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.config/opencode/skills', 'global', 'skills', 10, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

-- Global skills (compat)
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.claude/skills', 'global', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.agents/skills', 'global', 'skills', 20, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

-- Global config
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, scope, purpose, priority, verification_status)
SELECT id, '~/.config/opencode/config.json', 'global', 'config', 10, 'assumed'
  FROM provider_definitions WHERE key = 'pi';

-- Bump database_version
UPDATE app_settings
   SET database_version = 18, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
