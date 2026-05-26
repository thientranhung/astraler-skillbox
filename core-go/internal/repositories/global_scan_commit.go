package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// GlobalInstallScanResult carries one observed filesystem entry for CommitGlobalScan.
// Warning (if set) must have ScopeType=WarningScopeGlobalInstall; scope_id is filled in by commit.
type GlobalInstallScanResult struct {
	SkillID                   *int64
	SkillName                 string
	InstallMode               domain.InstallMode
	InstallStatus             domain.InstallStatus
	GlobalSkillPath           string
	SourceSkillPath           *string
	SymlinkTargetPath         *string
	InstalledFromHostFolderID *int64
	Warning                   *domain.Warning
}

// GlobalScanRepo persists one provider's global scan atomically.
type GlobalScanRepo struct{ db *sql.DB }

func NewGlobalScanRepo(db *sql.DB) *GlobalScanRepo { return &GlobalScanRepo{db: db} }

// CommitGlobalScan persists one provider's global scan atomically:
//  1. Upsert global_provider_locations by provider_definition_id; capture location id.
//  2. Clear active warnings scoped to this location and its global_installs.
//  3. Upsert present installs; DELETE installs no longer on disk for this location.
//  4. Insert location-scoped warnings (scope_id=locationID) and install-scoped warnings.
//  5. Update location.last_scanned_at + status.
func (r *GlobalScanRepo) CommitGlobalScan(
	ctx context.Context,
	providerDefID int64,
	path, skillsPath *string,
	status domain.GlobalLocationStatus,
	installs []GlobalInstallScanResult,
	locationWarnings []domain.Warning,
	now time.Time,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin global scan tx: %w", err)
	}
	defer tx.Rollback()

	nowStr := now.UTC().Format(time.RFC3339)

	// 1. Upsert global_provider_locations by provider_definition_id.
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO global_provider_locations
		  (provider_definition_id, path, skills_path, status, last_scanned_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(provider_definition_id) DO UPDATE SET
		  path=excluded.path,
		  skills_path=excluded.skills_path,
		  status=excluded.status,
		  last_scanned_at=excluded.last_scanned_at,
		  updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
		providerDefID, nullableStr(path), nullableStr(skillsPath), string(status), nowStr,
	); err != nil {
		return fmt.Errorf("upsert global_provider_locations: %w", err)
	}

	var locationID int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM global_provider_locations WHERE provider_definition_id=?`,
		providerDefID).Scan(&locationID); err != nil {
		return fmt.Errorf("select location id: %w", err)
	}

	// 2. Clear active warnings for this location and its installs.
	if _, err := tx.ExecContext(ctx, `
		UPDATE warnings SET is_resolved=1,
		 resolved_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		WHERE is_resolved=0 AND (
			(scope_type='global_provider_location' AND scope_id=?)
			OR (scope_type='global_install' AND scope_id IN (
				SELECT id FROM global_installs WHERE global_provider_location_id=?
			))
		)`, locationID, locationID,
	); err != nil {
		return fmt.Errorf("clear global warnings: %w", err)
	}

	// 3. Upsert present installs and collect their ids for warning attachment.
	seenPaths := make([]string, 0, len(installs))
	for _, inst := range installs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO global_installs
			  (global_provider_location_id, skill_id, skill_name, install_mode, install_status,
			   global_skill_path, source_skill_path, symlink_target_path,
			   installed_from_host_folder_id, last_scanned_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(global_provider_location_id, global_skill_path) DO UPDATE SET
			  skill_id=excluded.skill_id,
			  skill_name=excluded.skill_name,
			  install_mode=excluded.install_mode,
			  install_status=excluded.install_status,
			  source_skill_path=excluded.source_skill_path,
			  symlink_target_path=excluded.symlink_target_path,
			  installed_from_host_folder_id=excluded.installed_from_host_folder_id,
			  last_scanned_at=excluded.last_scanned_at,
			  updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
			locationID, ptrToSQL(inst.SkillID), inst.SkillName,
			string(inst.InstallMode), string(inst.InstallStatus),
			inst.GlobalSkillPath,
			nullableStr(inst.SourceSkillPath), nullableStr(inst.SymlinkTargetPath),
			ptrToSQL(inst.InstalledFromHostFolderID), nowStr,
		); err != nil {
			return fmt.Errorf("upsert global_install %q: %w", inst.GlobalSkillPath, err)
		}
		seenPaths = append(seenPaths, inst.GlobalSkillPath)

		if inst.Warning != nil {
			var installID int64
			if err := tx.QueryRowContext(ctx,
				`SELECT id FROM global_installs WHERE global_provider_location_id=? AND global_skill_path=?`,
				locationID, inst.GlobalSkillPath).Scan(&installID); err != nil {
				return fmt.Errorf("select install id for warning: %w", err)
			}
			w := *inst.Warning
			w.ScopeID = &installID
			if err := insertWarningTx(ctx, tx, w); err != nil {
				return fmt.Errorf("insert install warning: %w", err)
			}
		}
	}

	// Delete installs no longer on disk for this location.
	if err := deleteAbsentGlobalInstalls(ctx, tx, locationID, seenPaths); err != nil {
		return fmt.Errorf("delete absent global installs: %w", err)
	}

	// 4. Insert location-scoped warnings.
	for _, w := range locationWarnings {
		w.ScopeID = &locationID
		if err := insertWarningTx(ctx, tx, w); err != nil {
			return fmt.Errorf("insert location warning: %w", err)
		}
	}

	return tx.Commit()
}

func deleteAbsentGlobalInstalls(ctx context.Context, tx *sql.Tx, locationID int64, seenPaths []string) error {
	if len(seenPaths) == 0 {
		_, err := tx.ExecContext(ctx,
			`DELETE FROM global_installs WHERE global_provider_location_id=?`, locationID)
		return err
	}
	ph := strings.Repeat("?,", len(seenPaths))
	ph = ph[:len(ph)-1]
	args := make([]interface{}, 0, 1+len(seenPaths))
	args = append(args, locationID)
	for _, p := range seenPaths {
		args = append(args, p)
	}
	_, err := tx.ExecContext(ctx,
		"DELETE FROM global_installs WHERE global_provider_location_id=? AND global_skill_path NOT IN ("+ph+")",
		args...)
	return err
}
