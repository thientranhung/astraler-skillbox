package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type NetworkSettingsRepo struct {
	db *sql.DB
}

func NewNetworkSettingsRepo(db *sql.DB) *NetworkSettingsRepo {
	return &NetworkSettingsRepo{db: db}
}

func (r *NetworkSettingsRepo) Get(ctx context.Context) (*domain.NetworkSettings, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, cache_ttl_hours, created_at, updated_at
		   FROM network_settings WHERE id = 1`)
	s := &domain.NetworkSettings{}
	var createdAt, updatedAt string
	if err := row.Scan(&s.ID, &s.CacheTTLHours, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("network_settings get: %w", err)
	}
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return s, nil
}

func (r *NetworkSettingsRepo) SetCacheTTLHours(ctx context.Context, hours int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE network_settings SET cache_ttl_hours = ?,
		 updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = 1`, hours)
	return err
}
