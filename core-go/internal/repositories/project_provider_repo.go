package repositories

import (
	"context"
	"database/sql"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// projectProviderRow is an internal scan type for the joined SQL query.
// The public surface returns domain.ProjectProviderSummary.
type projectProviderRow struct {
	id                   int64
	projectID            int64
	providerDefinitionID int64
	providerKey          string
	providerDisplayName  string
	providerStatus       domain.ProviderStatus
	detectedPath         *string
	skillsPath           *string
	detectionStatus      domain.DetectionStatus
	entryCount           int
}

type ProjectProviderRepo struct {
	db *sql.DB
}

func NewProjectProviderRepo(db *sql.DB) *ProjectProviderRepo {
	return &ProjectProviderRepo{db: db}
}

// ListByProject returns all project_providers for a project joined with provider_definitions
// and a COUNT of installs (observed entries) for each provider.
// Returns []domain.ProjectProviderSummary so *ProjectProviderRepo satisfies services.ProjectProviderRepo.
func (r *ProjectProviderRepo) ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectProviderSummary, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			pp.id, pp.project_id, pp.provider_definition_id,
			pd.key, pd.display_name, pd.status,
			pp.detected_path, pp.skills_path, pp.detection_status,
			COUNT(i.id) AS entry_count
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

	var result []domain.ProjectProviderSummary
	for rows.Next() {
		row, err := scanProjectProviderRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, toProviderSummary(row))
	}
	return result, rows.Err()
}

func scanProjectProviderRow(rows *sql.Rows) (projectProviderRow, error) {
	var r projectProviderRow
	var detectedPath, skillsPath sql.NullString

	err := rows.Scan(
		&r.id, &r.projectID, &r.providerDefinitionID,
		&r.providerKey, &r.providerDisplayName, &r.providerStatus,
		&detectedPath, &skillsPath, &r.detectionStatus,
		&r.entryCount,
	)
	if err != nil {
		return r, err
	}
	if detectedPath.Valid {
		r.detectedPath = &detectedPath.String
	}
	if skillsPath.Valid {
		r.skillsPath = &skillsPath.String
	}
	return r, nil
}

func toProviderSummary(r projectProviderRow) domain.ProjectProviderSummary {
	return domain.ProjectProviderSummary{
		ProjectProviderID:   r.id,
		ProviderKey:         r.providerKey,
		ProviderDisplayName: r.providerDisplayName,
		ProviderStatus:      r.providerStatus,
		DetectionStatus:     r.detectionStatus,
		DetectedPath:        r.detectedPath,
		SkillsPath:          r.skillsPath,
		EntryCount:          r.entryCount,
	}
}
