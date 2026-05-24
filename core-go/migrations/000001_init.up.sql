-- 000001_init.up.sql
-- Slice 1 tables: app_settings, skill_host_folders, skills, skill_sources, operations, warnings
-- skill_sources included for skills.source_id FK integrity; slice 1 never writes to it.

CREATE TABLE IF NOT EXISTS app_settings (
    id                          INTEGER PRIMARY KEY,
    active_skill_host_folder_id INTEGER REFERENCES skill_host_folders(id),
    default_install_mode        TEXT    NOT NULL DEFAULT 'symlink',
    database_version            INTEGER NOT NULL DEFAULT 1,
    created_at                  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at                  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS skill_host_folders (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    name            TEXT,
    path            TEXT    NOT NULL,
    skills_path     TEXT    NOT NULL,
    status          TEXT    NOT NULL DEFAULT 'active',
    last_scanned_at TEXT,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_skill_host_path ON skill_host_folders(path);

-- skill_sources included for FK integrity; slice 1 has no write path here.
CREATE TABLE IF NOT EXISTS skill_sources (
    id                      INTEGER PRIMARY KEY AUTOINCREMENT,
    source_type             TEXT    NOT NULL,
    url                     TEXT,
    github_owner            TEXT,
    github_repo             TEXT,
    github_path             TEXT,
    github_ref              TEXT,
    vercel_skill_id         TEXT,
    local_source_path       TEXT,
    resolved_version        TEXT,
    resolved_commit         TEXT,
    last_fetched_at         TEXT,
    last_successful_fetch_at TEXT,
    last_fetch_status       TEXT    NOT NULL DEFAULT 'never_fetched',
    last_fetch_error        TEXT,
    created_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at              TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS skills (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    skill_host_folder_id INTEGER NOT NULL REFERENCES skill_host_folders(id),
    name                TEXT    NOT NULL,
    display_name        TEXT,
    relative_path       TEXT    NOT NULL,
    absolute_path       TEXT    NOT NULL,
    status              TEXT    NOT NULL DEFAULT 'unknown',
    source_id           INTEGER REFERENCES skill_sources(id),
    current_version     TEXT,
    current_commit      TEXT,
    current_checksum    TEXT,
    last_scanned_at     TEXT,
    created_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_skills_host ON skills(skill_host_folder_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_skills_host_relpath ON skills(skill_host_folder_id, relative_path);

CREATE TABLE IF NOT EXISTS operations (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    operation_type  TEXT    NOT NULL,
    target_type     TEXT    NOT NULL,
    target_id       INTEGER,
    status          TEXT    NOT NULL DEFAULT 'queued',
    started_at      TEXT,
    finished_at     TEXT,
    error_message   TEXT,
    metadata_json   TEXT,
    created_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at      TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_operations_target ON operations(target_type, target_id, status);

CREATE TABLE IF NOT EXISTS warnings (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    scope_type          TEXT    NOT NULL,
    scope_id            INTEGER,
    severity            TEXT    NOT NULL DEFAULT 'warning',
    code                TEXT    NOT NULL,
    message             TEXT    NOT NULL,
    action_key          TEXT,
    source_operation_id INTEGER REFERENCES operations(id),
    is_resolved         INTEGER NOT NULL DEFAULT 0,
    created_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at          TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    resolved_at         TEXT
);

CREATE INDEX IF NOT EXISTS idx_warnings_scope ON warnings(scope_type, scope_id, is_resolved);

-- Singleton app_settings row.
INSERT OR IGNORE INTO app_settings (id, default_install_mode, database_version)
VALUES (1, 'symlink', 1);
