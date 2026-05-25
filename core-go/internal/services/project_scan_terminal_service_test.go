package services

import (
	"context"
	"errors"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
)

// --- test constructor ---

func newProjectScanSvc(
	projRepo *mockProjectRepo,
	fs *mockProjectFS,
	runner *mockRunner,
	scanRepo *mockProjectScanCommitter,
) *ProjectService {
	return NewProjectService(
		projRepo,
		&mockProjectProviderRepo{byProject: make(map[int64][]domain.ProjectProviderSummary)},
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		fs,
	).WithScanDeps(runner, scanRepo)
}

// --- ScanProject boundary tests ---

func TestScanProject_ProjectNotFound_ReturnsValidationError(t *testing.T) {
	svc := newProjectScanSvc(newMockProjectRepo(), &mockProjectFS{}, &mockRunner{}, &mockProjectScanCommitter{})
	_, err := svc.ScanProject(context.Background(), 999)
	requireAppError(t, err, domain.CodeValidation)
}

func TestScanProject_ReturnsOperationID(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj", "/tmp/proj") //nolint:errcheck

	runner := &mockRunner{
		startFn: func(_ context.Context, _ operations.Target, _ domain.OperationType, _ operations.WorkFn) (int64, error) {
			return 42, nil
		},
	}
	svc := newProjectScanSvc(projRepo, &mockProjectFS{}, runner, &mockProjectScanCommitter{})

	opID, err := svc.ScanProject(ctx, 1)
	if err != nil {
		t.Fatalf("ScanProject: %v", err)
	}
	if opID != 42 {
		t.Errorf("opID: got %d want 42", opID)
	}
}

func TestScanProject_RunnerConflictError_PassedThrough(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj", "/tmp/proj") //nolint:errcheck

	runner := &mockRunner{
		startFn: func(_ context.Context, _ operations.Target, _ domain.OperationType, _ operations.WorkFn) (int64, error) {
			return 0, domain.NewConflictError("scan already running", "target locked")
		},
	}
	svc := newProjectScanSvc(projRepo, &mockProjectFS{}, runner, &mockProjectScanCommitter{})

	_, err := svc.ScanProject(ctx, 1)
	requireAppError(t, err, domain.CodeConflict)
}

func TestScanProject_RunnerRawError_WrappedAsDatabaseError(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj", "/tmp/proj") //nolint:errcheck

	runner := &mockRunner{
		startFn: func(_ context.Context, _ operations.Target, _ domain.OperationType, _ operations.WorkFn) (int64, error) {
			return 0, errors.New("connection pool exhausted")
		},
	}
	svc := newProjectScanSvc(projRepo, &mockProjectFS{}, runner, &mockProjectScanCommitter{})

	_, err := svc.ScanProject(ctx, 1)
	requireAppError(t, err, domain.CodeDatabase)
}

// --- scanProjectInternal terminal-path tests ---

func TestScanProjectInternal_ValidPath_NoCommitCalled(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj", "/tmp/proj") //nolint:errcheck

	scanRepo := &mockProjectScanCommitter{}
	svc := newProjectScanSvc(projRepo, &mockProjectFS{validateErr: nil}, &mockRunner{}, scanRepo)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	if scanRepo.terminalCallCount != 0 {
		t.Errorf("expected CommitProjectTerminal not called for valid path, got %d calls", scanRepo.terminalCallCount)
	}
}

func TestScanProjectInternal_MissingPath_CommitsTerminalMissing(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj", "/tmp/proj") //nolint:errcheck

	fs := &mockProjectFS{validateErr: &filesystem.FilesystemError{
		Code: filesystem.ErrPathNotFound, Path: "/tmp/proj", Message: "not found",
	}}
	scanRepo := &mockProjectScanCommitter{}
	svc := newProjectScanSvc(projRepo, fs, &mockRunner{}, scanRepo)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	if scanRepo.terminalCallCount != 1 {
		t.Fatalf("expected 1 CommitProjectTerminal call, got %d", scanRepo.terminalCallCount)
	}
	if scanRepo.lastTerminalStatus != domain.ProjectStatusMissing {
		t.Errorf("status: got %q want missing", scanRepo.lastTerminalStatus)
	}
	if scanRepo.lastTerminalProjectID != 1 {
		t.Errorf("projectID: got %d want 1", scanRepo.lastTerminalProjectID)
	}
}

func TestScanProjectInternal_UnreadablePath_CommitsTerminalUnreadable(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj", "/tmp/proj") //nolint:errcheck

	fs := &mockProjectFS{validateErr: &filesystem.FilesystemError{
		Code: filesystem.ErrPermission, Path: "/tmp/proj", Message: "permission denied",
	}}
	scanRepo := &mockProjectScanCommitter{}
	svc := newProjectScanSvc(projRepo, fs, &mockRunner{}, scanRepo)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	if err != nil {
		t.Fatalf("scanProjectInternal: %v", err)
	}
	if scanRepo.lastTerminalStatus != domain.ProjectStatusUnreadable {
		t.Errorf("status: got %q want unreadable", scanRepo.lastTerminalStatus)
	}
}

func TestScanProjectInternal_MissingPath_WarningIsProjectScoped(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj", "/tmp/proj") //nolint:errcheck

	fs := &mockProjectFS{validateErr: &filesystem.FilesystemError{
		Code: filesystem.ErrPathNotFound, Path: "/tmp/proj", Message: "not found",
	}}
	scanRepo := &mockProjectScanCommitter{}
	svc := newProjectScanSvc(projRepo, fs, &mockRunner{}, scanRepo)

	project, _ := projRepo.GetByID(ctx, 1)
	svc.scanProjectInternal(ctx, project, func(string, int, int, string) {}) //nolint:errcheck

	w := scanRepo.lastTerminalWarning
	if w == nil {
		t.Fatal("expected a warning to be committed with terminal path")
	}
	if w.ScopeType != domain.WarningScopeProject {
		t.Errorf("warning scope: got %q want project", w.ScopeType)
	}
	if w.ActionKey == nil || *w.ActionKey != "rescan" {
		t.Errorf("warning actionKey: got %v want rescan", w.ActionKey)
	}
}

func TestScanProjectInternal_CommitTerminalError_ReturnsDatabaseError(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj", "/tmp/proj") //nolint:errcheck

	fs := &mockProjectFS{validateErr: &filesystem.FilesystemError{
		Code: filesystem.ErrPathNotFound, Path: "/tmp/proj", Message: "not found",
	}}
	scanRepo := &mockProjectScanCommitter{terminalErr: errors.New("db full")}
	svc := newProjectScanSvc(projRepo, fs, &mockRunner{}, scanRepo)

	project, _ := projRepo.GetByID(ctx, 1)
	_, err := svc.scanProjectInternal(ctx, project, func(string, int, int, string) {})
	requireAppError(t, err, domain.CodeDatabase)
}
