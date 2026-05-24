package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// ScanRepo provides an atomic write path for host scan results.
type ScanRepo struct {
	db *sql.DB
}

func NewScanRepo(db *sql.DB) *ScanRepo {
	return &ScanRepo{db: db}
}

// CommitScanResults persists all scan results for a host in a single transaction:
// upserts found skills, marks missing skills, updates host.last_scanned_at,
// clears existing host-scope warnings, and inserts new warnings.
func (r *ScanRepo) CommitScanResults(
	ctx context.Context,
	hostID int64,
	skills []domain.Skill,
	warnings []domain.Warning,
	now time.Time,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin scan tx: %w", err)
	}
	defer tx.Rollback()

	nowStr := now.UTC().Format(time.RFC3339)

	// 1. Upsert found skills.
	foundPaths := make([]string, 0, len(skills))
	for _, s := range skills {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO skills
			  (skill_host_folder_id, name, relative_path, absolute_path, status, last_scanned_at)
			VALUES (?, ?, ?, ?, ?, ?)
			ON CONFLICT(skill_host_folder_id, relative_path)
			DO UPDATE SET
			  name=excluded.name,
			  absolute_path=excluded.absolute_path,
			  status=excluded.status,
			  last_scanned_at=excluded.last_scanned_at,
			  updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')`,
			hostID, s.Name, s.RelativePath, s.AbsolutePath, string(s.Status), nowStr,
		); err != nil {
			return fmt.Errorf("upsert skill %q: %w", s.Name, err)
		}
		foundPaths = append(foundPaths, s.RelativePath)
	}

	// 2. Mark absent skills as missing.
	if err := markMissingByPath(ctx, tx, hostID, foundPaths); err != nil {
		return fmt.Errorf("mark missing: %w", err)
	}

	// 3. Update host last_scanned_at.
	if _, err := tx.ExecContext(ctx,
		`UPDATE skill_host_folders SET last_scanned_at=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		nowStr, hostID,
	); err != nil {
		return fmt.Errorf("update host timestamp: %w", err)
	}

	// 4. Clear existing host-scope warnings.
	if _, err := tx.ExecContext(ctx,
		`UPDATE warnings SET is_resolved=1,
		 resolved_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		 WHERE scope_type='skill_host_folder' AND scope_id=? AND is_resolved=0`,
		hostID,
	); err != nil {
		return fmt.Errorf("clear warnings: %w", err)
	}

	// 5. Insert new warnings.
	for _, w := range warnings {
		var actionKey interface{}
		if w.ActionKey != nil {
			actionKey = *w.ActionKey
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO warnings
			  (scope_type, scope_id, severity, code, message, action_key, source_operation_id, is_resolved)
			 VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
			string(w.ScopeType), ptrToSQL(w.ScopeID),
			string(w.Severity), w.Code, w.Message,
			actionKey, ptrToSQL(w.SourceOperationID),
		); err != nil {
			return fmt.Errorf("insert warning %q: %w", w.Code, err)
		}
	}

	return tx.Commit()
}

func markMissingByPath(ctx context.Context, tx *sql.Tx, hostID int64, foundPaths []string) error {
	if len(foundPaths) == 0 {
		_, err := tx.ExecContext(ctx,
			`UPDATE skills SET status='missing',
			 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
			 WHERE skill_host_folder_id=?`, hostID)
		return err
	}
	ph := strings.Repeat("?,", len(foundPaths))
	ph = ph[:len(ph)-1]
	args := make([]interface{}, 0, 1+len(foundPaths))
	args = append(args, hostID)
	for _, p := range foundPaths {
		args = append(args, p)
	}
	_, err := tx.ExecContext(ctx,
		"UPDATE skills SET status='missing',"+
			" updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')"+
			" WHERE skill_host_folder_id=? AND relative_path NOT IN ("+ph+")",
		args...)
	return err
}
