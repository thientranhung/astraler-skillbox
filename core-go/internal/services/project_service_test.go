package services

import (
	"context"
	"errors"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

func newProjectSvc(fs *mockProjectFS, projRepo *mockProjectRepo) *ProjectService {
	return NewProjectService(
		projRepo,
		&mockProjectProviderRepo{byProject: make(map[int64][]domain.ProjectProviderSummary)},
		&mockProjectWarningRepo{},
		&mockProjectInstallRepo{},
		fs,
	)
}

// --- AddProject ---

func TestAddProject_HappyPath(t *testing.T) {
	projRepo := newMockProjectRepo()
	svc := newProjectSvc(&mockProjectFS{}, projRepo)

	result, err := svc.AddProject(context.Background(), "/tmp/my-project")
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if result.ProjectID <= 0 {
		t.Error("expected positive projectID")
	}
	if result.Name != "my-project" {
		t.Errorf("name: got %q want my-project", result.Name)
	}
	if result.Path != "/tmp/my-project" {
		t.Errorf("path: got %q want /tmp/my-project", result.Path)
	}
	if result.Status != domain.ProjectStatusActive {
		t.Errorf("status: got %q want active", result.Status)
	}
}

func TestAddProject_NormalizesTrailingSlash(t *testing.T) {
	svc := newProjectSvc(&mockProjectFS{}, newMockProjectRepo())
	result, err := svc.AddProject(context.Background(), "/tmp/my-project/")
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if result.Path != "/tmp/my-project" {
		t.Errorf("expected normalized path, got %q", result.Path)
	}
}

func TestAddProject_Idempotent(t *testing.T) {
	projRepo := newMockProjectRepo()
	svc := newProjectSvc(&mockProjectFS{}, projRepo)

	r1, err1 := svc.AddProject(context.Background(), "/tmp/proj")
	r2, err2 := svc.AddProject(context.Background(), "/tmp/proj")
	if err1 != nil || err2 != nil {
		t.Fatalf("AddProject errors: %v %v", err1, err2)
	}
	if r1.ProjectID != r2.ProjectID {
		t.Errorf("idempotent: IDs differ %d vs %d", r1.ProjectID, r2.ProjectID)
	}
}

func TestAddProject_ValidationError_PathNotFound(t *testing.T) {
	fs := &mockProjectFS{validateErr: &filesystem.FilesystemError{
		Code: filesystem.ErrPathNotFound, Path: "/bad", Message: "not found",
	}}
	svc := newProjectSvc(fs, newMockProjectRepo())

	_, err := svc.AddProject(context.Background(), "/bad")
	requireAppError(t, err, domain.CodeValidation)
}

func TestAddProject_ValidationError_NotDirectory(t *testing.T) {
	fs := &mockProjectFS{validateErr: &filesystem.FilesystemError{
		Code: filesystem.ErrNotADirectory, Path: "/bad/file", Message: "not a directory",
	}}
	svc := newProjectSvc(fs, newMockProjectRepo())

	_, err := svc.AddProject(context.Background(), "/bad/file")
	requireAppError(t, err, domain.CodeValidation)
}

func TestAddProject_ValidationError_NormalizeFailure(t *testing.T) {
	fs := &mockProjectFS{normalizeErr: errors.New("not absolute")}
	svc := newProjectSvc(fs, newMockProjectRepo())

	_, err := svc.AddProject(context.Background(), "relative/path")
	requireAppError(t, err, domain.CodeValidation)
}

func TestAddProject_DatabaseError(t *testing.T) {
	projRepo := newMockProjectRepo()
	projRepo.upsertErr = errors.New("db full")
	svc := newProjectSvc(&mockProjectFS{}, projRepo)

	_, err := svc.AddProject(context.Background(), "/tmp/proj")
	requireAppError(t, err, domain.CodeDatabase)
}

// --- ListProjects ---

func TestListProjects_Empty(t *testing.T) {
	svc := newProjectSvc(&mockProjectFS{}, newMockProjectRepo())
	items, err := svc.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected empty, got %d items", len(items))
	}
}

func TestListProjects_SkillCountIsSumOfEntries(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a") //nolint:errcheck

	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProjectProviderID: 1, ProviderKey: "generic_agents", EntryCount: 3}},
		},
	}
	warnRepo := &mockProjectWarningRepo{counts: map[int64]int{1: 2}}

	svc := NewProjectService(projRepo, ppRepo, warnRepo, &mockProjectInstallRepo{}, &mockProjectFS{})
	items, err := svc.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].SkillCount != 3 {
		t.Errorf("SkillCount: got %d want 3", items[0].SkillCount)
	}
	if items[0].WarningCount != 2 {
		t.Errorf("WarningCount: got %d want 2", items[0].WarningCount)
	}
	if len(items[0].Providers) != 1 {
		t.Errorf("providers: got %d want 1", len(items[0].Providers))
	}
}

func TestListProjects_MultipleProviders_SkillCountSummed(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a") //nolint:errcheck

	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {
				{ProjectProviderID: 1, ProviderKey: "generic_agents", EntryCount: 4},
				{ProjectProviderID: 2, ProviderKey: "other", EntryCount: 2},
			},
		},
	}

	svc := NewProjectService(projRepo, ppRepo, &mockProjectWarningRepo{}, &mockProjectInstallRepo{}, &mockProjectFS{})
	items, err := svc.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if items[0].SkillCount != 6 {
		t.Errorf("SkillCount: got %d want 6", items[0].SkillCount)
	}
}

func TestListProjects_DatabaseError(t *testing.T) {
	projRepo := newMockProjectRepo()
	projRepo.listErr = errors.New("db unavailable")
	svc := newProjectSvc(&mockProjectFS{}, projRepo)

	_, err := svc.ListProjects(context.Background())
	requireAppError(t, err, domain.CodeDatabase)
}

// --- GetProject ---

func TestGetProject_NotFound(t *testing.T) {
	svc := newProjectSvc(&mockProjectFS{}, newMockProjectRepo())
	_, err := svc.GetProject(context.Background(), 999)
	requireAppError(t, err, domain.CodeValidation)
}

func TestGetProject_HappyPath(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a") //nolint:errcheck

	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProjectProviderID: 1, ProviderKey: "generic_agents", EntryCount: 2}},
		},
	}
	installRepo := &mockProjectInstallRepo{
		byProject: map[int64][]domain.Install{
			1: {
				{ID: 1, ProjectProviderID: 1, SkillName: "tool-a", InstallMode: domain.InstallModeSymlink, InstallStatus: domain.InstallStatusCurrent},
				{ID: 2, ProjectProviderID: 1, SkillName: "tool-b", InstallMode: domain.InstallModeDirect, InstallStatus: domain.InstallStatusCurrent},
			},
		},
	}
	rescan := "rescan"
	warnRepo := &mockProjectWarningRepo{
		warnings: map[int64][]domain.Warning{
			1: {{ID: 1, Code: "broken_symlink", Severity: domain.WarningSeverityWarning, ActionKey: &rescan}},
		},
	}

	svc := NewProjectService(projRepo, ppRepo, warnRepo, installRepo, &mockProjectFS{})
	view, err := svc.GetProject(ctx, 1)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if view.Project.ID != 1 {
		t.Errorf("project ID: got %d want 1", view.Project.ID)
	}
	if len(view.Providers) != 1 {
		t.Errorf("providers: got %d want 1", len(view.Providers))
	}
	if len(view.Entries) != 2 {
		t.Errorf("entries: got %d want 2", len(view.Entries))
	}
	if len(view.Warnings) != 1 {
		t.Errorf("warnings: got %d want 1", len(view.Warnings))
	}
}

func TestGetProject_FreshProject_EmptyProviders(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj-b", "/tmp/proj-b") //nolint:errcheck

	svc := newProjectSvc(&mockProjectFS{}, projRepo)
	view, err := svc.GetProject(ctx, 1)
	if err != nil {
		t.Fatalf("GetProject: %v", err)
	}
	if len(view.Providers) != 0 {
		t.Errorf("expected no providers for fresh project, got %d", len(view.Providers))
	}
	if len(view.Entries) != 0 {
		t.Errorf("expected no entries, got %d", len(view.Entries))
	}
	if len(view.Warnings) != 0 {
		t.Errorf("expected no warnings, got %d", len(view.Warnings))
	}
}

// --- RemoveProject ---

func TestRemoveProject_HappyPath(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a") //nolint:errcheck

	svc := newProjectSvc(&mockProjectFS{}, projRepo)
	result, err := svc.RemoveProject(ctx, 1)
	if err != nil {
		t.Fatalf("RemoveProject: %v", err)
	}
	if !result.Removed {
		t.Error("expected Removed=true")
	}
}

func TestRemoveProject_NotFound_ValidationError(t *testing.T) {
	svc := newProjectSvc(&mockProjectFS{}, newMockProjectRepo())
	_, err := svc.RemoveProject(context.Background(), 999)
	requireAppError(t, err, domain.CodeValidation)
}

func TestRemoveProject_AlreadyRemoved_ValidationError(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()
	projRepo.UpsertByPath(ctx, "proj-a", "/tmp/proj-a") //nolint:errcheck

	svc := newProjectSvc(&mockProjectFS{}, projRepo)
	if _, err := svc.RemoveProject(ctx, 1); err != nil {
		t.Fatalf("first RemoveProject: %v", err)
	}

	_, err := svc.RemoveProject(ctx, 1)
	requireAppError(t, err, domain.CodeValidation)
}

func TestAddProject_AfterRemove_RevivesProject(t *testing.T) {
	projRepo := newMockProjectRepo()
	ctx := context.Background()

	svc := newProjectSvc(&mockProjectFS{}, projRepo)
	r1, _ := svc.AddProject(ctx, "/tmp/proj-a")
	svc.RemoveProject(ctx, r1.ProjectID) //nolint:errcheck

	r2, err := svc.AddProject(ctx, "/tmp/proj-a")
	if err != nil {
		t.Fatalf("AddProject after remove: %v", err)
	}
	if r2.ProjectID != r1.ProjectID {
		t.Errorf("expected same project ID on revival: got %d want %d", r2.ProjectID, r1.ProjectID)
	}
	if r2.Status != domain.ProjectStatusActive {
		t.Errorf("status: got %q want active", r2.Status)
	}
}

// --- helpers ---

func requireAppError(t *testing.T, err error, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	ae, ok := err.(*domain.AppError)
	if !ok {
		t.Fatalf("expected *domain.AppError, got %T: %v", err, err)
	}
	if ae.Code != wantCode {
		t.Errorf("error code: got %q want %q", ae.Code, wantCode)
	}
}
