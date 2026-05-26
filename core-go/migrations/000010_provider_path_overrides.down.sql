-- 000010_provider_path_overrides.down.sql
DROP TABLE IF EXISTS provider_path_overrides;

UPDATE app_settings
   SET database_version = 9, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
