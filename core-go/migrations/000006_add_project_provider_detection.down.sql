DELETE FROM warnings
WHERE scope_type = 'install'
  AND scope_id IN (
    SELECT i.id
    FROM installs i
    JOIN project_providers pp ON pp.id = i.project_provider_id
    JOIN provider_definitions pd ON pd.id = pp.provider_definition_id
    WHERE pd.key IN ('codex', 'gemini', 'antigravity_cli')
);

DELETE FROM warnings
WHERE scope_type = 'project_provider'
  AND scope_id IN (
    SELECT pp.id
    FROM project_providers pp
    JOIN provider_definitions pd ON pd.id = pp.provider_definition_id
    WHERE pd.key IN ('codex', 'gemini', 'antigravity_cli')
);

DELETE FROM installs
WHERE project_provider_id IN (
    SELECT pp.id
    FROM project_providers pp
    JOIN provider_definitions pd ON pd.id = pp.provider_definition_id
    WHERE pd.key IN ('codex', 'gemini', 'antigravity_cli')
);

DELETE FROM project_providers
WHERE provider_definition_id IN (
    SELECT id FROM provider_definitions
    WHERE key IN ('codex', 'gemini', 'antigravity_cli')
);

DELETE FROM provider_path_candidates
WHERE provider_definition_id IN (
    SELECT id FROM provider_definitions
    WHERE key IN ('codex', 'gemini', 'antigravity_cli')
);

DELETE FROM provider_definitions
WHERE key IN ('codex', 'gemini', 'antigravity_cli');

UPDATE app_settings
   SET database_version = 5, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
