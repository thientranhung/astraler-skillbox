DROP TABLE IF EXISTS provider_plugin_marketplaces;
DROP TABLE IF EXISTS provider_plugin_entries;
DROP TABLE IF EXISTS provider_plugin_layer_scans;

UPDATE app_settings SET database_version = 11 WHERE id = 1;
