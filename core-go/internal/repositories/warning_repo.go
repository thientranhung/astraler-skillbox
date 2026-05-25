package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type WarningRepo struct {
	db *sql.DB
}

func NewWarningRepo(db *sql.DB) *WarningRepo {
	return &WarningRepo{db: db}
}

func (r *WarningRepo) Insert(ctx context.Context, w domain.Warning) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO warnings (scope_type, scope_id, severity, code, message, action_key, source_operation_id, is_resolved)
		 VALUES (?, ?, ?, ?, ?, ?, ?, 0)`,
		string(w.ScopeType), ptrToSQL(w.ScopeID), string(w.Severity),
		w.Code, w.Message, w.ActionKey, ptrToSQL(w.SourceOperationID))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListByScope returns warnings for the given scope; if includeResolved is
// false only active (is_resolved=0) warnings are returned.
func (r *WarningRepo) ListByScope(ctx context.Context, scopeType domain.WarningScopeType, scopeID int64, includeResolved bool) ([]domain.Warning, error) {
	q := `SELECT id, scope_type, scope_id, severity, code, message, action_key,
		        source_operation_id, is_resolved, created_at, updated_at, resolved_at
		   FROM warnings WHERE scope_type=? AND scope_id=?`
	if !includeResolved {
		q += ` AND is_resolved=0`
	}
	q += ` ORDER BY id`

	rows, err := r.db.QueryContext(ctx, q, string(scopeType), scopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var warnings []domain.Warning
	for rows.Next() {
		w, err := scanWarning(rows)
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, w)
	}
	return warnings, rows.Err()
}

// CountActiveForProject returns the count of active warnings for a project across
// all three scopes: project, project_provider, and install.
func (r *WarningRepo) CountActiveForProject(ctx context.Context, projectID int64) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM warnings
		WHERE is_resolved = 0 AND (
			(scope_type = 'project' AND scope_id = ?)
			OR (scope_type = 'project_provider' AND scope_id IN (
				SELECT id FROM project_providers WHERE project_id = ?
			))
			OR (scope_type = 'install' AND scope_id IN (
				SELECT i.id FROM installs i
				JOIN project_providers pp ON pp.id = i.project_provider_id
				WHERE pp.project_id = ?
			))
		)`, projectID, projectID, projectID).Scan(&count)
	return count, err
}

// ListActiveForProject returns all active warnings for a project across
// project, project_provider, and install scopes, ordered by id.
func (r *WarningRepo) ListActiveForProject(ctx context.Context, projectID int64) ([]domain.Warning, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, scope_type, scope_id, severity, code, message, action_key,
		       source_operation_id, is_resolved, created_at, updated_at, resolved_at
		  FROM warnings
		 WHERE is_resolved = 0 AND (
			(scope_type = 'project' AND scope_id = ?)
			OR (scope_type = 'project_provider' AND scope_id IN (
				SELECT id FROM project_providers WHERE project_id = ?
			))
			OR (scope_type = 'install' AND scope_id IN (
				SELECT i.id FROM installs i
				JOIN project_providers pp ON pp.id = i.project_provider_id
				WHERE pp.project_id = ?
			))
		 )
		 ORDER BY id`, projectID, projectID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var warnings []domain.Warning
	for rows.Next() {
		w, err := scanWarning(rows)
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, w)
	}
	return warnings, rows.Err()
}

// ClearByScope marks all active warnings for the scope as resolved.
func (r *WarningRepo) ClearByScope(ctx context.Context, scopeType domain.WarningScopeType, scopeID int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE warnings SET is_resolved=1,
		 resolved_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now')
		 WHERE scope_type=? AND scope_id=? AND is_resolved=0`,
		string(scopeType), scopeID)
	return err
}

// activeWarningPredicate is the SQL WHERE clause that filters out resolved
// warnings and warnings belonging to removed projects (across project,
// project_provider, and install scopes).
const activeWarningPredicate = `
	is_resolved = 0
	AND NOT (
		  (scope_type = 'project'
		     AND scope_id IN (SELECT id FROM projects WHERE status = 'removed'))
		OR (scope_type = 'project_provider'
		     AND scope_id IN (
		          SELECT pp.id FROM project_providers pp
		          JOIN projects p ON p.id = pp.project_id
		          WHERE p.status = 'removed'))
		OR (scope_type = 'install'
		     AND scope_id IN (
		          SELECT i.id FROM installs i
		          JOIN project_providers pp ON pp.id = i.project_provider_id
		          JOIN projects p ON p.id = pp.project_id
		          WHERE p.status = 'removed'))
	)`

// CountActiveBySeverity returns counts of active non-removed-project warnings
// grouped by severity. Unrecognized severity values are silently ignored.
func (r *WarningRepo) CountActiveBySeverity(ctx context.Context) (domain.WarningSeverityCounts, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT severity, COUNT(*) FROM warnings WHERE `+activeWarningPredicate+` GROUP BY severity`)
	if err != nil {
		return domain.WarningSeverityCounts{}, err
	}
	defer rows.Close()

	var counts domain.WarningSeverityCounts
	for rows.Next() {
		var sev string
		var n int
		if err := rows.Scan(&sev, &n); err != nil {
			return domain.WarningSeverityCounts{}, err
		}
		switch domain.WarningSeverity(sev) {
		case domain.WarningSeverityInfo:
			counts.Info += n
		case domain.WarningSeverityWarning:
			counts.Warning += n
		case domain.WarningSeverityError:
			counts.Error += n
		case domain.WarningSeverityBlocking:
			counts.Blocking += n
		}
	}
	return counts, rows.Err()
}

// ListActive returns up to limit active non-removed-project warnings with
// recognized severities, ordered by id DESC. Unrecognized severities are
// excluded to preserve the outbound dashboard contract.
// Returns nil (not an error) on empty result.
func (r *WarningRepo) ListActive(ctx context.Context, limit int) ([]domain.Warning, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, scope_type, scope_id, severity, code, message, action_key,
		        source_operation_id, is_resolved, created_at, updated_at, resolved_at
		   FROM warnings
		  WHERE `+activeWarningPredicate+`
		    AND severity IN ('info', 'warning', 'error', 'blocking')
		  ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var warnings []domain.Warning
	for rows.Next() {
		w, err := scanWarning(rows)
		if err != nil {
			return nil, err
		}
		warnings = append(warnings, w)
	}
	return warnings, rows.Err()
}

func scanWarning(rows *sql.Rows) (domain.Warning, error) {
	var w domain.Warning
	var scopeID, sourceOpID sql.NullInt64
	var actionKey, resolvedAt sql.NullString
	var isResolved int
	var createdAt, updatedAt string

	err := rows.Scan(&w.ID, &w.ScopeType, &scopeID, &w.Severity, &w.Code, &w.Message,
		&actionKey, &sourceOpID, &isResolved, &createdAt, &updatedAt, &resolvedAt)
	if err != nil {
		return w, err
	}
	if scopeID.Valid {
		id := scopeID.Int64
		w.ScopeID = &id
	}
	if actionKey.Valid {
		w.ActionKey = &actionKey.String
	}
	if sourceOpID.Valid {
		id := sourceOpID.Int64
		w.SourceOperationID = &id
	}
	w.IsResolved = isResolved != 0
	if resolvedAt.Valid {
		t, _ := time.Parse(time.RFC3339, resolvedAt.String)
		w.ResolvedAt = &t
	}
	w.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	w.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return w, nil
}
