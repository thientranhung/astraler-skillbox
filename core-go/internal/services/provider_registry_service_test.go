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
	return NewProviderRegistryService(repo, overrideRepo)
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
	svc := NewProviderRegistryService(repo, &mockProviderOverrideRepo{})

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{entries: entries}, overrideRepo)

	err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{".custom"})
	if err != nil {
		t.Fatalf("UpdatePaths: %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_UnknownProvider(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

	err := svc.UpdatePaths(context.Background(), "claude", "global", "skills", []string{"/usr/local/claude/skills"})
	if err != nil {
		t.Fatalf("expected no error for absolute global path, got: %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_GlobalTildePathAllowed(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

	err := svc.UpdatePaths(context.Background(), "claude", "global", "skills", []string{"~/.claude/skills"})
	if err != nil {
		t.Fatalf("expected no error for tilde global path, got: %v", err)
	}
}

func TestProviderRegistryService_UpdatePaths_EmptyPaths(t *testing.T) {
	overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

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
	svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

	_, err := svc.ResetPaths(context.Background(), "claude", "bad_scope", "detect")
	if err == nil {
		t.Fatal("expected error for bad scope")
	}
	var ae *domain.AppError
	if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
		t.Errorf("expected validation_error, got %v", err)
	}
}
