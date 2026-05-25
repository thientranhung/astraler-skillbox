package services

import (
	"context"
	"path/filepath"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
)

// -- mock project filesystem --

type mockProjectFS struct {
	validateErr        error
	normalizedPath     string
	normalizeErr       error
	// pathInfoResult overrides the default (readable dir). nil = readable dir.
	pathInfoResult     *filesystem.PathInfo
	pathInfoErr        error
	listEntriesResult  []filesystem.ProjectEntry
	listEntriesErr     error
}

func (m *mockProjectFS) ValidateProjectPath(_ string) error { return m.validateErr }

func (m *mockProjectFS) NormalizeAbs(path string) (string, error) {
	if m.normalizeErr != nil {
		return "", m.normalizeErr
	}
	if m.normalizedPath != "" {
		return m.normalizedPath, nil
	}
	return filepath.Clean(path), nil
}

func (m *mockProjectFS) PathInfo(_ string) (filesystem.PathInfo, error) {
	if m.pathInfoErr != nil {
		return filesystem.PathInfo{}, m.pathInfoErr
	}
	if m.pathInfoResult != nil {
		return *m.pathInfoResult, nil
	}
	return filesystem.PathInfo{Exists: true, IsDir: true, Readable: true}, nil
}

func (m *mockProjectFS) ListSkillEntries(_ string) ([]filesystem.ProjectEntry, error) {
	return m.listEntriesResult, m.listEntriesErr
}

// -- mock project repo --

type mockProjectRepo struct {
	projects  map[int64]*domain.Project
	byPath    map[string]*domain.Project
	nextID    int64
	upsertErr error
	getErr    error
	listErr   error
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{
		projects: make(map[int64]*domain.Project),
		byPath:   make(map[string]*domain.Project),
		nextID:   1,
	}
}

func (m *mockProjectRepo) UpsertByPath(_ context.Context, name, path string) (int64, bool, error) {
	if m.upsertErr != nil {
		return 0, false, m.upsertErr
	}
	if p, ok := m.byPath[path]; ok {
		return p.ID, false, nil
	}
	id := m.nextID
	m.nextID++
	p := &domain.Project{ID: id, Name: name, Path: path, Status: domain.ProjectStatusActive}
	m.projects[id] = p
	m.byPath[path] = p
	return id, true, nil
}

func (m *mockProjectRepo) GetByID(_ context.Context, id int64) (*domain.Project, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.projects[id], nil
}

func (m *mockProjectRepo) List(_ context.Context) ([]domain.Project, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	result := make([]domain.Project, 0, len(m.projects))
	for _, p := range m.projects {
		result = append(result, *p)
	}
	return result, nil
}

// -- mock project provider repo --

type mockProjectProviderRepo struct {
	byProject map[int64][]domain.ProjectProviderSummary
	err       error
}

func (m *mockProjectProviderRepo) ListByProject(_ context.Context, projectID int64) ([]domain.ProjectProviderSummary, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byProject[projectID], nil
}

// -- mock project warning repo --

type mockProjectWarningRepo struct {
	counts   map[int64]int
	warnings map[int64][]domain.Warning
	countErr error
	listErr  error
}

func (m *mockProjectWarningRepo) CountActiveForProject(_ context.Context, projectID int64) (int, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	return m.counts[projectID], nil
}

func (m *mockProjectWarningRepo) ListActiveForProject(_ context.Context, projectID int64) ([]domain.Warning, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.warnings[projectID], nil
}

// -- mock project install repo --

type mockProjectInstallRepo struct {
	byProject map[int64][]domain.Install
	err       error
}

func (m *mockProjectInstallRepo) ListByProject(_ context.Context, projectID int64) ([]domain.Install, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byProject[projectID], nil
}

// -- mock project scan committer --

type mockProjectScanCommitter struct {
	terminalErr           error
	terminalCallCount     int
	lastTerminalProjectID int64
	lastTerminalStatus    domain.ProjectStatus
	lastTerminalWarning   *domain.Warning

	fullScanErr         error
	fullScanCallCount   int
	lastProviders       []repositories.ProviderScanResult
	lastProjectWarnings []domain.Warning
}

func (m *mockProjectScanCommitter) CommitProjectTerminal(
	_ context.Context,
	projectID int64,
	status domain.ProjectStatus,
	warning *domain.Warning,
	_ time.Time,
) error {
	m.terminalCallCount++
	m.lastTerminalProjectID = projectID
	m.lastTerminalStatus = status
	m.lastTerminalWarning = warning
	return m.terminalErr
}

func (m *mockProjectScanCommitter) CommitProjectScan(
	_ context.Context,
	_ int64,
	provs []repositories.ProviderScanResult,
	projectWarnings []domain.Warning,
	_ time.Time,
) error {
	m.fullScanCallCount++
	m.lastProviders = provs
	m.lastProjectWarnings = projectWarnings
	return m.fullScanErr
}

// -- mock provider registry --

type mockProviderRegistry struct {
	adapters []providers.ProviderAdapter
}

func (m *mockProviderRegistry) All() []providers.ProviderAdapter { return m.adapters }

// -- mock provider definition repo --

type mockProviderDefRepo struct {
	defs map[string]*domain.ProviderDefinition
	err  error
}

func (m *mockProviderDefRepo) GetByKey(_ context.Context, key string) (*domain.ProviderDefinition, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.defs[key], nil
}

// -- mock host lister --

type mockHostLister struct {
	hosts []domain.SkillHostFolder
	err   error
}

func (m *mockHostLister) ListAll(_ context.Context) ([]domain.SkillHostFolder, error) {
	return m.hosts, m.err
}

// -- mock skills-by-host lister --

type mockSkillsByHostLister struct {
	skills map[int64][]domain.Skill
	err    error
}

func (m *mockSkillsByHostLister) ListByHost(_ context.Context, hostID int64) ([]domain.Skill, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.skills[hostID], nil
}

// -- mock provider adapter --

type mockAdapter struct {
	key    string
	result providers.DetectResult
	err    error
}

func (m *mockAdapter) Key() string { return m.key }
func (m *mockAdapter) Detect(_ string, _ providers.FsReader) (providers.DetectResult, error) {
	return m.result, m.err
}
