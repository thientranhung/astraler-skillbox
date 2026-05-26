-- 000010_provider_path_overrides.up.sql
-- PR-2A: user overrides for built-in provider path candidates.
-- One override row per (provider_definition_id, scope, purpose).
-- paths_json stores a JSON array of path strings replacing builtin defaults.

CREATE TABLE IF NOT EXISTS provider_path_overrides (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_definition_id INTEGER NOT NULL REFERENCES provider_definitions(id),
    scope                  TEXT NOT NULL CHECK (scope IN ('project', 'global')),
    purpose                TEXT NOT NULL CHECK (purpose IN ('detect', 'skills', 'config', 'commands')),
    paths_json             TEXT NOT NULL DEFAULT '[]'
                               CHECK (json_valid(paths_json) AND json_type(paths_json) = 'array'),
    created_at             TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at             TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(provider_definition_id, scope, purpose)
);

UPDATE app_settings
   SET database_version = 10, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
