package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type AppSettingsRepo struct {
	db *sql.DB
}

func NewAppSettingsRepo(db *sql.DB) *AppSettingsRepo {
	return &AppSettingsRepo{db: db}
}

func (r *AppSettingsRepo) Get(ctx context.Context) (*domain.AppSettings, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, active_skill_host_folder_id, default_install_mode, database_version,
		        created_at, updated_at
		   FROM app_settings WHERE id = 1`)

	s := &domain.AppSettings{}
	var createdAt, updatedAt string
	var activeID sql.NullInt64
	if err := row.Scan(&s.ID, &activeID, &s.DefaultInstallMode, &s.DatabaseVersion,
		&createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("app_settings get: %w", err)
	}
	if activeID.Valid {
		id := activeID.Int64
		s.ActiveSkillHostFolderID = &id
	}
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return s, nil
}

func (r *AppSettingsRepo) UpdateActiveHost(ctx context.Context, hostID *int64) error {
	var err error
	if hostID == nil {
		_, err = r.db.ExecContext(ctx,
			`UPDATE app_settings SET active_skill_host_folder_id = NULL,
			 updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = 1`)
	} else {
		_, err = r.db.ExecContext(ctx,
			`UPDATE app_settings SET active_skill_host_folder_id = ?,
			 updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id = 1`, *hostID)
	}
	return err
}
