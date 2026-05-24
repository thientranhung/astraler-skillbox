package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type SkillHostFolderRepo struct {
	db *sql.DB
}

func NewSkillHostFolderRepo(db *sql.DB) *SkillHostFolderRepo {
	return &SkillHostFolderRepo{db: db}
}

func (r *SkillHostFolderRepo) GetByID(ctx context.Context, id int64) (*domain.SkillHostFolder, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, path, skills_path, status, last_scanned_at, created_at, updated_at
		   FROM skill_host_folders WHERE id = ?`, id)
	return scanHost(row)
}

func (r *SkillHostFolderRepo) GetByPath(ctx context.Context, path string) (*domain.SkillHostFolder, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, path, skills_path, status, last_scanned_at, created_at, updated_at
		   FROM skill_host_folders WHERE path = ?`, path)
	h, err := scanHost(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

func (r *SkillHostFolderRepo) GetActive(ctx context.Context) (*domain.SkillHostFolder, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT h.id, h.name, h.path, h.skills_path, h.status, h.last_scanned_at, h.created_at, h.updated_at
		   FROM skill_host_folders h
		   JOIN app_settings s ON s.active_skill_host_folder_id = h.id
		  WHERE s.id = 1`)
	h, err := scanHost(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return h, err
}

// UpsertAndActivate atomically upserts the host by path, activates it, and
// deactivates the previously active host in a single transaction.
// Returns (hostID, isNew, err). isNew=true means the row was INSERTed.
func (r *SkillHostFolderRepo) UpsertAndActivate(ctx context.Context, name, path, skillsPath string) (int64, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()

	var hostID int64
	var isNew bool

	// Check if host with this path already exists.
	var existing sql.NullInt64
	_ = tx.QueryRowContext(ctx, `SELECT id FROM skill_host_folders WHERE path = ?`, path).Scan(&existing)

	if existing.Valid {
		hostID = existing.Int64
		_, err = tx.ExecContext(ctx,
			`UPDATE skill_host_folders SET status='active', name=?,
			 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`, name, hostID)
		if err != nil {
			return 0, false, fmt.Errorf("update host: %w", err)
		}
	} else {
		res, err := tx.ExecContext(ctx,
			`INSERT INTO skill_host_folders (name, path, skills_path, status)
			 VALUES (?, ?, ?, 'active')`, name, path, skillsPath)
		if err != nil {
			return 0, false, fmt.Errorf("insert host: %w", err)
		}
		hostID, err = res.LastInsertId()
		if err != nil {
			return 0, false, err
		}
		isNew = true
	}

	// Deactivate previous active host if different.
	var currentActive sql.NullInt64
	_ = tx.QueryRowContext(ctx,
		`SELECT active_skill_host_folder_id FROM app_settings WHERE id=1`).Scan(&currentActive)

	if currentActive.Valid && currentActive.Int64 != hostID {
		_, err = tx.ExecContext(ctx,
			`UPDATE skill_host_folders SET status='inactive',
			 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
			currentActive.Int64)
		if err != nil {
			return 0, false, fmt.Errorf("deactivate old host: %w", err)
		}
	}

	// Update app_settings.
	_, err = tx.ExecContext(ctx,
		`UPDATE app_settings SET active_skill_host_folder_id=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=1`, hostID)
	if err != nil {
		return 0, false, fmt.Errorf("update app_settings: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return hostID, isNew, nil
}

func (r *SkillHostFolderRepo) UpdateStatus(ctx context.Context, id int64, status domain.SkillHostStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE skill_host_folders SET status=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`, string(status), id)
	return err
}

func (r *SkillHostFolderRepo) UpdateLastScannedAt(ctx context.Context, id int64, t time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE skill_host_folders SET last_scanned_at=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		t.UTC().Format(time.RFC3339), id)
	return err
}

func scanHost(row *sql.Row) (*domain.SkillHostFolder, error) {
	h := &domain.SkillHostFolder{}
	var name sql.NullString
	var lastScanned sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&h.ID, &name, &h.Path, &h.SkillsPath, &h.Status,
		&lastScanned, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if name.Valid {
		h.Name = name.String
	}
	if lastScanned.Valid {
		t, _ := time.Parse(time.RFC3339, lastScanned.String)
		h.LastScannedAt = &t
	}
	h.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	h.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return h, nil
}
