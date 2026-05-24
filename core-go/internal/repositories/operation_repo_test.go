package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

func TestOperationRepo_InsertAndGet(t *testing.T) {
	db := NewTestDB(t)
	repo := NewOperationRepo(db)
	ctx := context.Background()

	hostID := int64(42)
	id, err := repo.Insert(ctx, "skill_host_folder", &hostID, domain.OperationTypeScan)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive id, got %d", id)
	}

	op, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if op.Status != domain.OperationStatusQueued {
		t.Errorf("status: got %q want queued", op.Status)
	}
	if op.TargetID == nil || *op.TargetID != hostID {
		t.Errorf("targetID: got %v want %d", op.TargetID, hostID)
	}
}

func TestOperationRepo_UpdateStatus(t *testing.T) {
	db := NewTestDB(t)
	repo := NewOperationRepo(db)
	ctx := context.Background()

	id, _ := repo.Insert(ctx, "skill_host_folder", nil, domain.OperationTypeScan)
	_ = repo.MarkStarted(ctx, id)

	now := time.Now()
	meta := `{"skillsFound":3}`
	if err := repo.UpdateStatus(ctx, id, domain.OperationStatusSuccess, nil, &meta, &now); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	op, _ := repo.GetByID(ctx, id)
	if op.Status != domain.OperationStatusSuccess {
		t.Errorf("status: got %q", op.Status)
	}
	if op.MetadataJSON == nil || *op.MetadataJSON != meta {
		t.Errorf("metadata: got %v", op.MetadataJSON)
	}
}

func TestOperationRepo_ListActiveByTarget(t *testing.T) {
	db := NewTestDB(t)
	repo := NewOperationRepo(db)
	ctx := context.Background()

	hostID := int64(1)
	id, _ := repo.Insert(ctx, "skill_host_folder", &hostID, domain.OperationTypeScan)
	_ = repo.MarkStarted(ctx, id)

	ops, err := repo.ListActiveByTarget(ctx, "skill_host_folder", hostID)
	if err != nil {
		t.Fatalf("ListActiveByTarget: %v", err)
	}
	if len(ops) != 1 {
		t.Fatalf("expected 1 active op, got %d", len(ops))
	}

	// Finish the op; list should be empty.
	_ = repo.UpdateStatus(ctx, id, domain.OperationStatusSuccess, nil, nil, nil)
	ops2, _ := repo.ListActiveByTarget(ctx, "skill_host_folder", hostID)
	if len(ops2) != 0 {
		t.Fatalf("expected 0 after finish, got %d", len(ops2))
	}
}
