package services

import (
	"context"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
)

// -- mock filesystem --

type mockFS struct {
	validateErr   error
	ensureCreated bool
	ensureErr     error
	scanEntries   []filesystem.HostEntry
	scanErr       error
}

func (m *mockFS) ValidateHostPath(_ string) error         { return m.validateErr }
func (m *mockFS) EnsureAgentsSkills(_ string) (bool, error) { return m.ensureCreated, m.ensureErr }
func (m *mockFS) ScanHostFolder(_ string) ([]filesystem.HostEntry, error) {
	return m.scanEntries, m.scanErr
}

// -- mock host repo --

type mockHostRepo struct {
	hosts     map[int64]*domain.SkillHostFolder
	byPath    map[string]*domain.SkillHostFolder
	nextID    int64
	activeID  *int64
	upsertErr error
}

func newMockHostRepo() *mockHostRepo {
	return &mockHostRepo{
		hosts:  make(map[int64]*domain.SkillHostFolder),
		byPath: make(map[string]*domain.SkillHostFolder),
		nextID: 1,
	}
}

func (m *mockHostRepo) GetByID(_ context.Context, id int64) (*domain.SkillHostFolder, error) {
	h, ok := m.hosts[id]
	if !ok {
		return nil, nil
	}
	return h, nil
}

func (m *mockHostRepo) GetByPath(_ context.Context, path string) (*domain.SkillHostFolder, error) {
	return m.byPath[path], nil
}

func (m *mockHostRepo) UpsertAndActivate(_ context.Context, name, path, skillsPath string) (int64, bool, error) {
	if m.upsertErr != nil {
		return 0, false, m.upsertErr
	}
	if h, ok := m.byPath[path]; ok {
		h.Status = domain.SkillHostStatusActive
		m.activeID = &h.ID
		return h.ID, false, nil
	}
	id := m.nextID
	m.nextID++
	h := &domain.SkillHostFolder{
		ID: id, Name: name, Path: path, SkillsPath: skillsPath,
		Status: domain.SkillHostStatusActive,
	}
	m.hosts[id] = h
	m.byPath[path] = h
	m.activeID = &id
	return id, true, nil
}

func (m *mockHostRepo) UpdateStatus(_ context.Context, id int64, status domain.SkillHostStatus) error {
	if h, ok := m.hosts[id]; ok {
		h.Status = status
	}
	return nil
}

func (m *mockHostRepo) UpdateLastScannedAt(_ context.Context, id int64, t time.Time) error {
	if h, ok := m.hosts[id]; ok {
		h.LastScannedAt = &t
	}
	return nil
}

// -- mock app settings repo --

type mockAppSettingsRepo struct {
	settings *domain.AppSettings
}

func newMockSettings(activeID *int64) *mockAppSettingsRepo {
	return &mockAppSettingsRepo{settings: &domain.AppSettings{
		ID:                      1,
		ActiveSkillHostFolderID: activeID,
		DefaultInstallMode:      "symlink",
		DatabaseVersion:         1,
	}}
}

func (m *mockAppSettingsRepo) Get(_ context.Context) (*domain.AppSettings, error) {
	return m.settings, nil
}

// -- mock skill repo (used by SkillLibraryService tests) --

type mockSkillRepo struct {
	skills map[int64][]domain.Skill
}

func newMockSkillRepo() *mockSkillRepo {
	return &mockSkillRepo{skills: make(map[int64][]domain.Skill)}
}

func (m *mockSkillRepo) UpsertMany(_ context.Context, hostID int64, skills []domain.Skill) error {
	for _, s := range skills {
		s.ID = int64(len(m.skills[hostID]) + 1)
		m.skills[hostID] = append(m.skills[hostID], s)
	}
	return nil
}

func (m *mockSkillRepo) ListByHost(_ context.Context, hostID int64) ([]domain.Skill, error) {
	return m.skills[hostID], nil
}

func (m *mockSkillRepo) MarkMissing(_ context.Context, hostID int64, presentIDs []int64) error {
	present := make(map[int64]struct{}, len(presentIDs))
	for _, id := range presentIDs {
		present[id] = struct{}{}
	}
	for i, s := range m.skills[hostID] {
		if _, ok := present[s.ID]; !ok {
			m.skills[hostID][i].Status = domain.SkillStatusMissing
		}
	}
	return nil
}

func (m *mockSkillRepo) ListIDsByHost(_ context.Context, hostID int64) ([]int64, error) {
	var ids []int64
	for _, s := range m.skills[hostID] {
		ids = append(ids, s.ID)
	}
	return ids, nil
}

// -- mock warning repo (used by SkillLibraryService tests) --

type mockWarningRepo struct {
	warnings []domain.Warning
}

func (m *mockWarningRepo) Insert(_ context.Context, w domain.Warning) (int64, error) {
	w.ID = int64(len(m.warnings) + 1)
	m.warnings = append(m.warnings, w)
	return w.ID, nil
}

func (m *mockWarningRepo) ListByScope(_ context.Context, scopeType domain.WarningScopeType, scopeID int64, includeResolved bool) ([]domain.Warning, error) {
	var out []domain.Warning
	for _, w := range m.warnings {
		if w.ScopeType == scopeType && w.ScopeID != nil && *w.ScopeID == scopeID {
			if includeResolved || !w.IsResolved {
				out = append(out, w)
			}
		}
	}
	return out, nil
}

func (m *mockWarningRepo) ClearByScope(_ context.Context, _ domain.WarningScopeType, _ int64) error {
	for i := range m.warnings {
		m.warnings[i].IsResolved = true
	}
	return nil
}

// -- mock scan committer (used by SkillHostService tests) --

type mockScanWriter struct {
	skills   []domain.Skill
	warnings []domain.Warning
	err      error
}

func (m *mockScanWriter) CommitScanResults(_ context.Context, _ int64, skills []domain.Skill, warnings []domain.Warning, _ time.Time) error {
	m.skills = skills
	m.warnings = warnings
	return m.err
}

// -- mock runner --

type mockRunner struct {
	startFn func(ctx context.Context, target operations.Target, opType domain.OperationType, fn operations.WorkFn) (int64, error)
}

func (m *mockRunner) Start(ctx context.Context, target operations.Target, opType domain.OperationType, fn operations.WorkFn) (int64, error) {
	if m.startFn != nil {
		return m.startFn(ctx, target, opType, fn)
	}
	return 1, nil
}

func (m *mockRunner) Cancel(_ context.Context, _ int64) (bool, error) { return true, nil }
