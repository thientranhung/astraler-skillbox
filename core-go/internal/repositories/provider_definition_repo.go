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

// ListAll returns all provider definitions with their path candidates.
func (r *ProviderDefinitionRepo) ListAll(ctx context.Context) ([]domain.ProviderRegistryEntry, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, key, display_name, provider_type, icon_key, status,
		        can_create_structure, has_global_level, created_at, updated_at
		   FROM provider_definitions
		  ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var defs []domain.ProviderDefinition
	for rows.Next() {
		pd, err := scanProviderDefinitionRow(rows)
		if err != nil {
			return nil, err
		}
		defs = append(defs, *pd)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	candidatesByProvider, err := r.listAllCandidates(ctx)
	if err != nil {
		return nil, err
	}

	entries := make([]domain.ProviderRegistryEntry, len(defs))
	for i, d := range defs {
		entries[i] = domain.ProviderRegistryEntry{
			Definition: d,
			Candidates: candidatesByProvider[d.ID],
		}
		if entries[i].Candidates == nil {
			entries[i].Candidates = []domain.ProviderPathCandidate{}
		}
	}
	return entries, nil
}

func (r *ProviderDefinitionRepo) listAllCandidates(ctx context.Context) (map[int64][]domain.ProviderPathCandidate, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, provider_definition_id, relative_path, scope, purpose, priority, verification_status
		   FROM provider_path_candidates
		  ORDER BY provider_definition_id ASC, priority DESC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64][]domain.ProviderPathCandidate)
	for rows.Next() {
		c := domain.ProviderPathCandidate{}
		var scope, verStatus sql.NullString
		if err := rows.Scan(&c.ID, &c.ProviderDefinitionID, &c.RelativePath,
			&scope, &c.Purpose, &c.Priority, &verStatus); err != nil {
			return nil, err
		}
		if scope.Valid {
			c.Scope = scope.String
		} else {
			c.Scope = "project"
		}
		if verStatus.Valid {
			c.VerificationStatus = verStatus.String
		} else {
			c.VerificationStatus = "assumed"
		}
		result[c.ProviderDefinitionID] = append(result[c.ProviderDefinitionID], c)
	}
	return result, rows.Err()
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

func scanProviderDefinitionRow(rows *sql.Rows) (*domain.ProviderDefinition, error) {
	pd := &domain.ProviderDefinition{}
	var iconKey sql.NullString
	var canCreate, hasGlobal int
	var createdAt, updatedAt string

	err := rows.Scan(&pd.ID, &pd.Key, &pd.DisplayName, &pd.ProviderType, &iconKey,
		&pd.Status, &canCreate, &hasGlobal, &createdAt, &updatedAt)
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
