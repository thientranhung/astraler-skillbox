DROP TABLE IF EXISTS provider_user_settings;

UPDATE app_settings SET database_version = 10 WHERE id = 1;
