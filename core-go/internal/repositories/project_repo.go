package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type ProjectRepo struct {
	db *sql.DB
}

func NewProjectRepo(db *sql.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

// UpsertByPath inserts a new project or returns the existing one for that path.
// If a removed row exists for the path, it is revived to active with an updated name.
// Returns (id, isNew, err). isNew=true means the row was INSERTed.
func (r *ProjectRepo) UpsertByPath(ctx context.Context, name, path string) (int64, bool, error) {
	var existingID sql.NullInt64
	var existingStatus sql.NullString
	row := r.db.QueryRowContext(ctx,
		`SELECT id, status FROM projects WHERE path = ?`, path)
	_ = row.Scan(&existingID, &existingStatus)

	if existingID.Valid {
		if existingStatus.String == string(domain.ProjectStatusRemoved) {
			_, err := r.db.ExecContext(ctx,
				`UPDATE projects SET status='active', name=?,
				 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
				name, existingID.Int64)
			if err != nil {
				return 0, false, err
			}
		}
		return existingID.Int64, false, nil
	}

	res, err := r.db.ExecContext(ctx,
		`INSERT INTO projects (name, path, status) VALUES (?, ?, 'active')`, name, path)
	if err != nil {
		return 0, false, err
	}
	id, err := res.LastInsertId()
	return id, true, err
}

func (r *ProjectRepo) GetByID(ctx context.Context, id int64) (*domain.Project, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, name, path, status, last_scanned_at, created_at, updated_at
		   FROM projects WHERE id = ? AND status <> 'removed'`, id)
	p, err := scanProject(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return p, err
}

func (r *ProjectRepo) List(ctx context.Context) ([]domain.Project, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, path, status, last_scanned_at, created_at, updated_at
		   FROM projects WHERE status <> 'removed' ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		p, err := scanProjectRows(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepo) UpdateStatus(ctx context.Context, id int64, status domain.ProjectStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET status=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`, string(status), id)
	return err
}

// MarkRemoved sets a project's status to removed.
// Returns (true, nil) on success, (false, nil) if no row matched (not found or already removed),
// and (false, err) on a real database failure.
func (r *ProjectRepo) MarkRemoved(ctx context.Context, id int64) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE projects SET status='removed',
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		 WHERE id=? AND status <> 'removed'`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *ProjectRepo) UpdateLastScannedAt(ctx context.Context, id int64, t time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE projects SET last_scanned_at=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		t.UTC().Format(time.RFC3339), id)
	return err
}

func scanProject(row *sql.Row) (*domain.Project, error) {
	p := &domain.Project{}
	var lastScanned sql.NullString
	var createdAt, updatedAt string

	err := row.Scan(&p.ID, &p.Name, &p.Path, &p.Status, &lastScanned, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	if lastScanned.Valid {
		t, _ := time.Parse(time.RFC3339, lastScanned.String)
		p.LastScannedAt = &t
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return p, nil
}

func scanProjectRows(rows *sql.Rows) (domain.Project, error) {
	var p domain.Project
	var lastScanned sql.NullString
	var createdAt, updatedAt string

	err := rows.Scan(&p.ID, &p.Name, &p.Path, &p.Status, &lastScanned, &createdAt, &updatedAt)
	if err != nil {
		return p, err
	}
	if lastScanned.Valid {
		t, _ := time.Parse(time.RFC3339, lastScanned.String)
		p.LastScannedAt = &t
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	p.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return p, nil
}
