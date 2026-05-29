-- 000022_plugin_update_check_cache.up.sql
-- Add plugin update-check cache table and network settings (default OFF).

CREATE TABLE plugin_update_check_cache (
  id                INTEGER PRIMARY KEY,
  provider_key      TEXT NOT NULL,
  plugin_name       TEXT NOT NULL,
  marketplace_name  TEXT NOT NULL,
  source_url        TEXT NOT NULL,
  source_ref        TEXT,
  installed_sha     TEXT,
  installed_version TEXT,
  remote_sha        TEXT,
  remote_latest_tag TEXT,
  update_available  INTEGER,              -- 0=false / 1=true / NULL=unknown
  checked_at        TEXT NOT NULL,
  error             TEXT,
  UNIQUE(provider_key, plugin_name, marketplace_name)
);

CREATE TABLE network_settings (
  id                    INTEGER PRIMARY KEY CHECK (id = 1),
  update_check_enabled  INTEGER NOT NULL DEFAULT 0,  -- 0=off (privacy default), 1=on
  cache_ttl_hours       INTEGER NOT NULL DEFAULT 6,
  created_at            TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
  updated_at            TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

INSERT INTO network_settings (id, update_check_enabled, cache_ttl_hours)
VALUES (1, 0, 6);

UPDATE app_settings
   SET database_version = 22, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
