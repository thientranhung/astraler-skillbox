package operations

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// WorkFn is the function type executed by the runner inside a goroutine.
// It receives a cancellable context and a progress callback.
// On success it returns optional metadata (marshallable to JSON) and nil error.
type WorkFn func(ctx context.Context, progress ProgressFn) (metadata any, err error)

// OperationRepo is the minimal repo interface the runner depends on.
type OperationRepo interface {
	Insert(ctx context.Context, targetType string, targetID *int64, opType domain.OperationType) (int64, error)
	MarkStarted(ctx context.Context, id int64) error
	UpdateStatus(ctx context.Context, id int64, status domain.OperationStatus, errMsg *string, metadataJSON *string, finishedAt *time.Time) error
	// GetByID returns the operation, or (nil, nil) if not found.
	GetByID(ctx context.Context, id int64) (*domain.Operation, error)
}

// Runner manages the lifecycle of long-running operations.
type Runner struct {
	repo        OperationRepo
	mu          sync.Mutex
	activeLocks map[string]struct{} // per-target lock keys
	cancelFns   map[int64]context.CancelFunc
	progressCh  chan ProgressEvent
}

// NewRunner creates a Runner. progressCh receives all progress events;
// pass a buffered channel sized to your fan-out needs.
func NewRunner(repo OperationRepo, progressCh chan ProgressEvent) *Runner {
	return &Runner{
		repo:        repo,
		activeLocks: make(map[string]struct{}),
		cancelFns:   make(map[int64]context.CancelFunc),
		progressCh:  progressCh,
	}
}

// Start creates an operation record, acquires the per-target lock,
// and launches the work function in a goroutine.
// Returns the operation ID immediately or a conflict_error if the target is busy.
func (r *Runner) Start(ctx context.Context, target Target, opType domain.OperationType, fn WorkFn) (int64, error) {
	lockKey := target.String()

	r.mu.Lock()
	if _, busy := r.activeLocks[lockKey]; busy {
		r.mu.Unlock()
		return 0, domain.NewConflictError(
			fmt.Sprintf("%s is already running an operation", target.Type),
			fmt.Sprintf("target %s is locked", lockKey),
		)
	}
	r.activeLocks[lockKey] = struct{}{}
	r.mu.Unlock()

	targetID := target.ID
	opID, err := r.repo.Insert(ctx, target.Type, &targetID, opType)
	if err != nil {
		r.mu.Lock()
		delete(r.activeLocks, lockKey)
		r.mu.Unlock()
		return 0, err
	}

	opCtx, cancel := context.WithCancel(context.Background())
	r.mu.Lock()
	r.cancelFns[opID] = cancel
	r.mu.Unlock()

	go r.run(opCtx, opID, lockKey, fn)

	return opID, nil
}

func (r *Runner) run(ctx context.Context, opID int64, lockKey string, fn WorkFn) {
	defer func() {
		if rec := recover(); rec != nil {
			slog.Error("operation panicked", "operationId", opID, "panic", rec)
			errMsg := fmt.Sprintf("panic: %v", rec)
			now := time.Now()
			_ = r.repo.UpdateStatus(context.Background(), opID,
				domain.OperationStatusFailed, &errMsg, nil, &now)
		}
		r.mu.Lock()
		delete(r.activeLocks, lockKey)
		delete(r.cancelFns, opID)
		r.mu.Unlock()
	}()

	_ = r.repo.MarkStarted(ctx, opID)
	r.emit(opID, "running", "", nil, nil, nil)

	progressFn := func(phase string, processed, total int, msg string) {
		r.emit(opID, "running", phase, &processed, &total, &msg)
	}

	meta, err := fn(ctx, progressFn)

	// Marshal metadata once; used on both success and failure paths so that
	// partial-failure operations (returning metadata AND a non-nil error) still
	// persist their progress summary.
	var metaStr *string
	if meta != nil {
		if b, jerr := json.Marshal(meta); jerr == nil {
			s := string(b)
			metaStr = &s
		} else {
			slog.Error("operation metadata marshal failed", "operationId", opID, "error", jerr)
		}
	}

	now := time.Now()
	if err != nil {
		errMsg := err.Error()
		var status domain.OperationStatus
		if ctx.Err() != nil {
			status = domain.OperationStatusCancelled
		} else {
			status = domain.OperationStatusFailed
		}
		_ = r.repo.UpdateStatus(context.Background(), opID, status, &errMsg, metaStr, &now)
		r.emit(opID, string(status), "done", nil, nil, &errMsg)
		return
	}

	_ = r.repo.UpdateStatus(context.Background(), opID, domain.OperationStatusSuccess, nil, metaStr, &now)
	r.emit(opID, "success", "done", nil, nil, nil)
}

func (r *Runner) emit(opID int64, status, phase string, processed, total *int, msg *string) {
	if r.progressCh == nil {
		return
	}
	select {
	case r.progressCh <- ProgressEvent{
		OperationID: opID,
		Status:      status,
		Phase:       phase,
		Processed:   processed,
		Total:       total,
		Message:     msg,
	}:
	default:
		// Never block the operation goroutine.
	}
}

// Cancel sends a cancellation signal to the operation.
// Returns (true, nil) if the signal was sent to a running operation.
// Returns (false, nil) if the operation already finished.
// Returns (false, validation_error) if operationID does not exist.
// Returns (false, database_error) on a DB lookup failure.
func (r *Runner) Cancel(ctx context.Context, operationID int64) (bool, error) {
	r.mu.Lock()
	cancel, ok := r.cancelFns[operationID]
	r.mu.Unlock()

	if ok {
		cancel()
		return true, nil
	}

	// Not active in memory — check the DB to distinguish "finished" vs "never existed".
	op, err := r.repo.GetByID(ctx, operationID)
	if err != nil {
		return false, domain.NewDatabaseError("Could not check operation status", err.Error())
	}
	if op == nil {
		return false, domain.NewValidationError(
			"Operation not found",
			fmt.Sprintf("operationId %d does not exist", operationID),
		)
	}
	return false, nil
}

// MarkAllRunningAsFailed cancels all in-memory operations and marks them failed in the DB.
// Call during graceful shutdown; pair with OperationRepo.MarkStaleAsFailed for full coverage.
func (r *Runner) MarkAllRunningAsFailed(reason string) {
	r.mu.Lock()
	ids := make([]int64, 0, len(r.cancelFns))
	for id, cancel := range r.cancelFns {
		cancel()
		ids = append(ids, id)
	}
	r.mu.Unlock()

	now := time.Now()
	for _, id := range ids {
		_ = r.repo.UpdateStatus(context.Background(), id,
			domain.OperationStatusFailed, &reason, nil, &now)
	}
}
