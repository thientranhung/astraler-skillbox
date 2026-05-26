package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type ProviderOverrideRepo struct {
	db *sql.DB
}

func NewProviderOverrideRepo(db *sql.DB) *ProviderOverrideRepo {
	return &ProviderOverrideRepo{db: db}
}

// GetProviderIDByKey returns the provider_definition.id for the given key,
// or 0 if the key does not exist.
func (r *ProviderOverrideRepo) GetProviderIDByKey(ctx context.Context, key string) (int64, error) {
	var id int64
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM provider_definitions WHERE key = ?`, key,
	).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

// ListAll returns all overrides ordered by provider_definition_id, scope, purpose.
func (r *ProviderOverrideRepo) ListAll(ctx context.Context) ([]domain.ProviderPathOverride, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, provider_definition_id, scope, purpose, paths_json
		   FROM provider_path_overrides
		  ORDER BY provider_definition_id ASC, scope ASC, purpose ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.ProviderPathOverride
	for rows.Next() {
		var o domain.ProviderPathOverride
		var pathsJSON string
		if err := rows.Scan(&o.ID, &o.ProviderDefinitionID, &o.Scope, &o.Purpose, &pathsJSON); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(pathsJSON), &o.Paths); err != nil {
			return nil, err
		}
		result = append(result, o)
	}
	return result, rows.Err()
}

// Upsert inserts or replaces the override for (provider_definition_id, scope, purpose).
func (r *ProviderOverrideRepo) Upsert(ctx context.Context, o domain.ProviderPathOverride) error {
	pathsJSON, err := json.Marshal(o.Paths)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO provider_path_overrides (provider_definition_id, scope, purpose, paths_json, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider_definition_id, scope, purpose)
		DO UPDATE SET paths_json = excluded.paths_json, updated_at = excluded.updated_at
	`, o.ProviderDefinitionID, o.Scope, o.Purpose, string(pathsJSON), now, now)
	return err
}

// Delete removes the override for (providerDefinitionID, scope, purpose).
// Returns true if a row was deleted, false if none existed.
func (r *ProviderOverrideRepo) Delete(ctx context.Context, providerDefinitionID int64, scope, purpose string) (bool, error) {
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM provider_path_overrides
		 WHERE provider_definition_id = ? AND scope = ? AND purpose = ?
	`, providerDefinitionID, scope, purpose)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n > 0, err
}
