package services

import (
	"context"
	"errors"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

// -- mocks --

type mockProviderRegistryRepo struct {
	entries []domain.ProviderRegistryEntry
	err     error
}

func (m *mockProviderRegistryRepo) ListAll(_ context.Context) ([]domain.ProviderRegistryEntry, error) {
	return m.entries, m.err
}

func (m *mockProviderRegistryRepo) GetByKey(_ context.Context, key string) (*domain.ProviderDefinition, error) {
	for _, e := range m.entries {
		if e.Definition.Key == key {
			d := e.Definition
			return &d, nil
		}
	}
	return nil, nil
}

type mockProviderUserSettingsRepo struct {
	settings []domain.ProviderUserSetting
	listErr  error
	upsertErr error
}

func (m *mockProviderUserSettingsRepo) ListAll(_ context.Context) ([]domain.ProviderUserSetting, error) {
	return m.settings, m.listErr
}

func (m *mockProviderUserSettingsRepo) Upsert(_ context.Context, _ int64, _ bool) error {
	return m.upsertErr
}

type mockProviderOverrideRepo struct {
	overrides  []domain.ProviderPathOverride
	listErr    error
	upsertErr  error
	deleteRet  bool
	deleteErr  error
	idByKey    map[string]int64
	idByKeyErr error
}

func (m *mockProviderOverrideRepo) ListAll(_ context.Context) ([]domain.ProviderPathOverride, error) {
	return m.overrides, m.listErr
}

func (m *mockProviderOverrideRepo) Upsert(_ context.Context, o domain.ProviderPathOverride) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	m.overrides = append(m.overrides, o)
	return nil
}

func (m *mockProviderOverrideRepo) Delete(_ context.Context, _ int64, _, _ string) (bool, error) {
	return m.deleteRet, m.deleteErr
}

func (m *mockProviderOverrideRepo) GetProviderIDByKey(_ context.Context, key string) (int64, error) {
	if m.idByKeyErr != nil {
		return 0, m.idByKeyErr
	}
	if m.idByKey != nil {
		return m.idByKey[key], nil
	}
	return 0, nil
}

func makeTestEntry(key, status string) domain.ProviderRegistryEntry {
	iconKey := key
	return domain.ProviderRegistryEntry{
		Definition: domain.ProviderDefinition{
			ID:           1,
			Key:          key,
			DisplayName:  key,
			ProviderType: key,
			IconKey:      &iconKey,
			Status:       domain.ProviderStatus(status),
		},
		Candidates: []domain.ProviderPathCandidate{
			{RelativePath: "." + key, Scope: "project", Purpose: "detect", Priority: 10, VerificationStatus: "assumed"},
		},
	}
}

func makeSvc(entries []domain.ProviderRegistryEntry, overrides []domain.ProviderPathOverride) *ProviderRegistryService {
	repo := &mockProviderRegistryRepo{entries: entries}
	idByKey := make(map[string]int64)
	for _, e := range entries {
		idByKey[e.Definition.Key] = e.Definition.ID
	}
	overrideRepo := &mockProviderOverrideRepo{
		overrides: overrides,
		idByKey:   idByKey,
	}
	return NewProviderRegistryService(repo, overrideRepo, &mockProviderUserSettingsRepo{})
}

// -- List tests --

func TestProviderRegistryService_List_ReturnsEntries(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{
		makeTestEntry("generic_agents", "supported"),
		makeTestEntry("claude", "experimental"),
	}
	svc := makeSvc(entries, nil)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("len: got %d want 2", len(got))
	}
	if got[0].Definition.Key != "generic_agents" {
		t.Errorf("first key: got %q want generic_agents", got[0].Definition.Key)
	}
}

func TestProviderRegistryService_List_RepoErrorWrapped(t *testing.T) {
	repo := &mockProviderRegistryRepo{err: errors.New("db gone")}
	svc := NewProviderRegistryService(repo, &mockProviderOverrideRepo{}, &mockProviderUserSettingsRepo{})

	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *domain.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *domain.AppError, got %T: %v", err, err)
	}
	if appErr.Code != domain.CodeDatabase {
		t.Errorf("error code: got %q want database_error", appErr.Code)
	}
}

func TestProviderRegistryService_List_EmptyIsNotNil(t *testing.T) {
	svc := makeSvc([]domain.ProviderRegistryEntry{}, nil)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
}

func TestProviderRegistryService_List_BuiltinSourceStamped(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeTestEntry("claude", "experimental")}
	svc := makeSvc(entries, nil)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got[0].Candidates) == 0 {
		t.Fatal("expected candidates")
	}
	if got[0].Candidates[0].Source != "builtin" {
		t.Errorf("Source: got %q want builtin", got[0].Candidates[0].Source)
	}
}

func TestProviderRegistryService_List_MergesOverride(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeTestEntry("claude", "experimental")}
	overrides := []domain.ProviderPathOverride{
		{ProviderDefinitionID: 1, Scope: "project", Purpose: "detect", Paths: []string{".custom-claude"}},
	}
	svc := makeSvc(entries, overrides)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("entry count: got %d want 1", len(got))
	}

	var overrideCand *domain.ProviderPathCandidate
	for i := range got[0].Candidates {
		if got[0].Candidates[i].Source == "override" {
			overrideCand = &got[0].Candidates[i]
			break
		}
	}
	if overrideCand == nil {
		t.Fatal("no override candidate found")
	}
	if overrideCand.RelativePath != ".custom-claude" {
		t.Errorf("RelativePath: got %q want .custom-claude", overrideCand.RelativePath)
	}
}

func TestProviderRegistryService_List_BuiltinPreservedForNonOverriddenSlot(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeTestEntry("claude", "experimental")}
	// Override "skills" slot but not "detect"
	overrides := []domain.ProviderPathOverride{
		{ProviderDefinitionID: 1, Scope: "project", Purpose: "skills", Paths: []string{".custom/skills"}},
	}
	svc := makeSvc(entries, overrides)

	got, _ := svc.List(context.Background())
	// "detect" slot should still be builtin
	var detectCand *domain.ProviderPathCandidate
	for i := range got[0].Candidates {
		if got[0].Candidates[i].Purpose == "detect" {
			detectCand = &got[0].Candidates[i]
			break
		}
	}
	if detectCand == nil {
		t.Fatal("no detect candidate found after skills override")
	}
	if detectCand.Source != "builtin" {
		t.Errorf("detect candidate source: got %q want builtin", detectCand.Source)
	}
}

// -- UpdatePaths tests --

func TestProviderRegistryService_UpdatePaths_Success(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeTestEntry("claude", "experimental")}
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{entries: entries}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{".custom"})
	if err != nil {
		t.Fatalf("UpdatePaths: %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_UnknownProvider(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "no_such", "project", "detect", []string{".path"})
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_InvalidScope(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "invalid_scope", "detect", []string{".path"})
	if err == nil {
		t.Fatal("expected error for invalid scope")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_InvalidPurpose(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "project", "invalid_purpose", []string{".path"})
	if err == nil {
		t.Fatal("expected error for invalid purpose")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_ProjectPathWithDotDot(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{"../escape"})
	if err == nil {
		t.Fatal("expected error for path with ..")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_ProjectPathAbsolute(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{"/absolute"})
	if err == nil {
		t.Fatal("expected error for absolute project path")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_GlobalPathNoTilde(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "global", "skills", []string{"relative/path"})
	if err == nil {
		t.Fatal("expected error for global path without / or ~/")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_GlobalAbsolutePathAllowed(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "global", "skills", []string{"/usr/local/claude/skills"})
	if err != nil {
		t.Fatalf("expected no error for absolute global path, got: %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_GlobalTildePathAllowed(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "global", "skills", []string{"~/.claude/skills"})
	if err != nil {
		t.Fatalf("expected no error for tilde global path, got: %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_EmptyPaths(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{})
	if err == nil {
		t.Fatal("expected error for empty paths")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

// -- ResetPaths tests --

func TestProviderRegistryService_ResetPaths_ExistingOverride(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{
		idByKey:   map[string]int64{"claude": 1},
		deleteRet: true,
	}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	reset, err := svc.ResetPaths(context.Background(), "claude", "project", "detect")
	if err != nil {
		t.Fatalf("ResetPaths: %v", err)
	}
	if !reset {
		t.Error("expected reset=true when override existed")
	}
}

func TestProviderRegistryService_ResetPaths_NoOverride(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{
		idByKey:   map[string]int64{"claude": 1},
		deleteRet: false,
	}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	reset, err := svc.ResetPaths(context.Background(), "claude", "project", "detect")
	if err != nil {
		t.Fatalf("ResetPaths: %v", err)
	}
	if reset {
		t.Error("expected reset=false when no override existed")
	}
}

func TestProviderRegistryService_ResetPaths_UnknownProvider(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	_, err := svc.ResetPaths(context.Background(), "no_such", "project", "detect")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_ResetPaths_InvalidScope(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo, &mockProviderUserSettingsRepo{})

	_, err := svc.ResetPaths(context.Background(), "claude", "bad_scope", "detect")
	if err == nil {
		t.Fatal("expected error for bad scope")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

// -- GlobalPaths tests --

// makeGlobalEntry builds a registry entry with has_global_level and optional global candidates.
func makeGlobalEntry(key string, hasGlobal bool, detectRel, skillsRel string) domain.ProviderRegistryEntry {
	iconKey := key
	e := domain.ProviderRegistryEntry{
		Definition: domain.ProviderDefinition{
			ID:             1,
			Key:            key,
			DisplayName:    key,
			ProviderType:   key,
			IconKey:        &iconKey,
			Status:         domain.ProviderStatusSupported,
			HasGlobalLevel: hasGlobal,
		},
	}
	if detectRel != "" {
		e.Candidates = append(e.Candidates, domain.ProviderPathCandidate{
			RelativePath: detectRel, Scope: "global", Purpose: "detect", Priority: 10, VerificationStatus: "assumed",
		})
	}
	if skillsRel != "" {
		e.Candidates = append(e.Candidates, domain.ProviderPathCandidate{
			RelativePath: skillsRel, Scope: "global", Purpose: "skills", Priority: 10, VerificationStatus: "assumed",
		})
	}
	return e
}

// TestGlobalPaths_BuiltinCandidates returns the seeded global detect/skills paths.
func TestGlobalPaths_BuiltinCandidates(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{
		makeGlobalEntry("generic_agents", true, "~/.agents", "~/.agents/skills"),
	}
	svc := makeSvc(entries, nil)

	got, err := svc.GlobalPaths(context.Background())
	if err != nil {
		t.Fatalf("GlobalPaths: %v", err)
	}
	p, ok := got["generic_agents"]
	if !ok {
		t.Fatal("expected generic_agents in result")
	}
	if p.DetectRel != "~/.agents" {
		t.Errorf("DetectRel: got %q want ~/.agents", p.DetectRel)
	}
	if p.SkillsRel != "~/.agents/skills" {
		t.Errorf("SkillsRel: got %q want ~/.agents/skills", p.SkillsRel)
	}
}

// TestGlobalPaths_GlobalOverride replaces the builtin skills path.
func TestGlobalPaths_GlobalOverride(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{
		makeGlobalEntry("generic_agents", true, "~/.agents", "~/.agents/skills"),
	}
	overrides := []domain.ProviderPathOverride{
		{ProviderDefinitionID: 1, Scope: "global", Purpose: "skills", Paths: []string{"/custom/skills"}},
	}
	svc := makeSvc(entries, overrides)

	got, err := svc.GlobalPaths(context.Background())
	if err != nil {
		t.Fatalf("GlobalPaths: %v", err)
	}
	p := got["generic_agents"]
	if p.SkillsRel != "/custom/skills" {
		t.Errorf("SkillsRel: got %q want /custom/skills", p.SkillsRel)
	}
	// Detect path not overridden — should still use builtin.
	if p.DetectRel != "~/.agents" {
		t.Errorf("DetectRel: got %q want ~/.agents", p.DetectRel)
	}
}

// TestGlobalPaths_NoGlobalLevel excludes providers with has_global_level=false.
func TestGlobalPaths_NoGlobalLevel(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{
		makeGlobalEntry("codex", false, "", ""),
	}
	svc := makeSvc(entries, nil)

	got, err := svc.GlobalPaths(context.Background())
	if err != nil {
		t.Fatalf("GlobalPaths: %v", err)
	}
	if _, ok := got["codex"]; ok {
		t.Error("codex should not appear in GlobalPaths (has_global_level=false)")
	}
}

// TestGlobalPaths_MissingCandidates_Excluded guards against a provider that has has_global_level=true
// but no seeded global detect/skills candidates. Without this guard, expandGlobalPath("") would
// return homeDir causing the adapter to scan the home directory itself.
func TestGlobalPaths_MissingCandidates_Excluded(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{
		makeGlobalEntry("future_provider", true, "", ""),
	}
	svc := makeSvc(entries, nil)

	got, err := svc.GlobalPaths(context.Background())
	if err != nil {
		t.Fatalf("GlobalPaths: %v", err)
	}
	if _, ok := got["future_provider"]; ok {
		t.Error("future_provider should be excluded: has_global_level=true but no global candidates seeded")
	}
}

// TestGlobalPaths_PartialCandidates_Excluded guards the case where only one of detect/skills
// candidates is seeded. Both are required for a well-formed global location.
func TestGlobalPaths_PartialCandidates_Excluded(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{
		makeGlobalEntry("half_provider", true, "~/.half", ""),
	}
	svc := makeSvc(entries, nil)

	got, err := svc.GlobalPaths(context.Background())
	if err != nil {
		t.Fatalf("GlobalPaths: %v", err)
	}
	if _, ok := got["half_provider"]; ok {
		t.Error("half_provider should be excluded: skills candidate missing")
	}
}

// -- IsEnabled / CanToggle derivation tests --

func makeEntryWithID(id int64, key, status string) domain.ProviderRegistryEntry {
	iconKey := key
	return domain.ProviderRegistryEntry{
		Definition: domain.ProviderDefinition{
			ID:           id,
			Key:          key,
			DisplayName:  key,
			ProviderType: key,
			IconKey:      &iconKey,
			Status:       domain.ProviderStatus(status),
		},
	}
}

func TestProviderRegistryService_List_IsEnabled_DefaultsFromStatus(t *testing.T) {
	cases := []struct {
		status    string
		wantEnabled bool
		wantCanToggle bool
	}{
		{"supported", true, true},
		{"experimental", true, true},
		{"unsupported", false, false},
		{"disabled", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.status, func(t *testing.T) {
			entries := []domain.ProviderRegistryEntry{makeEntryWithID(1, "prov", tc.status)}
			svc := makeSvc(entries, nil)

			got, err := svc.List(context.Background())
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if got[0].IsEnabled != tc.wantEnabled {
				t.Errorf("IsEnabled: got %v want %v (status=%s)", got[0].IsEnabled, tc.wantEnabled, tc.status)
			}
			if got[0].CanToggle != tc.wantCanToggle {
				t.Errorf("CanToggle: got %v want %v (status=%s)", got[0].CanToggle, tc.wantCanToggle, tc.status)
			}
		})
	}
}

func TestProviderRegistryService_List_IsEnabled_UserSettingOverridesDefault(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(42, "claude", "experimental")}
	repo := &mockProviderRegistryRepo{entries: entries}
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 42}}
	userSettingsRepo := &mockProviderUserSettingsRepo{
		settings: []domain.ProviderUserSetting{{ID: 1, ProviderDefinitionID: 42, Enabled: false}},
	}
	svc := NewProviderRegistryService(repo, overrideRepo, userSettingsRepo)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got[0].IsEnabled {
		t.Error("IsEnabled should be false when user setting is false, even for experimental provider")
	}
}

func TestProviderRegistryService_List_IsEnabled_UserEnableOverridesDefault(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(7, "prov", "supported")}
	repo := &mockProviderRegistryRepo{entries: entries}
	overrideRepo := &mockProviderOverrideRepo{}
	userSettingsRepo := &mockProviderUserSettingsRepo{
		settings: []domain.ProviderUserSetting{{ID: 1, ProviderDefinitionID: 7, Enabled: true}},
	}
	svc := NewProviderRegistryService(repo, overrideRepo, userSettingsRepo)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !got[0].IsEnabled {
		t.Error("IsEnabled should be true when user setting is true")
	}
}

// -- SetEnabled tests --

func makeSvcWithUserSettings(
	entries []domain.ProviderRegistryEntry,
	userSettings []domain.ProviderUserSetting,
) (*ProviderRegistryService, *mockProviderUserSettingsRepo) {
	repo := &mockProviderRegistryRepo{entries: entries}
	overrideRepo := &mockProviderOverrideRepo{}
	us := &mockProviderUserSettingsRepo{settings: userSettings}
	return NewProviderRegistryService(repo, overrideRepo, us), us
}

func TestProviderRegistryService_SetEnabled_PersistsTrue(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(1, "claude", "experimental")}
	svc, usRepo := makeSvcWithUserSettings(entries, nil)

	if err := svc.SetEnabled(context.Background(), "claude", true); err != nil {
		t.Fatalf("SetEnabled(true): %v", err)
	}
	// The mock just doesn't error; verify no error is the success signal.
	_ = usRepo
}

func TestProviderRegistryService_SetEnabled_PersistsFalse(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(1, "generic_agents", "supported")}
	svc, _ := makeSvcWithUserSettings(entries, nil)

	if err := svc.SetEnabled(context.Background(), "generic_agents", false); err != nil {
		t.Fatalf("SetEnabled(false): %v", err)
	}
}

func TestProviderRegistryService_SetEnabled_UnknownProvider(t *testing.T) {
	svc, _ := makeSvcWithUserSettings(nil, nil)

	err := svc.SetEnabled(context.Background(), "no_such_provider", true)
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_SetEnabled_EnableUnsupportedRejected(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(1, "prov", "unsupported")}
	svc, _ := makeSvcWithUserSettings(entries, nil)

	err := svc.SetEnabled(context.Background(), "prov", true)
	if err == nil {
		t.Fatal("expected error when enabling unsupported provider")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_SetEnabled_DisableUnsupportedAllowed(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(1, "prov", "unsupported")}
	svc, _ := makeSvcWithUserSettings(entries, nil)

	// Disabling an already-disabled provider is fine (no-op in practice).
	if err := svc.SetEnabled(context.Background(), "prov", false); err != nil {
		t.Fatalf("SetEnabled(false) for unsupported should be allowed: %v", err)
	}
}

func TestProviderRegistryService_SetEnabled_EmptyKeyRejected(t *testing.T) {
	svc, _ := makeSvcWithUserSettings(nil, nil)

	err := svc.SetEnabled(context.Background(), "", true)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}

func TestProviderRegistryService_List_IsEnabled_ClampedForUnsupportedWithStaleStoredSetting(t *testing.T) {
	// Regression: if a provider was previously supported+enabled then later became unsupported,
	// a stale user setting of enabled=true must not leak through as isEnabled=true.
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(5, "prov", "unsupported")}
	repo := &mockProviderRegistryRepo{entries: entries}
	overrideRepo := &mockProviderOverrideRepo{}
	userSettingsRepo := &mockProviderUserSettingsRepo{
		settings: []domain.ProviderUserSetting{{ID: 1, ProviderDefinitionID: 5, Enabled: true}},
	}
	svc := NewProviderRegistryService(repo, overrideRepo, userSettingsRepo)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got[0].IsEnabled {
		t.Error("IsEnabled must be false for unsupported provider even when stale user setting says enabled=true")
	}
	if got[0].CanToggle {
		t.Error("CanToggle must be false for unsupported provider")
	}
}

func TestProviderRegistryService_List_IsEnabled_ClampedForDisabledWithStaleStoredSetting(t *testing.T) {
	entries := []domain.ProviderRegistryEntry{makeEntryWithID(6, "prov", "disabled")}
	repo := &mockProviderRegistryRepo{entries: entries}
	overrideRepo := &mockProviderOverrideRepo{}
	userSettingsRepo := &mockProviderUserSettingsRepo{
		settings: []domain.ProviderUserSetting{{ID: 1, ProviderDefinitionID: 6, Enabled: true}},
	}
	svc := NewProviderRegistryService(repo, overrideRepo, userSettingsRepo)

	got, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got[0].IsEnabled {
		t.Error("IsEnabled must be false for disabled provider even when stale user setting says enabled=true")
	}
}
