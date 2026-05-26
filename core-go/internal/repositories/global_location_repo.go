package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// GlobalLocationRepo reads global_provider_locations and global_installs.
type GlobalLocationRepo struct{ db *sql.DB }

func NewGlobalLocationRepo(db *sql.DB) *GlobalLocationRepo { return &GlobalLocationRepo{db: db} }

// ProviderDefByKey returns id, display_name, and status for a provider key.
func (r *GlobalLocationRepo) ProviderDefByKey(ctx context.Context, key string) (id int64, displayName, status string, err error) {
	err = r.db.QueryRowContext(ctx,
		`SELECT id, display_name, status FROM provider_definitions WHERE key=?`, key,
	).Scan(&id, &displayName, &status)
	if err == sql.ErrNoRows {
		return 0, "", "", fmt.Errorf("provider definition not found: %q", key)
	}
	return id, displayName, status, err
}

// ListForView returns all persisted global_provider_locations joined to provider_definitions,
// each with its global_installs (ordered by skill_name) and active warnings.
func (r *GlobalLocationRepo) ListForView(ctx context.Context) ([]domain.GlobalLocationView, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT gl.id, pd.key, pd.display_name, pd.status,
		       gl.path, gl.skills_path, gl.status, gl.last_scanned_at
		FROM global_provider_locations gl
		JOIN provider_definitions pd ON pd.id = gl.provider_definition_id
		ORDER BY pd.key`)
	if err != nil {
		return nil, fmt.Errorf("list global locations: %w", err)
	}
	defer rows.Close()

	var locations []domain.GlobalLocationView
	for rows.Next() {
		var loc domain.GlobalLocationView
		var path, skillsPath, lastScanned sql.NullString
		if err := rows.Scan(
			&loc.GlobalProviderLocationID,
			&loc.ProviderKey,
			&loc.ProviderDisplayName,
			&loc.ProviderStatus,
			&path, &skillsPath, &loc.Status, &lastScanned,
		); err != nil {
			return nil, fmt.Errorf("scan location row: %w", err)
		}
		if path.Valid {
			loc.Path = &path.String
		}
		if skillsPath.Valid {
			loc.SkillsPath = &skillsPath.String
		}
		if lastScanned.Valid {
			loc.LastScannedAt = &lastScanned.String
		}
		locations = append(locations, loc)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate locations: %w", err)
	}

	for i := range locations {
		loc := &locations[i]

		// Load installs ordered by skill_name.
		instRows, err := r.db.QueryContext(ctx, `
			SELECT id, skill_id, skill_name, install_mode, install_status,
			       global_skill_path, source_skill_path, symlink_target_path
			FROM global_installs
			WHERE global_provider_location_id=?
			ORDER BY skill_name`, loc.GlobalProviderLocationID)
		if err != nil {
			return nil, fmt.Errorf("list global installs for location %d: %w", loc.GlobalProviderLocationID, err)
		}
		defer instRows.Close()

		for instRows.Next() {
			var inst domain.GlobalInstallView
			var skillID sql.NullInt64
			var source, symlink sql.NullString
			if err := instRows.Scan(
				&inst.GlobalInstallID, &skillID, &inst.SkillName,
				&inst.Mode, &inst.Status,
				&inst.GlobalSkillPath, &source, &symlink,
			); err != nil {
				return nil, fmt.Errorf("scan install row: %w", err)
			}
			if skillID.Valid {
				v := skillID.Int64
				inst.SkillID = &v
			}
			if source.Valid {
				inst.SourceSkillPath = &source.String
			}
			if symlink.Valid {
				inst.SymlinkTargetPath = &symlink.String
			}
			loc.Entries = append(loc.Entries, inst)
		}
		if err := instRows.Err(); err != nil {
			return nil, fmt.Errorf("iterate installs: %w", err)
		}

		// Load active warnings for this location and its installs.
		warnRows, err := r.db.QueryContext(ctx, `
			SELECT scope_type, scope_id, severity, code, message, action_key
			FROM warnings
			WHERE is_resolved=0 AND (
				(scope_type='global_provider_location' AND scope_id=?)
				OR (scope_type='global_install' AND scope_id IN (
					SELECT id FROM global_installs WHERE global_provider_location_id=?
				))
			)
			ORDER BY id`, loc.GlobalProviderLocationID, loc.GlobalProviderLocationID)
		if err != nil {
			return nil, fmt.Errorf("list warnings for location %d: %w", loc.GlobalProviderLocationID, err)
		}
		defer warnRows.Close()

		for warnRows.Next() {
			var w domain.Warning
			var scopeID sql.NullInt64
			var actionKey sql.NullString
			if err := warnRows.Scan(
				&w.ScopeType, &scopeID, &w.Severity, &w.Code, &w.Message, &actionKey,
			); err != nil {
				return nil, fmt.Errorf("scan warning row: %w", err)
			}
			if scopeID.Valid {
				v := scopeID.Int64
				w.ScopeID = &v
			}
			if actionKey.Valid {
				w.ActionKey = &actionKey.String
			}
			loc.Warnings = append(loc.Warnings, w)
		}
		if err := warnRows.Err(); err != nil {
			return nil, fmt.Errorf("iterate warnings: %w", err)
		}

		if loc.Entries == nil {
			loc.Entries = []domain.GlobalInstallView{}
		}
		if loc.Warnings == nil {
			loc.Warnings = []domain.Warning{}
		}
		locations[i] = *loc
	}

	if locations == nil {
		locations = []domain.GlobalLocationView{}
	}
	return locations, nil
}
