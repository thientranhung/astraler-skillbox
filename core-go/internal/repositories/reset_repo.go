package repositories

import (
	"context"
	"database/sql"
	"fmt"
)

// ResetAllData truncates all user data tables and resets singleton settings rows
// to their defaults, in a single transaction. The DB remains open and the schema
// is untouched — no file deletion, no close.
func ResetAllData(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("reset: begin tx: %w", err)
	}
	defer tx.Rollback()

	stmts := []string{
		// Clear app_settings FK reference before skill_host_folders is deleted.
		`UPDATE app_settings
		    SET active_skill_host_folder_id = NULL,
		        default_install_mode        = 'symlink',
		        updated_at                  = strftime('%Y-%m-%dT%H:%M:%SZ','now')
		  WHERE id = 1`,

		// Delete children before parents (FK order).
		`DELETE FROM installs`,
		`DELETE FROM global_installs`,
		`DELETE FROM skill_sources`,
		`DELETE FROM warnings`,
		`DELETE FROM project_providers`,
		`DELETE FROM provider_plugin_marketplaces`,
		`DELETE FROM provider_plugin_layer_scans`,
		`DELETE FROM provider_plugin_entries`,
		`DELETE FROM skills`,
		`DELETE FROM skill_host_folders`,
		`DELETE FROM projects`,
		`DELETE FROM global_provider_locations`,
		`DELETE FROM operations`,
		`DELETE FROM provider_user_settings`,
		`DELETE FROM provider_path_overrides`,
		`DELETE FROM plugin_update_check_cache`,

		// Reset singleton network settings row to defaults. update_check is
		// always-on (ADR-0002); only cache_ttl_hours is resettable.
		`UPDATE network_settings
		    SET cache_ttl_hours = 6,
		        updated_at      = strftime('%Y-%m-%dT%H:%M:%SZ','now')
		  WHERE id = 1`,
	}

	for _, s := range stmts {
		if _, err := tx.ExecContext(ctx, s); err != nil {
			return fmt.Errorf("reset: exec %q: %w", s[:min(40, len(s))], err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("reset: commit: %w", err)
	}
	return nil
}
