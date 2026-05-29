-- 000021_plugin_entry_version.up.sql
-- Add nullable version column to provider_plugin_entries.
-- NULL = version unknown (non-Claude providers, or plugin not in installed_plugins.json).

ALTER TABLE provider_plugin_entries ADD COLUMN version TEXT;

UPDATE app_settings
   SET database_version = 21, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
