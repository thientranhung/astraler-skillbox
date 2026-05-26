ALTER TABLE provider_plugin_layer_scans ADD COLUMN scan_warnings TEXT NOT NULL DEFAULT '[]';
UPDATE app_settings SET database_version = 13 WHERE id = 1;
