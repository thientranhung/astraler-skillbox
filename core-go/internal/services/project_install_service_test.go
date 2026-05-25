package services

import (
	"context"
	"os"
	"path/filepath"
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
