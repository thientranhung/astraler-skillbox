package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// ProjectProviderRow is a joined view of project_providers + provider_definitions + install count.
type ProjectProviderRow struct {
	ID                   int64
	ProjectID            int64
	ProviderDefinitionID int64
	ProviderKey          string
	ProviderDisplayName  string
	ProviderStatus       domain.ProviderStatus
	DetectedPath         *string
	SkillsPath           *string
	DetectionStatus      domain.DetectionStatus
	LastScannedAt        *time.Time
	EntryCount           int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type ProjectProviderRepo struct {
	db *sql.DB
}

func NewProjectProviderRepo(db *sql.DB) *ProjectProviderRepo {
	return &ProjectProviderRepo{db: db}
}

// ListByProject returns all project_providers for a project joined with provider_definitions
// and a COUNT of installs (observed entries) for each provider.
func (r *ProjectProviderRepo) ListByProject(ctx context.Context, projectID int64) ([]ProjectProviderRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			pp.id, pp.project_id, pp.provider_definition_id,
			pd.key, pd.display_name, pd.status,
			pp.detected_path, pp.skills_path, pp.detection_status, pp.last_scanned_at,
			COUNT(i.id) AS entry_count,
			pp.created_at, pp.updated_at
		FROM project_providers pp
		JOIN provider_definitions pd ON pd.id = pp.provider_definition_id
		LEFT JOIN installs i ON i.project_provider_id = pp.id
		WHERE pp.project_id = ?
		GROUP BY pp.id
		ORDER BY pp.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProjectProviderRow
	for rows.Next() {
		row, err := scanProjectProviderRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func scanProjectProviderRow(rows *sql.Rows) (ProjectProviderRow, error) {
	var r ProjectProviderRow
	var detectedPath, skillsPath, lastScanned sql.NullString
	var createdAt, updatedAt string

	err := rows.Scan(
		&r.ID, &r.ProjectID, &r.ProviderDefinitionID,
		&r.ProviderKey, &r.ProviderDisplayName, &r.ProviderStatus,
		&detectedPath, &skillsPath, &r.DetectionStatus, &lastScanned,
		&r.EntryCount,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return r, err
	}
	if detectedPath.Valid {
		r.DetectedPath = &detectedPath.String
	}
	if skillsPath.Valid {
		r.SkillsPath = &skillsPath.String
	}
	if lastScanned.Valid {
		t, _ := time.Parse(time.RFC3339, lastScanned.String)
		r.LastScannedAt = &t
	}
	r.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	r.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return r, nil
}
