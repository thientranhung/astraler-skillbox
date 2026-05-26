CREATE TABLE IF NOT EXISTS provider_user_settings (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_definition_id INTEGER NOT NULL UNIQUE REFERENCES provider_definitions(id) ON DELETE CASCADE,
    enabled                INTEGER NOT NULL CHECK(enabled IN (0, 1)),
    created_at             TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at             TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

UPDATE app_settings SET database_version = 11 WHERE id = 1;
