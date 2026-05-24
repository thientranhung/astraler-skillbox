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
