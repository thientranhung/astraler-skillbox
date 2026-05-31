-- 000023_remove_update_check_gate.up.sql
-- Option A (ADR-0002): plugin update-check is always-on. Remove the network
-- opt-in gate column. The network_settings table is retained for cache_ttl_hours.
-- DROP COLUMN requires SQLite >= 3.35 (driver modernc.org/sqlite v1.50.1 → 3.53.1).

ALTER TABLE network_settings DROP COLUMN update_check_enabled;

UPDATE app_settings
   SET database_version = 23, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
