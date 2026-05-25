package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type SkillRepo struct {
	db *sql.DB
}

func NewSkillRepo(db *sql.DB) *SkillRepo {
	return &SkillRepo{db: db}
}

// UpsertMany inserts or updates skills for a given host in a single transaction.
func (r *SkillRepo) UpsertMany(ctx context.Context, hostID int64, skills []domain.Skill) error {
	if len(skills) == 0 {
		return nil
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, s := range skills {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO skills (skill_host_folder_id, name, relative_path, absolute_path, status, last_scanned_at)
			 VALUES (?, ?, ?, ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ','now'))
			 ON CONFLICT(skill_host_folder_id, relative_path)
			 DO UPDATE SET name=excluded.name, absolute_path=excluded.absolute_path,
			               status=excluded.status,
			               last_scanned_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),
			               updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
			hostID, s.Name, s.RelativePath, s.AbsolutePath, string(s.Status))
		if err != nil {
			return fmt.Errorf("upsert skill %q: %w", s.Name, err)
		}
	}
	return tx.Commit()
}

// MarkMissing sets status='missing' for all skills of hostID whose IDs are
// not in presentIDs (i.e., they were not found during the last scan).
func (r *SkillRepo) MarkMissing(ctx context.Context, hostID int64, presentIDs []int64) error {
	if len(presentIDs) == 0 {
		// All skills are missing.
		_, err := r.db.ExecContext(ctx,
			`UPDATE skills SET status='missing', updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
			  WHERE skill_host_folder_id=?`, hostID)
		return err
	}

	// Build placeholders.
	ph := make([]string, len(presentIDs))
	args := make([]interface{}, 0, len(presentIDs)+1)
	args = append(args, hostID)
	for i, id := range presentIDs {
		ph[i] = "?"
		args = append(args, id)
	}

	q := fmt.Sprintf(
		"UPDATE skills SET status='missing', updated_at=strftime('%%Y-%%m-%%dT%%H:%%M:%%SZ','now')"+
			" WHERE skill_host_folder_id=? AND id NOT IN (%s)",
		strings.Join(ph, ","))
	_, err := r.db.ExecContext(ctx, q, args...)
	return err
}

// ListByHost returns all skills for the given host ordered by name.
func (r *SkillRepo) ListByHost(ctx context.Context, hostID int64) ([]domain.Skill, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, skill_host_folder_id, name, display_name, relative_path, absolute_path,
		        status, source_id, last_scanned_at, created_at, updated_at
		   FROM skills WHERE skill_host_folder_id=? ORDER BY name`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []domain.Skill
	for rows.Next() {
		s, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		skills = append(skills, s)
	}
	return skills, rows.Err()
}

// ListIDsByHost returns only the IDs of skills for the given host.
func (r *SkillRepo) ListIDsByHost(ctx context.Context, hostID int64) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id FROM skills WHERE skill_host_folder_id=?`, hostID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CountByHost returns the total number of skills for the given host.
func (r *SkillRepo) CountByHost(ctx context.Context, hostID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM skills WHERE skill_host_folder_id=?`, hostID).Scan(&count)
	return count, err
}

func scanSkill(rows *sql.Rows) (domain.Skill, error) {
	var s domain.Skill
	var displayName, lastScanned, createdAt, updatedAt sql.NullString
	var sourceID sql.NullInt64

	err := rows.Scan(&s.ID, &s.SkillHostFolderID, &s.Name, &displayName,
		&s.RelativePath, &s.AbsolutePath, &s.Status, &sourceID,
		&lastScanned, &createdAt, &updatedAt)
	if err != nil {
		return s, err
	}
	if displayName.Valid {
		s.DisplayName = &displayName.String
	}
	if sourceID.Valid {
		id := sourceID.Int64
		s.SourceID = &id
	}
	if lastScanned.Valid {
		t, _ := time.Parse(time.RFC3339, lastScanned.String)
		s.LastScannedAt = &t
	}
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt.String)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt.String)
	return s, nil
}
