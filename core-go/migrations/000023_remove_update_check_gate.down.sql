-- 000023_remove_update_check_gate.down.sql
-- Re-add the gate column to restore the 000022 schema shape. Data dropped on the
-- up migration is unrecoverable (the column was an unused gate). The row is set to
-- 1 so a rollback to pre-ADR-0002 code does not re-disable the feature.

ALTER TABLE network_settings ADD COLUMN update_check_enabled INTEGER NOT NULL DEFAULT 0;
UPDATE network_settings SET update_check_enabled = 1 WHERE id = 1;

UPDATE app_settings
   SET database_version = 22, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
