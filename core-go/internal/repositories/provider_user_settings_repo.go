package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type ProviderUserSettingsRepo struct {
	db *sql.DB
}

func NewProviderUserSettingsRepo(db *sql.DB) *ProviderUserSettingsRepo {
	return &ProviderUserSettingsRepo{db: db}
}

// ListAll returns all user-set provider preferences ordered by provider_definition_id.
func (r *ProviderUserSettingsRepo) ListAll(ctx context.Context) ([]domain.ProviderUserSetting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, provider_definition_id, enabled
		   FROM provider_user_settings
		  ORDER BY provider_definition_id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProviderUserSetting
	for rows.Next() {
		var s domain.ProviderUserSetting
		var enabledInt int
		if err := rows.Scan(&s.ID, &s.ProviderDefinitionID, &enabledInt); err != nil {
			return nil, err
		}
		s.Enabled = enabledInt != 0
		result = append(result, s)
	}
	return result, rows.Err()
}

// Upsert inserts or replaces the enabled preference for providerDefinitionID.
func (r *ProviderUserSettingsRepo) Upsert(ctx context.Context, providerDefinitionID int64, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO provider_user_settings (provider_definition_id, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(provider_definition_id)
		DO UPDATE SET enabled = excluded.enabled, updated_at = excluded.updated_at
	`, providerDefinitionID, enabledInt, now, now)
	return err
}
