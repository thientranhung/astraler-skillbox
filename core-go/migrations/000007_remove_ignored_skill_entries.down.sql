UPDATE app_settings
   SET database_version = 6, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
