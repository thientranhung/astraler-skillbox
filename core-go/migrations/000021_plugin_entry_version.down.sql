-- 000021_plugin_entry_version.down.sql

ALTER TABLE provider_plugin_entries DROP COLUMN version;

UPDATE app_settings
   SET database_version = 20, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
