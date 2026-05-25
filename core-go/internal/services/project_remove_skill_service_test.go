package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// removeFixture builds a project on disk with one symlinked skill plus the
// service wiring needed to run removeSkillInternal end to end. The install row
// is registered in installRepo so RemoveSkill's load/ownership check passes.
type removeFixture struct {
	svc       *ProjectService
	project   *domain.Project
	install   domain.Install
	removeFS  *mockRemoveFS
	deleter   *mockInstallDeleter
	scanRepo  *mockProjectScanCommitter
	linkPath  string
	hostSkill string
}

func newRemoveFixture(t *testing.T) *removeFixture {
	t.Helper()

	projectDir := t.TempDir()
	projectSkillsDir := filepath.Join(projectDir, ".agents", "skills")
	if err := os.MkdirAll(projectSkillsDir, 0o755); err != nil {
		t.Fatalf("mkdir project skills: %v", err)
	}
	hostSkillsDir := t.TempDir()
	hostSkill := filepath.Join(hostSkillsDir, "documentation-writer")
	if err := os.MkdirAll(hostSkill, 0o755); err != nil {
		t.Fatalf("mkdir host skill: %v", err)
	}
	linkPath := filepath.Join(projectSkillsDir, "documentation-writer")
	if err := os.Symlink(hostSkill, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	gw := filesystem.NewGateway()
	project := &domain.Project{ID: 1, Name: "proj", Path: projectDir, Status: domain.ProjectStatusActive}
	projRepo := newMockProjectRepo()
	projRepo.projects[1] = project

	install := domain.Install{
		ID:                1001,
		ProjectProviderID: 50,
		SkillName:         "documentation-writer",
		InstallMode:       domain.InstallModeSymlink,
		InstallStatus:     domain.InstallStatusCurrent,
		ProjectSkillPath:  linkPath,
	}
	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProjectProviderID: 50, ProviderKey: providers.GenericAgentsKey, DetectionStatus: domain.DetectionStatusDetected}},
		},
	}
	installRepo := &mockProjectInstallRepo{byProject: map[int64][]domain.Install{1: {install}}}

	activeHost := &domain.SkillHostFolder{ID: 1, SkillsPath: hostSkillsDir, Status: domain.SkillHostStatusActive}
	hostReader := &mockActiveHostReader{host: activeHost}

	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{providers.NewGenericAgentsAdapter()}}
	hostLister := &mockHostLister{hosts: []domain.SkillHostFolder{*activeHost}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		providers.GenericAgentsKey: {ID: 10, Key: providers.GenericAgentsKey, Status: domain.ProviderStatusSupported, CanCreateStructure: true},
	}}
	skillLister := &mockSkillsByHostLister{skills: map[int64][]domain.Skill{1: {{ID: 1, Name: "documentation-writer", AbsolutePath: hostSkill, Status: domain.SkillStatusAvailable}}}}
	scanRepo := &mockProjectScanCommitter{}
	removeFS := &mockRemoveFS{facts: filesystem.EntryFacts{Exists: true, IsSymlink: true, ResolvedTarget: hostSkill}}
	deleter := &mockInstallDeleter{}

	svc := NewProjectService(projRepo, ppRepo, &mockProjectWarningRepo{}, installRepo, gw).
		WithScanDeps(&mockRunner{}, scanRepo).
		WithProviderDeps(registry, pdRepo, hostLister, skillLister).
		WithInstallDeps(gw, hostReader, skillLister).
		WithRemoveDeps(removeFS, deleter)

	return &removeFixture{svc: svc, project: project, install: install, removeFS: removeFS, deleter: deleter, scanRepo: scanRepo, linkPath: linkPath, hostSkill: hostSkill}
}

func TestRemoveSkillInternal_HappyPath(t *testing.T) {
	f := newRemoveFixture(t)
	meta, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	if err != nil {
		t.Fatalf("removeSkillInternal: %v", err)
	}
	m, ok := meta.(removeSkillMetadata)
	if !ok {
		t.Fatalf("metadata type: %T", meta)
	}
	if m.AlreadyAbsent {
		t.Errorf("AlreadyAbsent: got true want false")
	}
	if m.SkillName != "documentation-writer" || m.ProviderKey != providers.GenericAgentsKey {
		t.Errorf("metadata: %+v", m)
	}
	if f.removeFS.removeCalls != 1 {
		t.Errorf("RemoveSymlink calls: got %d want 1", f.removeFS.removeCalls)
	}
	if f.scanRepo.fullScanCallCount != 1 {
		t.Errorf("rescan calls: got %d want 1", f.scanRepo.fullScanCallCount)
	}
	if len(f.deleter.deletedIDs) != 1 || f.deleter.deletedIDs[0] != 1001 {
		t.Errorf("DeleteByID calls: got %v want [1001]", f.deleter.deletedIDs)
	}
}

func TestRemoveSkillInternal_AlreadyAbsent(t *testing.T) {
	f := newRemoveFixture(t)
	f.removeFS.facts = filesystem.EntryFacts{Exists: false}
	meta, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	if err != nil {
		t.Fatalf("removeSkillInternal: %v", err)
	}
	m := meta.(removeSkillMetadata)
	if !m.AlreadyAbsent {
		t.Errorf("AlreadyAbsent: got false want true")
	}
	if f.removeFS.removeCalls != 0 {
		t.Errorf("RemoveSymlink should not be called when absent, got %d", f.removeFS.removeCalls)
	}
	if f.scanRepo.fullScanCallCount != 1 || len(f.deleter.deletedIDs) != 1 {
		t.Errorf("rescan+delete should still run: scan=%d del=%v", f.scanRepo.fullScanCallCount, f.deleter.deletedIDs)
	}
}

func TestRemoveSkillInternal_NotSymlinkOnDisk_Conflict(t *testing.T) {
	f := newRemoveFixture(t)
	f.removeFS.facts = filesystem.EntryFacts{Exists: true, IsSymlink: false}
	_, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	assertAppErrorCode(t, err, domain.CodeConflict)
	if f.removeFS.removeCalls != 0 {
		t.Errorf("must not unlink a real entry, removeCalls=%d", f.removeFS.removeCalls)
	}
}

func TestRemoveSkillInternal_SymlinkOutsideActiveHost_Conflict(t *testing.T) {
	f := newRemoveFixture(t)
	f.removeFS.facts = filesystem.EntryFacts{Exists: true, IsSymlink: true, ResolvedTarget: filepath.Join(t.TempDir(), "elsewhere")}
	_, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	assertAppErrorCode(t, err, domain.CodeConflict)
	if f.removeFS.removeCalls != 0 {
		t.Errorf("must not unlink, removeCalls=%d", f.removeFS.removeCalls)
	}
}

func TestRemoveSkillInternal_UnlinkFails_NoRescanNoDelete(t *testing.T) {
	f := newRemoveFixture(t)
	f.removeFS.removeErr = os.ErrPermission
	_, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	assertAppErrorCode(t, err, domain.CodeFilesystem)
	if f.scanRepo.fullScanCallCount != 0 {
		t.Errorf("rescan must not run after unlink failure, got %d", f.scanRepo.fullScanCallCount)
	}
	if len(f.deleter.deletedIDs) != 0 {
		t.Errorf("delete must not run after unlink failure, got %v", f.deleter.deletedIDs)
	}
}

func TestRemoveSkill_Sync_RejectsNonCurrentStatus(t *testing.T) {
	f := newRemoveFixture(t)
	bad := f.install
	bad.InstallStatus = domain.InstallStatusOldHost
	f.svc.installRepo = &mockProjectInstallRepo{byProject: map[int64][]domain.Install{1: {bad}}}
	_, err := f.svc.RemoveSkill(context.Background(), 1, 1001)
	assertAppErrorCode(t, err, domain.CodeValidation)
}

func TestRemoveSkill_Sync_RejectsDirectMode(t *testing.T) {
	f := newRemoveFixture(t)
	bad := f.install
	bad.InstallMode = domain.InstallModeDirect
	f.svc.installRepo = &mockProjectInstallRepo{byProject: map[int64][]domain.Install{1: {bad}}}
	_, err := f.svc.RemoveSkill(context.Background(), 1, 1001)
	assertAppErrorCode(t, err, domain.CodeValidation)
}

func TestRemoveSkill_Sync_InstallNotFound(t *testing.T) {
	f := newRemoveFixture(t)
	_, err := f.svc.RemoveSkill(context.Background(), 1, 4242)
	assertAppErrorCode(t, err, domain.CodeValidation)
}

func TestRemoveSkill_Sync_ProjectNotActive(t *testing.T) {
	f := newRemoveFixture(t)
	f.project.Status = domain.ProjectStatusRemoved
	_, err := f.svc.RemoveSkill(context.Background(), 1, 1001)
	assertAppErrorCode(t, err, domain.CodeValidation)
}

func TestRemoveSkill_Sync_PathEscapesRoot(t *testing.T) {
	f := newRemoveFixture(t)

	outside := filepath.Join(t.TempDir(), "outside-skill")
	bad := f.install
	bad.ProjectSkillPath = outside
	f.svc.installRepo = &mockProjectInstallRepo{byProject: map[int64][]domain.Install{1: {bad}}}

	startCalled := false
	spyRunner := &mockRunner{startFn: func(_ context.Context, _ operations.Target, _ domain.OperationType, _ operations.WorkFn) (int64, error) {
		startCalled = true
		return 1, nil
	}}
	f.svc.WithScanDeps(spyRunner, f.scanRepo)

	_, err := f.svc.RemoveSkill(context.Background(), 1, 1001)
	assertAppErrorCode(t, err, domain.CodeValidation)
	if startCalled {
		t.Errorf("RemoveSkill must reject before runner.Start; Start was called")
	}
	if f.removeFS.removeCalls != 0 || len(f.deleter.deletedIDs) != 0 || f.scanRepo.fullScanCallCount != 0 {
		t.Errorf("no work should run on path-escape reject: remove=%d del=%v scan=%d",
			f.removeFS.removeCalls, len(f.deleter.deletedIDs), f.scanRepo.fullScanCallCount)
	}
}

// assertAppErrorCode asserts that err is a *domain.AppError with the given code.
// want is one of the domain.Code* string constants (e.g. domain.CodeConflict).
func assertAppErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", want)
	}
	ae, ok := err.(*domain.AppError)
	if !ok {
		t.Fatalf("expected *domain.AppError, got %T (%v)", err, err)
	}
	if ae.Code != want {
		t.Fatalf("error code: got %q want %q", ae.Code, want)
	}
}
