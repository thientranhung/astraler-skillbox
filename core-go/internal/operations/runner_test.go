package operations

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// -- fake repo --

type fakeOpRepo struct {
	mu     sync.Mutex
	ops    map[int64]*fakeOp
	nextID int64
}

type fakeOp struct {
	status       domain.OperationStatus
	errMsg       *string
	metadataJSON *string
}

func newFakeRepo() *fakeOpRepo {
	return &fakeOpRepo{ops: make(map[int64]*fakeOp), nextID: 1}
}

func (f *fakeOpRepo) Insert(_ context.Context, _ string, _ *int64, _ domain.OperationType) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id := f.nextID
	f.nextID++
	f.ops[id] = &fakeOp{status: domain.OperationStatusQueued}
	return id, nil
}

func (f *fakeOpRepo) MarkStarted(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if op, ok := f.ops[id]; ok {
		op.status = domain.OperationStatusRunning
	}
	return nil
}

func (f *fakeOpRepo) UpdateStatus(_ context.Context, id int64, status domain.OperationStatus, errMsg *string, metaJSON *string, _ *time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if op, ok := f.ops[id]; ok {
		op.status = status
		op.errMsg = errMsg
		op.metadataJSON = metaJSON
	}
	return nil
}

// GetByID returns (nil, nil) when not found — matches the repositories contract.
func (f *fakeOpRepo) GetByID(_ context.Context, id int64) (*domain.Operation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	op, ok := f.ops[id]
	if !ok {
		return nil, nil
	}
	return &domain.Operation{ID: id, Status: op.status}, nil
}

func (f *fakeOpRepo) getStatus(id int64) domain.OperationStatus {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ops[id].status
}

func (f *fakeOpRepo) getMeta(id int64) *string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ops[id].metadataJSON
}

// -- helpers --

func newRunner() (*Runner, chan ProgressEvent) {
	ch := make(chan ProgressEvent, 32)
	repo := newFakeRepo()
	return NewRunner(repo, ch), ch
}

func waitStatus(t *testing.T, repo *fakeOpRepo, id int64, want domain.OperationStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if repo.getStatus(id) == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for status %q; got %q", want, repo.getStatus(id))
}

// -- tests --

func TestRunner_HappyPath(t *testing.T) {
	r, _ := newRunner()
	repo := r.repo.(*fakeOpRepo)
	target := Target{Type: "skill_host_folder", ID: 1}

	done := make(chan struct{})
	id, err := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, progress ProgressFn) (any, error) {
			progress("working", 1, 1, "")
			close(done)
			return map[string]int{"found": 3}, nil
		})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	<-done
	waitStatus(t, repo, id, domain.OperationStatusSuccess)

	meta := repo.getMeta(id)
	if meta == nil || *meta != `{"found":3}` {
		t.Errorf("metadata: %v", meta)
	}
}

func TestRunner_FailPath(t *testing.T) {
	r, _ := newRunner()
	repo := r.repo.(*fakeOpRepo)
	target := Target{Type: "skill_host_folder", ID: 1}

	id, err := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			return nil, errors.New("something went wrong")
		})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitStatus(t, repo, id, domain.OperationStatusFailed)
}

func TestRunner_Cancel(t *testing.T) {
	r, _ := newRunner()
	repo := r.repo.(*fakeOpRepo)
	target := Target{Type: "skill_host_folder", ID: 1}

	started := make(chan struct{})
	id, _ := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			close(started)
			<-ctx.Done()
			return nil, ctx.Err()
		})

	<-started
	acked, err := r.Cancel(context.Background(), id)
	if err != nil || !acked {
		t.Errorf("Cancel running op: acked=%v err=%v", acked, err)
	}
	waitStatus(t, repo, id, domain.OperationStatusCancelled)
}

func TestRunner_Cancel_AlreadyFinished(t *testing.T) {
	r, _ := newRunner()
	repo := r.repo.(*fakeOpRepo)
	target := Target{Type: "skill_host_folder", ID: 1}

	done := make(chan struct{})
	id, _ := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			close(done)
			return nil, nil
		})
	<-done
	waitStatus(t, repo, id, domain.OperationStatusSuccess)

	acked, err := r.Cancel(context.Background(), id)
	if err != nil {
		t.Fatalf("Cancel finished op: unexpected error %v", err)
	}
	if acked {
		t.Error("expected acknowledged=false for already-finished operation")
	}
}

func TestRunner_Cancel_Nonexistent(t *testing.T) {
	r, _ := newRunner()
	_, err := r.Cancel(context.Background(), 99999)
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error for nonexistent id, got %v", err)
	}
}

func TestRunner_Lock_ConflictError(t *testing.T) {
	r, _ := newRunner()
	target := Target{Type: "skill_host_folder", ID: 1}

	ready := make(chan struct{})
	hold := make(chan struct{})
	_, err := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			close(ready)
			<-hold
			return nil, nil
		})
	if err != nil {
		t.Fatalf("first Start: %v", err)
	}

	<-ready
	_, err2 := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) { return nil, nil })
	close(hold)

	if err2 == nil {
		t.Fatal("expected conflict_error, got nil")
	}
	ae, ok := err2.(*domain.AppError)
	if !ok || ae.Code != domain.CodeConflict {
		t.Errorf("expected conflict_error, got %v", err2)
	}
}

func TestRunner_PanicRecovery(t *testing.T) {
	r, _ := newRunner()
	repo := r.repo.(*fakeOpRepo)
	target := Target{Type: "skill_host_folder", ID: 1}

	id, _ := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			panic("surprise!")
		})

	waitStatus(t, repo, id, domain.OperationStatusFailed)
}

func TestRunner_LockReleasedAfterCompletion(t *testing.T) {
	r, _ := newRunner()
	target := Target{Type: "skill_host_folder", ID: 1}

	id, _ := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			return nil, nil
		})
	repo := r.repo.(*fakeOpRepo)
	waitStatus(t, repo, id, domain.OperationStatusSuccess)

	done := make(chan struct{})
	id2, err := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			close(done)
			return nil, nil
		})
	if err != nil {
		t.Fatalf("second Start after completion: %v", err)
	}
	<-done
	waitStatus(t, repo, id2, domain.OperationStatusSuccess)
}

// TestRunner_SuccessStoresMetadata verifies that metadata returned alongside a
// nil error is persisted in the SUCCESS path.
func TestRunner_SuccessStoresMetadata(t *testing.T) {
	r, _ := newRunner()
	repo := r.repo.(*fakeOpRepo)
	target := Target{Type: "skill_host_folder", ID: 42}

	type installMeta struct {
		Requested int `json:"requested"`
		Created   int `json:"created"`
		Failed    int `json:"failed"`
	}

	done := make(chan struct{})
	id, err := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			close(done)
			return installMeta{Requested: 2, Created: 2, Failed: 0}, nil
		})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	<-done
	waitStatus(t, repo, id, domain.OperationStatusSuccess)

	meta := repo.getMeta(id)
	if meta == nil {
		t.Fatal("expected metadataJSON to be non-nil on success, got nil")
	}
	want := `{"requested":2,"created":2,"failed":0}`
	if *meta != want {
		t.Errorf("metadataJSON mismatch:\n  got  %s\n  want %s", *meta, want)
	}
}

// TestRunner_FailedPathStoresMetadata verifies that metadata returned alongside
// a non-nil error (partial-failure path, e.g. Slice 2F install) is also
// persisted when the operation is marked FAILED.
func TestRunner_FailedPathStoresMetadata(t *testing.T) {
	r, _ := newRunner()
	repo := r.repo.(*fakeOpRepo)
	target := Target{Type: "project", ID: 7}

	type installMeta struct {
		Requested int `json:"requested"`
		Created   int `json:"created"`
		Failed    int `json:"failed"`
	}

	done := make(chan struct{})
	id, err := r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, _ ProgressFn) (any, error) {
			close(done)
			// Return BOTH metadata and a non-nil error — the partial-failure case.
			return installMeta{Requested: 2, Created: 1, Failed: 1},
				errors.New("filesystem_error: one symlink failed")
		})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	<-done
	waitStatus(t, repo, id, domain.OperationStatusFailed)

	meta := repo.getMeta(id)
	if meta == nil {
		t.Fatal("expected metadataJSON to be non-nil on failure with returned metadata, got nil")
	}
	want := `{"requested":2,"created":1,"failed":1}`
	if *meta != want {
		t.Errorf("metadataJSON mismatch on failed path:\n  got  %s\n  want %s", *meta, want)
	}
}

func TestRunner_ProgressEvents(t *testing.T) {
	r, ch := newRunner()
	target := Target{Type: "skill_host_folder", ID: 1}

	finished := make(chan struct{})
	r.Start(context.Background(), target, domain.OperationTypeScan,
		func(ctx context.Context, progress ProgressFn) (any, error) {
			progress("phase1", 0, 10, "starting")
			progress("phase2", 10, 10, "done")
			close(finished)
			return nil, nil
		})
	<-finished

	time.Sleep(20 * time.Millisecond)
	if len(ch) == 0 {
		t.Fatal("expected at least one progress event")
	}
}
