package services

import (
	"context"
	"errors"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
)

func newHostService(fs *mockFS, hostRepo *mockHostRepo) *SkillHostService {
	return NewSkillHostService(hostRepo, newMockSettings(nil), fs, &mockRunner{}, &mockScanWriter{})
}

func TestChooseHost_Happy(t *testing.T) {
	fs := &mockFS{ensureCreated: true}
	hostRepo := newMockHostRepo()
	svc := newHostService(fs, hostRepo)

	result, err := svc.ChooseHost(context.Background(), "/tmp/myhost")
	if err != nil {
		t.Fatalf("ChooseHost: %v", err)
	}
	if result.HostID <= 0 {
		t.Error("expected positive hostID")
	}
	if !result.Initialized {
		t.Error("expected Initialized=true")
	}
	if result.Status != domain.SkillHostStatusActive {
		t.Errorf("status: got %q want active", result.Status)
	}
}

func TestChooseHost_ValidationError(t *testing.T) {
	fs := &mockFS{validateErr: &filesystem.FilesystemError{
		Code: filesystem.ErrPathNotFound, Path: "/bad", Message: "not found",
	}}
	svc := newHostService(fs, newMockHostRepo())

	_, err := svc.ChooseHost(context.Background(), "/bad")
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestChooseHost_FilesystemError(t *testing.T) {
	fs := &mockFS{ensureErr: errors.New("no write permission")}
	svc := newHostService(fs, newMockHostRepo())

	_, err := svc.ChooseHost(context.Background(), "/tmp/host")
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeFilesystem {
		t.Errorf("expected filesystem_error, got %v", err)
	}
}

func TestChooseHost_Idempotent(t *testing.T) {
	fs := &mockFS{ensureCreated: true}
	hostRepo := newMockHostRepo()
	svc := newHostService(fs, hostRepo)

	r1, err1 := svc.ChooseHost(context.Background(), "/tmp/myhost")
	r2, err2 := svc.ChooseHost(context.Background(), "/tmp/myhost")
	if err1 != nil || err2 != nil {
		t.Fatalf("ChooseHost errors: %v, %v", err1, err2)
	}
	if r1.HostID != r2.HostID {
		t.Errorf("expected same hostID: %d vs %d", r1.HostID, r2.HostID)
	}
}

func TestScanHost_ValidationError_UnknownHost(t *testing.T) {
	svc := NewSkillHostService(newMockHostRepo(), newMockSettings(nil), &mockFS{}, &mockRunner{}, &mockScanWriter{})
	_, err := svc.ScanHost(context.Background(), 99999)
	if err == nil {
		t.Fatal("expected error for unknown hostID")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestScanHost_ReturnsOperationID(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	runner := &mockRunner{
		startFn: func(_ context.Context, _ operations.Target, _ domain.OperationType, _ operations.WorkFn) (int64, error) {
			return 42, nil
		},
	}

	svc := NewSkillHostService(hostRepo, newMockSettings(nil), &mockFS{}, runner, &mockScanWriter{})
	opID, err := svc.ScanHost(ctx, hostID)
	if err != nil {
		t.Fatalf("ScanHost: %v", err)
	}
	if opID != 42 {
		t.Errorf("opID: got %d want 42", opID)
	}
}

func TestScanHost_RunnerRawError_WrappedAsDatabaseError(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	runner := &mockRunner{
		startFn: func(_ context.Context, _ operations.Target, _ domain.OperationType, _ operations.WorkFn) (int64, error) {
			return 0, errors.New("connection pool exhausted")
		},
	}

	svc := NewSkillHostService(hostRepo, newMockSettings(nil), &mockFS{}, runner, &mockScanWriter{})
	_, err := svc.ScanHost(ctx, hostID)
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeDatabase {
		t.Errorf("expected database_error, got %v", err)
	}
}

func TestScanHost_RunnerConflictError_PassedThrough(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	runner := &mockRunner{
		startFn: func(_ context.Context, _ operations.Target, _ domain.OperationType, _ operations.WorkFn) (int64, error) {
			return 0, domain.NewConflictError("scan already running", "target locked")
		},
	}

	svc := NewSkillHostService(hostRepo, newMockSettings(nil), &mockFS{}, runner, &mockScanWriter{})
	_, err := svc.ScanHost(ctx, hostID)
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeConflict {
		t.Errorf("expected conflict_error, got %v", err)
	}
}

func TestScanHostInternal_SkillsPassedToCommitter(t *testing.T) {
	hostRepo := newMockHostRepo()
	scanWriter := &mockScanWriter{}
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	fs := &mockFS{
		scanEntries: []filesystem.HostEntry{
			{Name: "skill-a", RelativePath: "skill-a", AbsolutePath: "/tmp/host/.agents/skills/skill-a", IsDir: true},
		},
	}

	svc := NewSkillHostService(hostRepo, newMockSettings(nil), fs, &mockRunner{}, scanWriter)
	host, _ := hostRepo.GetByID(ctx, hostID)
	_, err := svc.scanHostInternal(ctx, host, func(_ string, _, _ int, _ string) {})
	if err != nil {
		t.Fatalf("scanHostInternal: %v", err)
	}
	if len(scanWriter.skills) != 1 {
		t.Errorf("expected 1 skill committed, got %d", len(scanWriter.skills))
	}
	if scanWriter.skills[0].Name != "skill-a" {
		t.Errorf("skill name: got %q want skill-a", scanWriter.skills[0].Name)
	}
}

func TestScanHostInternal_WarningsScopeIsHost(t *testing.T) {
	hostRepo := newMockHostRepo()
	scanWriter := &mockScanWriter{}
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	fs := &mockFS{
		scanEntries: []filesystem.HostEntry{
			{Name: "broken", RelativePath: "broken", IsSymlink: true, Broken: true},
			{Name: "external", RelativePath: "external", IsSymlink: true, External: true},
		},
	}

	svc := NewSkillHostService(hostRepo, newMockSettings(nil), fs, &mockRunner{}, scanWriter)
	host, _ := hostRepo.GetByID(ctx, hostID)
	_, err := svc.scanHostInternal(ctx, host, func(_ string, _, _ int, _ string) {})
	if err != nil {
		t.Fatalf("scanHostInternal: %v", err)
	}
	if len(scanWriter.warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d", len(scanWriter.warnings))
	}
	for _, w := range scanWriter.warnings {
		if w.ScopeType != domain.WarningScopeSkillHostFolder {
			t.Errorf("warning %q scope: got %q want skill_host_folder", w.Code, w.ScopeType)
		}
		if w.ScopeID == nil || *w.ScopeID != hostID {
			t.Errorf("warning %q scopeID: got %v want %d", w.Code, w.ScopeID, hostID)
		}
	}
}

// TestClassifyEntry_ExternalSymlink verifies that an external symlink entry is
// classified as external_symlink (not available), so install.skill rejects it.
func TestClassifyEntry_ExternalSymlink(t *testing.T) {
	e := filesystem.HostEntry{
		Name:      "evil-host-symlink",
		IsSymlink: true,
		External:  true,
	}
	got := classifyEntry(e)
	if got != domain.SkillStatusExternalSymlink {
		t.Errorf("classifyEntry(external symlink): got %q want %q", got, domain.SkillStatusExternalSymlink)
	}
}

// TestClassifyEntry_InternalSymlink verifies that a symlink pointing within the
// host skills folder continues to be classified as available.
func TestClassifyEntry_InternalSymlink(t *testing.T) {
	e := filesystem.HostEntry{Name: "safe-link", IsSymlink: true, External: false}
	got := classifyEntry(e)
	if got != domain.SkillStatusAvailable {
		t.Errorf("classifyEntry(internal symlink): got %q want %q", got, domain.SkillStatusAvailable)
	}
}

// TestClassifyEntry_Dir verifies that a plain directory is classified as available.
func TestClassifyEntry_Dir(t *testing.T) {
	e := filesystem.HostEntry{Name: "my-skill", IsDir: true}
	got := classifyEntry(e)
	if got != domain.SkillStatusAvailable {
		t.Errorf("classifyEntry(dir): got %q want %q", got, domain.SkillStatusAvailable)
	}
}

// TestClassifyEntry_Broken verifies that a broken symlink is classified as unreadable.
func TestClassifyEntry_Broken(t *testing.T) {
	e := filesystem.HostEntry{Name: "dead-link", IsSymlink: true, Broken: true}
	got := classifyEntry(e)
	if got != domain.SkillStatusUnreadable {
		t.Errorf("classifyEntry(broken symlink): got %q want %q", got, domain.SkillStatusUnreadable)
	}
}

func TestScanHostInternal_FilesystemError(t *testing.T) {
	hostRepo := newMockHostRepo()
	ctx := context.Background()
	hostID, _, _ := hostRepo.UpsertAndActivate(ctx, "host", "/tmp/host", "/tmp/host/.agents/skills")

	fs := &mockFS{scanErr: errors.New("read failed")}
	svc := NewSkillHostService(hostRepo, newMockSettings(nil), fs, &mockRunner{}, &mockScanWriter{})
	host, _ := hostRepo.GetByID(ctx, hostID)
	_, err := svc.scanHostInternal(ctx, host, func(_ string, _, _ int, _ string) {})
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*domain.AppError)
	if !ok || ae.Code != domain.CodeFilesystem {
		t.Errorf("expected filesystem_error, got %v", err)
	}
}
