CREATE TABLE IF NOT EXISTS global_provider_locations (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_definition_id  INTEGER NOT NULL REFERENCES provider_definitions(id),
    name                    TEXT,
    path                    TEXT,
    skills_path             TEXT,
    status                  TEXT NOT NULL DEFAULT 'not_configured',
    last_scanned_at         TEXT,
    created_at              TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at              TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_global_loc_provider ON global_provider_locations(provider_definition_id);

CREATE TABLE IF NOT EXISTS global_installs (
    id                            INTEGER PRIMARY KEY AUTOINCREMENT,
    global_provider_location_id   INTEGER NOT NULL REFERENCES global_provider_locations(id),
    skill_id                      INTEGER REFERENCES skills(id),
    skill_name                    TEXT    NOT NULL,
    install_mode                  TEXT    NOT NULL,
    install_status                TEXT    NOT NULL DEFAULT 'current',
    global_skill_path             TEXT    NOT NULL,
    source_skill_path             TEXT,
    symlink_target_path           TEXT,
    installed_from_host_folder_id INTEGER REFERENCES skill_host_folders(id),
    installed_version             TEXT,
    installed_commit              TEXT,
    installed_checksum            TEXT,
    last_synced_at                TEXT,
    last_scanned_at               TEXT,
    created_at                    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at                    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_global_installs_loc_path
    ON global_installs(global_provider_location_id, global_skill_path);
CREATE INDEX IF NOT EXISTS idx_global_installs_location
    ON global_installs(global_provider_location_id);

UPDATE provider_definitions
   SET has_global_level = 1, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE key = 'generic_agents';

UPDATE app_settings
   SET database_version = 5, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
