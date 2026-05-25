package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

func noopProgress(_ string, _, _ int, _ string) {}

// TestInstallSkillsInternal_HappyPath installs a single host skill into a project's
// existing .agents/skills folder via symlink, then verifies the inline rescan ran.
func TestInstallSkillsInternal_HappyPath(t *testing.T) {
	ctx := context.Background()

	// Project dir with an existing .agents/skills directory.
	projectDir := t.TempDir()
	projectSkillsDir := filepath.Join(projectDir, ".agents", "skills")
	if err := os.MkdirAll(projectSkillsDir, 0o755); err != nil {
		t.Fatalf("mkdir project skills: %v", err)
	}

	// Host dir with one skill folder.
	hostSkillsDir := t.TempDir()
	hostSkillPath := filepath.Join(hostSkillsDir, "documentation-writer")
	if err := os.MkdirAll(hostSkillPath, 0o755); err != nil {
		t.Fatalf("mkdir host skill: %v", err)
	}

	// Real gateway satisfies both ProjectFilesystem and InstallFilesystem.
	gw := filesystem.NewGateway()

	project := &domain.Project{
		ID:     1,
		Name:   "myproject",
		Path:   projectDir,
		Status: domain.ProjectStatusActive,
	}
	projRepo := newMockProjectRepo()
	projRepo.projects[1] = project

	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {
				{
					ProviderKey:     providers.GenericAgentsKey,
					DetectionStatus: domain.DetectionStatusDetected,
				},
			},
		},
	}

	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			providers.GenericAgentsKey: {
				ID:                 10,
				Key:                providers.GenericAgentsKey,
				Status:             domain.ProviderStatusSupported,
				CanCreateStructure: true,
			},
		},
	}

	activeHost := &domain.SkillHostFolder{
		ID:         1,
		SkillsPath: hostSkillsDir,
		Status:     domain.SkillHostStatusActive,
	}
	hostReader := &mockActiveHostReader{host: activeHost}

	skillLister := &mockSkillsByHostLister{
		skills: map[int64][]domain.Skill{
			1: {
				{
					ID:           1,
					Name:         "documentation-writer",
					AbsolutePath: hostSkillPath,
					Status:       domain.SkillStatusAvailable,
				},
			},
		},
	}

	// Rescan dependencies: real generic_agents adapter detects the on-disk folder,
	// host lister returns the active host so buildHostSummaries can classify.
	registry := &mockProviderRegistry{
		adapters: []providers.ProviderAdapter{providers.NewGenericAgentsAdapter()},
	}
	hostLister := &mockHostLister{hosts: []domain.SkillHostFolder{*activeHost}}
	scanRepo := &mockProjectScanCommitter{}
	runner := &mockRunner{}

	svc := NewProjectService(
		projRepo,
		ppRepo,
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		gw,
	).WithScanDeps(runner, scanRepo).
		WithProviderDeps(registry, pdRepo, hostLister, skillLister).
		WithInstallDeps(gw, hostReader, skillLister)

	meta, err := svc.installSkillsInternal(ctx, project, providers.GenericAgentsKey, []int64{1}, noopProgress)
	if err != nil {
		t.Fatalf("installSkillsInternal returned error: %v", err)
	}

	// Symlink exists and points to the host skill path.
	linkPath := filepath.Join(projectSkillsDir, "documentation-writer")
	fi, lerr := os.Lstat(linkPath)
	if lerr != nil {
		t.Fatalf("expected symlink at %q: %v", linkPath, lerr)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %q to be a symlink, got mode %v", linkPath, fi.Mode())
	}
	dest, rerr := os.Readlink(linkPath)
	if rerr != nil {
		t.Fatalf("readlink: %v", rerr)
	}
	if dest != hostSkillPath {
		t.Fatalf("symlink target: got %q want %q", dest, hostSkillPath)
	}

	// Rescan ran exactly once.
	if scanRepo.fullScanCallCount != 1 {
		t.Fatalf("fullScanCallCount: got %d want 1", scanRepo.fullScanCallCount)
	}

	// Metadata reflects a single successful install.
	im, ok := meta.(installMetadata)
	if !ok {
		t.Fatalf("metadata type: got %T want installMetadata", meta)
	}
	if im.Requested != 1 || im.Created != 1 || im.Failed != 0 {
		t.Fatalf("metadata: got %+v want Requested=1 Created=1 Failed=0", im)
	}
	if im.ProviderKey != providers.GenericAgentsKey {
		t.Fatalf("metadata ProviderKey: got %q want %q", im.ProviderKey, providers.GenericAgentsKey)
	}
}

// installSvcFixture bundles the inputs and observable collaborators needed to
// exercise installSkillsInternal against the real filesystem gateway.
type installSvcFixture struct {
	svc       *ProjectService
	project   *domain.Project
	scanRepo  *mockProjectScanCommitter
	ppRepo    *mockProjectProviderRepo
	pdRepo    *mockProviderDefRepo
	hostReader *mockActiveHostReader
	skillLister *mockSkillsByHostLister
	installFS InstallFilesystem
}

// installFixtureOpts tunes the generic_agents install fixture for a single test.
type installFixtureOpts struct {
	// createProjectSkillsDir pre-creates .agents/skills under the project root.
	createProjectSkillsDir bool
	// skills are the host skills available on the active host (ID 1).
	skills []domain.Skill
	// installFS overrides the InstallFilesystem; nil uses the real gateway.
	installFS InstallFilesystem
}

// newGenericInstallFixture wires a generic_agents install scenario backed by the
// real filesystem gateway. The active host lives in a temp dir and each requested
// host skill must point at a real folder beneath it.
func newGenericInstallFixture(t *testing.T, opts installFixtureOpts) installSvcFixture {
	t.Helper()

	projectDir := t.TempDir()
	if opts.createProjectSkillsDir {
		if err := os.MkdirAll(filepath.Join(projectDir, ".agents", "skills"), 0o755); err != nil {
			t.Fatalf("mkdir project skills: %v", err)
		}
	}

	gw := filesystem.NewGateway()

	project := &domain.Project{
		ID:     1,
		Name:   "myproject",
		Path:   projectDir,
		Status: domain.ProjectStatusActive,
	}
	projRepo := newMockProjectRepo()
	projRepo.projects[1] = project

	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProviderKey: providers.GenericAgentsKey, DetectionStatus: domain.DetectionStatusDetected}},
		},
	}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			providers.GenericAgentsKey: {
				ID:                 10,
				Key:                providers.GenericAgentsKey,
				Status:             domain.ProviderStatusSupported,
				CanCreateStructure: true,
			},
		},
	}

	activeHost := &domain.SkillHostFolder{ID: 1, SkillsPath: t.TempDir(), Status: domain.SkillHostStatusActive}
	hostReader := &mockActiveHostReader{host: activeHost}
	skillLister := &mockSkillsByHostLister{skills: map[int64][]domain.Skill{1: opts.skills}}

	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{providers.NewGenericAgentsAdapter()}}
	hostLister := &mockHostLister{hosts: []domain.SkillHostFolder{*activeHost}}
	scanRepo := &mockProjectScanCommitter{}
	runner := &mockRunner{}

	var installFS InstallFilesystem = gw
	if opts.installFS != nil {
		installFS = opts.installFS
	}

	svc := NewProjectService(
		projRepo,
		ppRepo,
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		gw,
	).WithScanDeps(runner, scanRepo).
		WithProviderDeps(registry, pdRepo, hostLister, skillLister).
		WithInstallDeps(installFS, hostReader, skillLister)

	return installSvcFixture{
		svc:         svc,
		project:     project,
		scanRepo:    scanRepo,
		ppRepo:      ppRepo,
		pdRepo:      pdRepo,
		hostReader:  hostReader,
		skillLister: skillLister,
		installFS:   installFS,
	}
}

// makeHostSkill creates a real folder under the active host and returns a skill
// pointing at it. id 0 leaves the skill unmade on disk (caller supplies path).
func makeHostSkill(t *testing.T, hostSkillsDir string, id int64, name string, status domain.SkillStatus) domain.Skill {
	t.Helper()
	abs := filepath.Join(hostSkillsDir, name)
	if err := os.MkdirAll(abs, 0o755); err != nil {
		t.Fatalf("mkdir host skill %q: %v", name, err)
	}
	return domain.Skill{ID: id, Name: name, AbsolutePath: abs, Status: status}
}

func mustAppErr(t *testing.T, err error) *domain.AppError {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	ae, ok := err.(*domain.AppError)
	if !ok {
		t.Fatalf("error type: got %T want *domain.AppError (%v)", err, err)
	}
	return ae
}

// --- Task 7: auto-create skills dir (generic_agents) ---

// TestInstallSkillsInternal_AutoCreateSkillsDir verifies that when .agents/skills
// does not exist, the generic_agents provider (CanCreateStructure=true) creates it,
// the symlink lands inside, and the install is classified current.
func TestInstallSkillsInternal_AutoCreateSkillsDir(t *testing.T) {
	ctx := context.Background()

	fx := newGenericInstallFixture(t, installFixtureOpts{
		createProjectSkillsDir: false, // do NOT pre-create
	})
	sk := makeHostSkill(t, fx.hostReader.host.SkillsPath, 1, "documentation-writer", domain.SkillStatusAvailable)
	fx.skillLister.skills[1] = []domain.Skill{sk}

	skillsDir := filepath.Join(fx.project.Path, ".agents", "skills")
	if _, err := os.Lstat(skillsDir); !os.IsNotExist(err) {
		t.Fatalf("precondition: skills dir should be absent, got err=%v", err)
	}

	meta, err := fx.svc.installSkillsInternal(ctx, fx.project, providers.GenericAgentsKey, []int64{1}, noopProgress)
	if err != nil {
		t.Fatalf("installSkillsInternal returned error: %v", err)
	}

	// Skills dir was created on disk.
	di, derr := os.Stat(skillsDir)
	if derr != nil {
		t.Fatalf("expected skills dir created at %q: %v", skillsDir, derr)
	}
	if !di.IsDir() {
		t.Fatalf("expected %q to be a directory", skillsDir)
	}

	// Symlink exists inside it.
	linkPath := filepath.Join(skillsDir, "documentation-writer")
	fi, lerr := os.Lstat(linkPath)
	if lerr != nil {
		t.Fatalf("expected symlink at %q: %v", linkPath, lerr)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected %q to be a symlink, got mode %v", linkPath, fi.Mode())
	}

	if fx.scanRepo.fullScanCallCount != 1 {
		t.Fatalf("fullScanCallCount: got %d want 1", fx.scanRepo.fullScanCallCount)
	}

	im, ok := meta.(installMetadata)
	if !ok {
		t.Fatalf("metadata type: got %T want installMetadata", meta)
	}
	if im.Requested != 1 || im.Created != 1 || im.Failed != 0 {
		t.Fatalf("metadata: got %+v want Requested=1 Created=1 Failed=0", im)
	}
}

// --- Task 8: Claude no-scaffold block ---

// TestInstallSkillsInternal_ClaudeNoScaffold verifies that an experimental claude
// provider with CanCreateStructure=false and an absent .claude/skills folder fails
// with provider_error before any write, and that the rescan never runs (the error
// is raised in the ensure-dir phase, ahead of symlink creation and rescan).
func TestInstallSkillsInternal_ClaudeNoScaffold(t *testing.T) {
	ctx := context.Background()

	projectDir := t.TempDir()
	hostSkillsDir := t.TempDir()
	skillPath := filepath.Join(hostSkillsDir, "documentation-writer")
	if err := os.MkdirAll(skillPath, 0o755); err != nil {
		t.Fatalf("mkdir host skill: %v", err)
	}

	gw := filesystem.NewGateway()

	project := &domain.Project{ID: 1, Name: "myproject", Path: projectDir, Status: domain.ProjectStatusActive}
	projRepo := newMockProjectRepo()
	projRepo.projects[1] = project

	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProviderKey: providers.ClaudeKey, DetectionStatus: domain.DetectionStatusDetected}},
		},
	}
	pdRepo := &mockProviderDefRepo{
		defs: map[string]*domain.ProviderDefinition{
			providers.ClaudeKey: {
				ID:                 20,
				Key:                providers.ClaudeKey,
				Status:             domain.ProviderStatusExperimental,
				CanCreateStructure: false,
			},
		},
	}

	activeHost := &domain.SkillHostFolder{ID: 1, SkillsPath: hostSkillsDir, Status: domain.SkillHostStatusActive}
	hostReader := &mockActiveHostReader{host: activeHost}
	skillLister := &mockSkillsByHostLister{
		skills: map[int64][]domain.Skill{
			1: {{ID: 1, Name: "documentation-writer", AbsolutePath: skillPath, Status: domain.SkillStatusAvailable}},
		},
	}

	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{providers.NewGenericAgentsAdapter()}}
	hostLister := &mockHostLister{hosts: []domain.SkillHostFolder{*activeHost}}
	scanRepo := &mockProjectScanCommitter{}
	runner := &mockRunner{}

	svc := NewProjectService(
		projRepo, ppRepo, &mockProjectWarningRepo{}, &mockProjectInstallRepo{}, gw,
	).WithScanDeps(runner, scanRepo).
		WithProviderDeps(registry, pdRepo, hostLister, skillLister).
		WithInstallDeps(gw, hostReader, skillLister)

	skillsDir := filepath.Join(projectDir, ".claude", "skills")
	if _, err := os.Lstat(skillsDir); !os.IsNotExist(err) {
		t.Fatalf("precondition: .claude/skills should be absent, got err=%v", err)
	}

	_, err := svc.installSkillsInternal(ctx, project, providers.ClaudeKey, []int64{1}, noopProgress)
	ae := mustAppErr(t, err)
	if ae.Code != domain.CodeProvider {
		t.Fatalf("error code: got %q want %q", ae.Code, domain.CodeProvider)
	}

	// No write happened: .claude and .claude/skills must still be absent.
	if _, derr := os.Lstat(skillsDir); !os.IsNotExist(derr) {
		t.Fatalf(".claude/skills should not have been created, got err=%v", derr)
	}
	if _, derr := os.Lstat(filepath.Join(projectDir, ".claude")); !os.IsNotExist(derr) {
		t.Fatalf(".claude should not have been created, got err=%v", derr)
	}

	// The provider_error is raised before the write phase, so rescan never runs.
	if scanRepo.fullScanCallCount != 0 {
		t.Fatalf("fullScanCallCount: got %d want 0", scanRepo.fullScanCallCount)
	}
}

// --- Task 9: conflict abort (atomic) ---

// TestInstallSkillsInternal_ConflictAbort verifies that a pre-existing entry where
// the symlink would go aborts the whole install with conflict_error (naming the
// skill), leaves the filesystem untouched, and never reaches the rescan. Both a real
// directory and a broken symlink count as conflicts because the check uses Lstat.
func TestInstallSkillsInternal_ConflictAbort(t *testing.T) {
	cases := []struct {
		name string
		// seed creates the pre-existing entry at linkPath; returns a fingerprint to
		// later assert the filesystem is unchanged.
		seed func(t *testing.T, linkPath string)
	}{
		{
			name: "real directory",
			seed: func(t *testing.T, linkPath string) {
				if err := os.MkdirAll(linkPath, 0o755); err != nil {
					t.Fatalf("seed dir: %v", err)
				}
			},
		},
		{
			name: "broken symlink",
			seed: func(t *testing.T, linkPath string) {
				// Point at a target that does not exist -> broken symlink.
				if err := os.Symlink(filepath.Join(t.TempDir(), "does-not-exist"), linkPath); err != nil {
					t.Fatalf("seed broken symlink: %v", err)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			fx := newGenericInstallFixture(t, installFixtureOpts{createProjectSkillsDir: true})
			sk := makeHostSkill(t, fx.hostReader.host.SkillsPath, 1, "documentation-writer", domain.SkillStatusAvailable)
			fx.skillLister.skills[1] = []domain.Skill{sk}

			skillsDir := filepath.Join(fx.project.Path, ".agents", "skills")
			linkPath := filepath.Join(skillsDir, "documentation-writer")
			tc.seed(t, linkPath)

			// Capture the entry mode before the call so we can prove it is unchanged.
			before, berr := os.Lstat(linkPath)
			if berr != nil {
				t.Fatalf("lstat seeded entry: %v", berr)
			}

			_, err := fx.svc.installSkillsInternal(ctx, fx.project, providers.GenericAgentsKey, []int64{1}, noopProgress)
			ae := mustAppErr(t, err)
			if ae.Code != domain.CodeConflict {
				t.Fatalf("error code: got %q want %q", ae.Code, domain.CodeConflict)
			}
			if !strings.Contains(ae.TechnicalMessage, "documentation-writer") {
				t.Fatalf("conflict error should name the skill, got %q", ae.TechnicalMessage)
			}

			// Filesystem unchanged: the seeded entry retains its original mode and no
			// extra symlink was created (still exactly one entry in the dir).
			after, aerr := os.Lstat(linkPath)
			if aerr != nil {
				t.Fatalf("lstat after: %v", aerr)
			}
			if before.Mode() != after.Mode() {
				t.Fatalf("entry mode changed: before=%v after=%v", before.Mode(), after.Mode())
			}
			entries, derr := os.ReadDir(skillsDir)
			if derr != nil {
				t.Fatalf("readdir: %v", derr)
			}
			if len(entries) != 1 {
				t.Fatalf("skills dir should contain only the seeded entry, got %d entries", len(entries))
			}

			// Conflict is pre-write, so the rescan never runs.
			if fx.scanRepo.fullScanCallCount != 0 {
				t.Fatalf("fullScanCallCount: got %d want 0", fx.scanRepo.fullScanCallCount)
			}
		})
	}
}

// --- Task 10: validation + within-root enforcement ---

// TestInstallSkillsInternal_ValidationErrors exercises the synchronous validation
// guards in installSkillsInternal. Each case mutates a healthy generic_agents
// fixture into one bad precondition and asserts the expected error code, that the
// rescan never runs, and (for the unsafe-name case) that no symlink is written.
func TestInstallSkillsInternal_ValidationErrors(t *testing.T) {
	cases := []struct {
		name     string
		wantCode string
		// mutate adjusts the fixture into the failure precondition. It runs after a
		// default-available skill (ID 1) has been seeded on the active host.
		mutate func(t *testing.T, fx installSvcFixture)
	}{
		{
			name:     "provider not in project",
			wantCode: domain.CodeValidation,
			mutate: func(_ *testing.T, fx installSvcFixture) {
				fx.ppRepo.byProject[1] = nil
			},
		},
		{
			name:     "provider def unsupported",
			wantCode: domain.CodeProvider,
			mutate: func(_ *testing.T, fx installSvcFixture) {
				fx.pdRepo.defs[providers.GenericAgentsKey].Status = domain.ProviderStatusUnsupported
			},
		},
		{
			name:     "detection status missing",
			wantCode: domain.CodeValidation,
			mutate: func(_ *testing.T, fx installSvcFixture) {
				fx.ppRepo.byProject[1] = []domain.ProjectProviderSummary{
					{ProviderKey: providers.GenericAgentsKey, DetectionStatus: domain.DetectionStatusMissing},
				}
			},
		},
		{
			name:     "skill id not found",
			wantCode: domain.CodeValidation,
			mutate: func(_ *testing.T, fx installSvcFixture) {
				fx.skillLister.skills[1] = nil
			},
		},
		{
			name:     "skill status missing",
			wantCode: domain.CodeValidation,
			mutate: func(_ *testing.T, fx installSvcFixture) {
				sk := fx.skillLister.skills[1][0]
				sk.Status = domain.SkillStatusMissing
				fx.skillLister.skills[1] = []domain.Skill{sk}
			},
		},
		{
			name:     "no active host",
			wantCode: domain.CodeValidation,
			mutate: func(_ *testing.T, fx installSvcFixture) {
				fx.hostReader.host = nil
			},
		},
		{
			name:     "unsafe skill name",
			wantCode: domain.CodeValidation,
			mutate: func(_ *testing.T, fx installSvcFixture) {
				// A resolvable, available skill whose Name escapes the skills dir.
				// Validation rejects it before any symlink is attempted, so the
				// AbsolutePath need not exist on disk.
				fx.skillLister.skills[1] = []domain.Skill{{
					ID:           1,
					Name:         "../escape",
					AbsolutePath: filepath.Join(fx.hostReader.host.SkillsPath, "escape"),
					Status:       domain.SkillStatusAvailable,
				}}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			fx := newGenericInstallFixture(t, installFixtureOpts{createProjectSkillsDir: true})
			sk := makeHostSkill(t, fx.hostReader.host.SkillsPath, 1, "documentation-writer", domain.SkillStatusAvailable)
			fx.skillLister.skills[1] = []domain.Skill{sk}

			tc.mutate(t, fx)

			_, err := fx.svc.installSkillsInternal(ctx, fx.project, providers.GenericAgentsKey, []int64{1}, noopProgress)
			ae := mustAppErr(t, err)
			if ae.Code != tc.wantCode {
				t.Fatalf("error code: got %q want %q", ae.Code, tc.wantCode)
			}

			// All these guards fire before the write phase: no rescan, no symlink.
			if fx.scanRepo.fullScanCallCount != 0 {
				t.Fatalf("fullScanCallCount: got %d want 0", fx.scanRepo.fullScanCallCount)
			}
			entries, derr := os.ReadDir(filepath.Join(fx.project.Path, ".agents", "skills"))
			if derr != nil {
				t.Fatalf("readdir skills dir: %v", derr)
			}
			if len(entries) != 0 {
				t.Fatalf("skills dir should be empty after a pre-write failure, got %d entries", len(entries))
			}
		})
	}
}

// --- Task 11: multi-skill partial filesystem failure ---

// flakyInstallFS wraps the real gateway but injects a CreateSymlink failure once
// the call count exceeds failAfter. Calls up to and including failAfter delegate to
// the real os, so successful links land on disk for the rescan to observe.
type flakyInstallFS struct {
	callCount int
	failAfter int // fail CreateSymlink on calls > failAfter
	gw        *filesystem.Gateway
}

func (f *flakyInstallFS) LstatExists(path string) (bool, error) { return f.gw.LstatExists(path) }
func (f *flakyInstallFS) EnsureDir(path string) error           { return f.gw.EnsureDir(path) }
func (f *flakyInstallFS) CreateSymlink(src, link string) error {
	f.callCount++
	if f.callCount > f.failAfter {
		return &filesystem.FilesystemError{Code: filesystem.ErrNotWritable, Path: link, Message: "injected failure"}
	}
	return f.gw.CreateSymlink(src, link)
}

// TestInstallSkillsInternal_PartialFailureStopsAndRescans installs two skills where
// the second symlink fails. The loop stops after the first error, the rescan still
// runs once (so the successfully-linked skill is classified), the returned error is
// filesystem_error, and metadata reports requested=2 created=1 failed=1.
func TestInstallSkillsInternal_PartialFailureStopsAndRescans(t *testing.T) {
	ctx := context.Background()

	flaky := &flakyInstallFS{failAfter: 1, gw: filesystem.NewGateway()}
	fx := newGenericInstallFixture(t, installFixtureOpts{
		createProjectSkillsDir: true,
		installFS:              flaky,
	})
	first := makeHostSkill(t, fx.hostReader.host.SkillsPath, 1, "alpha-skill", domain.SkillStatusAvailable)
	second := makeHostSkill(t, fx.hostReader.host.SkillsPath, 2, "beta-skill", domain.SkillStatusAvailable)
	fx.skillLister.skills[1] = []domain.Skill{first, second}

	meta, err := fx.svc.installSkillsInternal(ctx, fx.project, providers.GenericAgentsKey, []int64{1, 2}, noopProgress)

	ae := mustAppErr(t, err)
	if ae.Code != domain.CodeFilesystem {
		t.Fatalf("error code: got %q want %q", ae.Code, domain.CodeFilesystem)
	}

	// Loop stopped after the first failure: exactly two CreateSymlink attempts.
	if flaky.callCount != 2 {
		t.Fatalf("CreateSymlink call count: got %d want 2", flaky.callCount)
	}

	// The first symlink was actually written; the rescan classifies it as current.
	skillsDir := filepath.Join(fx.project.Path, ".agents", "skills")
	if _, lerr := os.Lstat(filepath.Join(skillsDir, "alpha-skill")); lerr != nil {
		t.Fatalf("first symlink should exist on disk: %v", lerr)
	}
	if _, lerr := os.Lstat(filepath.Join(skillsDir, "beta-skill")); !os.IsNotExist(lerr) {
		t.Fatalf("second symlink should not exist, got err=%v", lerr)
	}

	// Rescan still ran exactly once despite the partial failure.
	if fx.scanRepo.fullScanCallCount != 1 {
		t.Fatalf("fullScanCallCount: got %d want 1", fx.scanRepo.fullScanCallCount)
	}

	im, ok := meta.(installMetadata)
	if !ok {
		t.Fatalf("metadata type: got %T want installMetadata", meta)
	}
	if im.Requested != 2 || im.Created != 1 || im.Failed != 1 {
		t.Fatalf("metadata: got %+v want Requested=2 Created=1 Failed=1", im)
	}
}

// TestInstallSkillsInternal_EnsureDirPathFirstSymlinkFails covers the ensure-dir
// branch: .agents/skills does not exist, gets created, then the very first
// CreateSymlink fails. The rescan still runs once, the operation reports
// filesystem_error, and metadata is requested=1 created=0 failed=1.
func TestInstallSkillsInternal_EnsureDirPathFirstSymlinkFails(t *testing.T) {
	ctx := context.Background()

	flaky := &flakyInstallFS{failAfter: 0, gw: filesystem.NewGateway()}
	fx := newGenericInstallFixture(t, installFixtureOpts{
		createProjectSkillsDir: false, // ensure-dir branch
		installFS:              flaky,
	})
	sk := makeHostSkill(t, fx.hostReader.host.SkillsPath, 1, "alpha-skill", domain.SkillStatusAvailable)
	fx.skillLister.skills[1] = []domain.Skill{sk}

	meta, err := fx.svc.installSkillsInternal(ctx, fx.project, providers.GenericAgentsKey, []int64{1}, noopProgress)

	ae := mustAppErr(t, err)
	if ae.Code != domain.CodeFilesystem {
		t.Fatalf("error code: got %q want %q", ae.Code, domain.CodeFilesystem)
	}

	// EnsureDir created the skills directory even though the symlink then failed.
	skillsDir := filepath.Join(fx.project.Path, ".agents", "skills")
	di, derr := os.Stat(skillsDir)
	if derr != nil || !di.IsDir() {
		t.Fatalf("skills dir should have been created: err=%v", derr)
	}
	if _, lerr := os.Lstat(filepath.Join(skillsDir, "alpha-skill")); !os.IsNotExist(lerr) {
		t.Fatalf("symlink should not exist after failure, got err=%v", lerr)
	}

	// Rescan still runs once because the write phase was reached.
	if fx.scanRepo.fullScanCallCount != 1 {
		t.Fatalf("fullScanCallCount: got %d want 1", fx.scanRepo.fullScanCallCount)
	}

	im, ok := meta.(installMetadata)
	if !ok {
		t.Fatalf("metadata type: got %T want installMetadata", meta)
	}
	if im.Requested != 1 || im.Created != 0 || im.Failed != 1 {
		t.Fatalf("metadata: got %+v want Requested=1 Created=0 Failed=1", im)
	}
}
