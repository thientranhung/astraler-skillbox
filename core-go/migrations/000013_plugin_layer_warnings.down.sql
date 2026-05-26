ALTER TABLE provider_plugin_layer_scans DROP COLUMN scan_warnings;
UPDATE app_settings SET database_version = 12 WHERE id = 1;
