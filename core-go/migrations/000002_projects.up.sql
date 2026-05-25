-- 000002_projects.up.sql
-- Slice 2A tables: projects, provider_definitions, provider_path_candidates, project_providers, installs
-- scan_results intentionally omitted; scan summary stored in operations.metadata_json.

CREATE TABLE IF NOT EXISTS projects (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT    NOT NULL,
    path            TEXT    NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'active',
    last_scanned_at TEXT,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_projects_path ON projects(path);

CREATE TABLE IF NOT EXISTS provider_definitions (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    key                 TEXT    NOT NULL,
    display_name        TEXT    NOT NULL,
    provider_type       TEXT    NOT NULL,
    icon_key            TEXT,
    status              TEXT    NOT NULL DEFAULT 'supported',
    can_create_structure INTEGER NOT NULL DEFAULT 0,
    has_global_level    INTEGER NOT NULL DEFAULT 0,
    created_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_provider_definitions_key ON provider_definitions(key);

CREATE TABLE IF NOT EXISTS provider_path_candidates (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_definition_id  INTEGER NOT NULL REFERENCES provider_definitions(id),
    relative_path           TEXT    NOT NULL,
    purpose                 TEXT    NOT NULL,
    priority                INTEGER NOT NULL DEFAULT 10,
    description             TEXT,
    created_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_provider_path_candidates_provider ON provider_path_candidates(provider_definition_id);

CREATE TABLE IF NOT EXISTS project_providers (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id              INTEGER NOT NULL REFERENCES projects(id),
    provider_definition_id  INTEGER NOT NULL REFERENCES provider_definitions(id),
    detected_path           TEXT,
    skills_path             TEXT,
    detection_status        TEXT    NOT NULL DEFAULT 'missing',
    last_scanned_at         TEXT,
    created_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_project_providers_project_provider ON project_providers(project_id, provider_definition_id);
CREATE INDEX IF NOT EXISTS idx_project_providers_project ON project_providers(project_id);

CREATE TABLE IF NOT EXISTS installs (
    id                          INTEGER PRIMARY KEY AUTOINCREMENT,
    project_provider_id         INTEGER NOT NULL REFERENCES project_providers(id),
    skill_id                    INTEGER REFERENCES skills(id),
    skill_name                  TEXT    NOT NULL,
    install_mode                TEXT    NOT NULL,
    install_status              TEXT    NOT NULL DEFAULT 'current',
    project_skill_path          TEXT    NOT NULL,
    source_skill_path           TEXT,
    symlink_target_path         TEXT,
    installed_from_host_folder_id INTEGER REFERENCES skill_host_folders(id),
    installed_version           TEXT,
    installed_commit            TEXT,
    installed_checksum          TEXT,
    last_synced_at              TEXT,
    last_scanned_at             TEXT,
    created_at                  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at                  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_installs_provider_path ON installs(project_provider_id, project_skill_path);
CREATE INDEX IF NOT EXISTS idx_installs_project_provider ON installs(project_provider_id);

-- Seed generic_agents provider definition.
INSERT OR IGNORE INTO provider_definitions (key, display_name, provider_type, status, can_create_structure, has_global_level)
VALUES ('generic_agents', 'Generic Agents', 'generic_agents', 'supported', 0, 0);

-- Seed generic_agents path candidates: detect=.agents, skills=.agents/skills.
INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.agents', 'detect', 10 FROM provider_definitions WHERE key = 'generic_agents';

INSERT OR IGNORE INTO provider_path_candidates (provider_definition_id, relative_path, purpose, priority)
SELECT id, '.agents/skills', 'skills', 10 FROM provider_definitions WHERE key = 'generic_agents';
