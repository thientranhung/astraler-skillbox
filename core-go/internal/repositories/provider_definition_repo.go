package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type ProviderDefinitionRepo struct {
	db *sql.DB
}

func NewProviderDefinitionRepo(db *sql.DB) *ProviderDefinitionRepo {
	return &ProviderDefinitionRepo{db: db}
}

func (r *ProviderDefinitionRepo) GetByKey(ctx context.Context, key string) (*domain.ProviderDefinition, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, key, display_name, provider_type, icon_key, status,
		        can_create_structure, has_global_level, created_at, updated_at
		   FROM provider_definitions WHERE key = ?`, key)
	return scanProviderDefinition(row)
}

func scanProviderDefinition(row *sql.Row) (*domain.ProviderDefinition, error) {
	pd := &domain.ProviderDefinition{}
	var iconKey sql.NullString
	var canCreate, hasGlobal int
	var createdAt, updatedAt string

	err := row.Scan(&pd.ID, &pd.Key, &pd.DisplayName, &pd.ProviderType, &iconKey,
		&pd.Status, &canCreate, &hasGlobal, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if iconKey.Valid {
		pd.IconKey = &iconKey.String
	}
	pd.CanCreateStructure = canCreate != 0
	pd.HasGlobalLevel = hasGlobal != 0
	pd.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	pd.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return pd, nil
}
