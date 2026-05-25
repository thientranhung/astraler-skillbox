DROP TABLE IF EXISTS installs;
DROP TABLE IF EXISTS project_providers;
DROP TABLE IF EXISTS provider_path_candidates;
DROP TABLE IF EXISTS provider_definitions;
DROP TABLE IF EXISTS projects;

-- Restore schema version.
UPDATE app_settings SET database_version = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1;
