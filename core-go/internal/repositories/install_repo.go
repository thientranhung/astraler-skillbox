package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type InstallRepo struct {
	db *sql.DB
}

func NewInstallRepo(db *sql.DB) *InstallRepo {
	return &InstallRepo{db: db}
}

// ListByProject returns all installs for a given project by joining through project_providers.
func (r *InstallRepo) ListByProject(ctx context.Context, projectID int64) ([]domain.Install, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT i.id, i.project_provider_id, i.skill_id, i.skill_name,
		       i.install_mode, i.install_status, i.project_skill_path,
		       i.source_skill_path, i.symlink_target_path, i.installed_from_host_folder_id,
		       i.installed_version, i.installed_commit, i.installed_checksum,
		       i.last_synced_at, i.last_scanned_at, i.created_at, i.updated_at
		  FROM installs i
		  JOIN project_providers pp ON pp.id = i.project_provider_id
		 WHERE pp.project_id = ?
		 ORDER BY i.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var installs []domain.Install
	for rows.Next() {
		inst, err := scanInstall(rows)
		if err != nil {
			return nil, err
		}
		installs = append(installs, inst)
	}
	return installs, rows.Err()
}

// DeleteByID hard-deletes a single install row. It is the only hard delete of an
// install row in the app. Idempotent: deleting an absent id affects 0 rows and
// is not an error.
func (r *InstallRepo) DeleteByID(ctx context.Context, installID int64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM installs WHERE id = ?`, installID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanInstall(rows *sql.Rows) (domain.Install, error) {
	var inst domain.Install
	var skillID, installedFromHostFolderID sql.NullInt64
	var sourceSkillPath, symlinkTargetPath, installedVersion, installedCommit, installedChecksum sql.NullString
	var lastSyncedAt, lastScannedAt sql.NullString
	var createdAt, updatedAt string

	err := rows.Scan(
		&inst.ID, &inst.ProjectProviderID, &skillID, &inst.SkillName,
		&inst.InstallMode, &inst.InstallStatus, &inst.ProjectSkillPath,
		&sourceSkillPath, &symlinkTargetPath, &installedFromHostFolderID,
		&installedVersion, &installedCommit, &installedChecksum,
		&lastSyncedAt, &lastScannedAt, &createdAt, &updatedAt,
	)
	if err != nil {
		return inst, err
	}
	if skillID.Valid {
		id := skillID.Int64
		inst.SkillID = &id
	}
	if installedFromHostFolderID.Valid {
		id := installedFromHostFolderID.Int64
		inst.InstalledFromHostFolderID = &id
	}
	if sourceSkillPath.Valid {
		inst.SourceSkillPath = &sourceSkillPath.String
	}
	if symlinkTargetPath.Valid {
		inst.SymlinkTargetPath = &symlinkTargetPath.String
	}
	if installedVersion.Valid {
		inst.InstalledVersion = &installedVersion.String
	}
	if installedCommit.Valid {
		inst.InstalledCommit = &installedCommit.String
	}
	if installedChecksum.Valid {
		inst.InstalledChecksum = &installedChecksum.String
	}
	if lastSyncedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastSyncedAt.String)
		inst.LastSyncedAt = &t
	}
	if lastScannedAt.Valid {
		t, _ := time.Parse(time.RFC3339, lastScannedAt.String)
		inst.LastScannedAt = &t
	}
	inst.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	inst.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return inst, nil
}
