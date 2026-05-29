-- 000022_plugin_update_check_cache.down.sql

DROP TABLE IF EXISTS network_settings;
DROP TABLE IF EXISTS plugin_update_check_cache;

UPDATE app_settings
   SET database_version = 21, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
