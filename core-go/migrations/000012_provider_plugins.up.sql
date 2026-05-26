CREATE TABLE IF NOT EXISTS provider_plugin_layer_scans (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_definition_id INTEGER NOT NULL REFERENCES provider_definitions(id) ON DELETE CASCADE,
    project_id             INTEGER REFERENCES projects(id) ON DELETE CASCADE,
    settings_layer         TEXT NOT NULL CHECK(settings_layer IN ('user', 'project', 'local')),
    scan_status            TEXT NOT NULL CHECK(scan_status IN ('ok', 'missing', 'unreadable', 'malformed', 'too_large', 'symlink', 'path_escape')),
    settings_file_path     TEXT NOT NULL,
    last_scanned_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    source_operation_id    INTEGER REFERENCES operations(id) ON DELETE SET NULL,
    -- user layer must have null project_id; project/local layers must have non-null project_id
    CHECK (
        (settings_layer = 'user' AND project_id IS NULL) OR
        (settings_layer IN ('project', 'local') AND project_id IS NOT NULL)
    )
);

-- Partial unique indexes handle NULL project_id correctly (SQLite UNIQUE treats NULLs as distinct).
CREATE UNIQUE INDEX IF NOT EXISTS uq_plugin_layer_scans_user
ON provider_plugin_layer_scans(provider_definition_id, settings_layer)
WHERE project_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_plugin_layer_scans_project
ON provider_plugin_layer_scans(provider_definition_id, project_id, settings_layer)
WHERE project_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS provider_plugin_entries (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    layer_scan_id    INTEGER NOT NULL REFERENCES provider_plugin_layer_scans(id) ON DELETE CASCADE,
    plugin_name      TEXT NOT NULL,
    marketplace_name TEXT NOT NULL,
    declaration      TEXT NOT NULL CHECK(declaration IN ('enabled', 'disabled')),
    UNIQUE(layer_scan_id, plugin_name, marketplace_name)
);

CREATE TABLE IF NOT EXISTS provider_plugin_marketplaces (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    layer_scan_id    INTEGER NOT NULL REFERENCES provider_plugin_layer_scans(id) ON DELETE CASCADE,
    marketplace_name TEXT NOT NULL,
    source_type      TEXT NOT NULL,
    source_summary   TEXT NOT NULL
);

UPDATE app_settings SET database_version = 12 WHERE id = 1;
