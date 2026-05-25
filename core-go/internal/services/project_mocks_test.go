package services

import (
	"context"
	"path/filepath"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// -- mock project filesystem --

type mockProjectFS struct {
	validateErr    error
	normalizedPath string
	normalizeErr   error
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
