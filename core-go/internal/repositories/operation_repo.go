package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type OperationRepo struct {
	db *sql.DB
}

func NewOperationRepo(db *sql.DB) *OperationRepo {
	return &OperationRepo{db: db}
}

func (r *OperationRepo) Insert(ctx context.Context, targetType string, targetID *int64, opType domain.OperationType) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO operations (operation_type, target_type, target_id, status)
		 VALUES (?, ?, ?, 'queued')`, string(opType), targetType, ptrToSQL(targetID))
	if err != nil {
		return 0, fmt.Errorf("insert operation: %w", err)
	}
	return res.LastInsertId()
}

func (r *OperationRepo) UpdateStatus(ctx context.Context, id int64, status domain.OperationStatus, errMsg *string, metadataJSON *string, finishedAt *time.Time) error {
	var finAt interface{}
	if finishedAt != nil {
		finAt = finishedAt.UTC().Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE operations SET status=?, error_message=?, metadata_json=?, finished_at=?,
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`,
		string(status), errMsg, metadataJSON, finAt, id)
	return err
}

func (r *OperationRepo) MarkStarted(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE operations SET status='running',
		 started_at=strftime('%Y-%m-%dT%H:%M:%SZ','now'),
		 updated_at=strftime('%Y-%m-%dT%H:%M:%SZ','now') WHERE id=?`, id)
	return err
}

func (r *OperationRepo) GetByID(ctx context.Context, id int64) (*domain.Operation, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, operation_type, target_type, target_id, status,
		        started_at, finished_at, error_message, metadata_json,
		        created_at, updated_at
		   FROM operations WHERE id=?`, id)
	return scanOperation(row)
}

func (r *OperationRepo) ListActiveByTarget(ctx context.Context, targetType string, targetID int64) ([]domain.Operation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, operation_type, target_type, target_id, status,
		        started_at, finished_at, error_message, metadata_json,
		        created_at, updated_at
		   FROM operations
		  WHERE target_type=? AND target_id=? AND status IN ('queued','running')`,
		targetType, targetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ops []domain.Operation
	for rows.Next() {
		op, err := scanOperationRows(rows)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, rows.Err()
}

func ptrToSQL(p *int64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func scanOperation(row *sql.Row) (*domain.Operation, error) {
	var op domain.Operation
	var startedAt, finishedAt, errMsg, metaJSON sql.NullString
	var targetID sql.NullInt64
	var createdAt, updatedAt string

	err := row.Scan(&op.ID, &op.OperationType, &op.TargetType, &targetID, &op.Status,
		&startedAt, &finishedAt, &errMsg, &metaJSON, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	fillOperation(&op, targetID, startedAt, finishedAt, errMsg, metaJSON, createdAt, updatedAt)
	return &op, nil
}

func scanOperationRows(rows *sql.Rows) (domain.Operation, error) {
	var op domain.Operation
	var startedAt, finishedAt, errMsg, metaJSON sql.NullString
	var targetID sql.NullInt64
	var createdAt, updatedAt string

	err := rows.Scan(&op.ID, &op.OperationType, &op.TargetType, &targetID, &op.Status,
		&startedAt, &finishedAt, &errMsg, &metaJSON, &createdAt, &updatedAt)
	if err != nil {
		return op, err
	}
	fillOperation(&op, targetID, startedAt, finishedAt, errMsg, metaJSON, createdAt, updatedAt)
	return op, nil
}

func fillOperation(op *domain.Operation, targetID sql.NullInt64, startedAt, finishedAt, errMsg, metaJSON sql.NullString, createdAt, updatedAt string) {
	if targetID.Valid {
		id := targetID.Int64
		op.TargetID = &id
	}
	if startedAt.Valid {
		t, _ := time.Parse(time.RFC3339, startedAt.String)
		op.StartedAt = &t
	}
	if finishedAt.Valid {
		t, _ := time.Parse(time.RFC3339, finishedAt.String)
		op.FinishedAt = &t
	}
	if errMsg.Valid {
		op.ErrorMessage = &errMsg.String
	}
	if metaJSON.Valid {
		op.MetadataJSON = &metaJSON.String
	}
	op.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	op.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
}
