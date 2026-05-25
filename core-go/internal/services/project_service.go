package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
)

// AddProjectResult is returned by AddProject.
type AddProjectResult struct {
	ProjectID int64
	Name      string
	Path      string
	Status    domain.ProjectStatus
}

// ProjectListItem is one row in the projects list response.
type ProjectListItem struct {
	ID            int64
	Name          string
	Path          string
	Status        domain.ProjectStatus
	Providers     []domain.ProjectProviderSummary
	SkillCount    int
	WarningCount  int
	LastScannedAt *time.Time
}

// ProjectDetailView is the full project detail response.
type ProjectDetailView struct {
	Project   domain.Project
	Providers []domain.ProjectProviderSummary
	Entries   []domain.Install
	Warnings  []domain.Warning
}

// ProjectService handles project operations (add, list, detail, scan).
type ProjectService struct {
	projectRepo ProjectRepo
	ppRepo      ProjectProviderRepo
	warningRepo ProjectWarningRepo
	installRepo ProjectInstallRepo
	fs          ProjectFilesystem
	// scan dependencies — nil until WithScanDeps is called
	runner   OperationRunner
	scanRepo ProjectScanCommitter
}

// NewProjectService constructs a ProjectService for read/add operations.
func NewProjectService(
	projectRepo ProjectRepo,
	ppRepo ProjectProviderRepo,
	warningRepo ProjectWarningRepo,
	installRepo ProjectInstallRepo,
	fs ProjectFilesystem,
) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
		ppRepo:      ppRepo,
		warningRepo: warningRepo,
		installRepo: installRepo,
		fs:          fs,
	}
}

// WithScanDeps attaches the dependencies required for ScanProject.
// Returns the receiver to allow chaining with NewProjectService.
func (s *ProjectService) WithScanDeps(runner OperationRunner, scanRepo ProjectScanCommitter) *ProjectService {
	s.runner = runner
	s.scanRepo = scanRepo
	return s
}

// AddProject validates path, normalizes it, and persists the project idempotently.
func (s *ProjectService) AddProject(ctx context.Context, path string) (*AddProjectResult, error) {
	normalized, err := s.fs.NormalizeAbs(path)
	if err != nil {
		return nil, domain.NewValidationError("Invalid project path", err.Error())
	}

	if err := s.fs.ValidateProjectPath(normalized); err != nil {
		fe, ok := err.(*filesystem.FilesystemError)
		if ok {
			return nil, domain.NewValidationError(
				"Invalid project folder",
				string(fe.Code)+": "+fe.Message,
			)
		}
		return nil, domain.NewValidationError("Invalid project folder", err.Error())
	}

	name := filepath.Base(normalized)
	projectID, _, err := s.projectRepo.UpsertByPath(ctx, name, normalized)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not persist project", err.Error())
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil || project == nil {
		msg := "project not found after upsert"
		if err != nil {
			msg = err.Error()
		}
		return nil, domain.NewDatabaseError("Could not fetch project", msg)
	}

	return &AddProjectResult{
		ProjectID: project.ID,
		Name:      project.Name,
		Path:      project.Path,
		Status:    project.Status,
	}, nil
}

// ListProjects returns all projects with per-project provider summaries, skill count, and warning count.
func (s *ProjectService) ListProjects(ctx context.Context) ([]ProjectListItem, error) {
	projects, err := s.projectRepo.List(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list projects", err.Error())
	}

	items := make([]ProjectListItem, 0, len(projects))
	for _, p := range projects {
		providers, err := s.ppRepo.ListByProject(ctx, p.ID)
		if err != nil {
			return nil, domain.NewDatabaseError("Could not list project providers", err.Error())
		}

		skillCount := 0
		for _, pp := range providers {
			skillCount += pp.EntryCount
		}

		warningCount, err := s.warningRepo.CountActiveForProject(ctx, p.ID)
		if err != nil {
			return nil, domain.NewDatabaseError("Could not count warnings", err.Error())
		}

		items = append(items, ProjectListItem{
			ID:            p.ID,
			Name:          p.Name,
			Path:          p.Path,
			Status:        p.Status,
			Providers:     providers,
			SkillCount:    skillCount,
			WarningCount:  warningCount,
			LastScannedAt: p.LastScannedAt,
		})
	}
	return items, nil
}

// GetProject returns the full project detail view or validation_error if not found.
func (s *ProjectService) GetProject(ctx context.Context, projectID int64) (*ProjectDetailView, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not fetch project", err.Error())
	}
	if project == nil {
		return nil, domain.NewValidationError(
			"Project not found",
			fmt.Sprintf("projectId %d does not exist", projectID),
		)
	}

	providers, err := s.ppRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list providers", err.Error())
	}

	entries, err := s.installRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list entries", err.Error())
	}

	warnings, err := s.warningRepo.ListActiveForProject(ctx, projectID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list warnings", err.Error())
	}

	return &ProjectDetailView{
		Project:   *project,
		Providers: providers,
		Entries:   entries,
		Warnings:  warnings,
	}, nil
}

// ScanProject queues an async scan operation for the given project.
// Returns conflict_error if a scan is already running for this project.
func (s *ProjectService) ScanProject(ctx context.Context, projectID int64) (int64, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return 0, domain.NewDatabaseError("Could not fetch project", err.Error())
	}
	if project == nil {
		return 0, domain.NewValidationError(
			"Project not found",
			fmt.Sprintf("projectId %d does not exist", projectID),
		)
	}

	target := operations.Target{Type: "project", ID: projectID}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeScan,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return s.scanProjectInternal(opCtx, project, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not queue scan operation", err.Error())
	}
	return opID, nil
}

func (s *ProjectService) scanProjectInternal(
	ctx context.Context,
	project *domain.Project,
	progress operations.ProgressFn,
) (any, error) {
	progress("reading_project", 0, 0, "")

	if err := s.fs.ValidateProjectPath(project.Path); err != nil {
		return s.commitTerminalPath(ctx, project, err)
	}

	// M3c2b2: provider detection, classification, commit scan — not yet implemented.
	return nil, nil
}

func (s *ProjectService) commitTerminalPath(
	ctx context.Context,
	project *domain.Project,
	pathErr error,
) (any, error) {
	status := domain.ProjectStatusUnreadable
	warnCode := "project_unreadable"
	warnMsg := "Project folder is unreadable: " + project.Path

	if fe, ok := pathErr.(*filesystem.FilesystemError); ok && fe.Code == filesystem.ErrPathNotFound {
		status = domain.ProjectStatusMissing
		warnCode = "project_missing"
		warnMsg = "Project folder not found: " + project.Path
	}

	rescan := "rescan"
	warning := domain.Warning{
		ScopeType: domain.WarningScopeProject,
		Severity:  domain.WarningSeverityWarning,
		Code:      warnCode,
		Message:   warnMsg,
		ActionKey: &rescan,
	}

	if err := s.scanRepo.CommitProjectTerminal(ctx, project.ID, status, &warning, time.Now()); err != nil {
		return nil, domain.NewDatabaseError("Could not commit terminal scan state", err.Error())
	}
	return nil, nil
}
