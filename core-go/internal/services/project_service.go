package services

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
)

// ProjectPluginScanner scans a project's plugin settings layers within the caller's
// operation context (no new operation). Implemented by *ProviderPluginService.
type ProjectPluginScanner interface {
	ScanProjectLayers(ctx context.Context, project *domain.Project, progress operations.ProgressFn) error
}

// ProjectPluginCounter returns per-project effective plugin counts.
// Implemented by *ProviderPluginService.
type ProjectPluginCounter interface {
	PluginCountsByProject(ctx context.Context) (map[int64]domain.PluginCount, error)
}

// ProjectRemoveResult is returned by RemoveProject.
type ProjectRemoveResult struct {
	Removed bool
}

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
	SkillCount         int
	WarningCount       int
	LastScannedAt      *time.Time
	PluginEnabledCount int
	PluginTotalCount   int
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
	// provider-detection scan deps — nil until WithProviderDeps is called
	providerRegistry   ProviderRegistry
	providerDefRepo    ProviderDefinitionRepo
	hostLister         SkillHostLister
	skillsByHostLister SkillsByHostLister
	// pathResolver resolves effective project-scope detect/skills paths (override ?? builtin).
	pathResolver ProjectProviderPathResolver
	// install deps — nil until WithInstallDeps is called
	installFS        InstallFilesystem
	activeHostReader ActiveHostReader
	// installSkillReader is separate from skillsByHostLister (scan) to avoid silent overwrite.
	installSkillReader SkillsByHostLister
	// remove deps — nil until WithRemoveDeps is called
	removeFS       RemoveFilesystem
	installDeleter RemoveInstallDeleter
	// plugin deps — nil until WithPluginDeps is called
	pluginScanner ProjectPluginScanner
	pluginCounter ProjectPluginCounter
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

// WithScanDeps attaches the operation runner and scan committer required for ScanProject.
// Returns the receiver to allow chaining.
func (s *ProjectService) WithScanDeps(runner OperationRunner, scanRepo ProjectScanCommitter) *ProjectService {
	s.runner = runner
	s.scanRepo = scanRepo
	return s
}

// WithProviderDeps attaches the provider-detection dependencies required for full project scans.
// Returns the receiver to allow chaining.
func (s *ProjectService) WithProviderDeps(
	registry ProviderRegistry,
	pdRepo ProviderDefinitionRepo,
	hostLister SkillHostLister,
	skillsByHostLister SkillsByHostLister,
) *ProjectService {
	s.providerRegistry = registry
	s.providerDefRepo = pdRepo
	s.hostLister = hostLister
	s.skillsByHostLister = skillsByHostLister
	return s
}

// WithPathResolver attaches the effective-path resolver used by scan and install.
// Returns the receiver to allow chaining.
func (s *ProjectService) WithPathResolver(resolver ProjectProviderPathResolver) *ProjectService {
	s.pathResolver = resolver
	return s
}

// WithInstallDeps attaches the filesystem, active host reader, and host skill reader
// required for InstallSkills. Returns the receiver to allow chaining.
func (s *ProjectService) WithInstallDeps(
	installFS InstallFilesystem,
	activeHostReader ActiveHostReader,
	hostSkillReader SkillsByHostLister,
) *ProjectService {
	s.installFS = installFS
	s.activeHostReader = activeHostReader
	s.installSkillReader = hostSkillReader
	return s
}

// WithRemoveDeps attaches the filesystem and install-row deleter required for
// RemoveSkill. Returns the receiver to allow chaining.
func (s *ProjectService) WithRemoveDeps(
	removeFS RemoveFilesystem,
	installDeleter RemoveInstallDeleter,
) *ProjectService {
	s.removeFS = removeFS
	s.installDeleter = installDeleter
	return s
}

// WithPluginDeps attaches the plugin scanner (folded into project scan) and the
// plugin counter (used by ListProjects). Either may be nil. Returns the receiver.
func (s *ProjectService) WithPluginDeps(
	scanner ProjectPluginScanner,
	counter ProjectPluginCounter,
) *ProjectService {
	s.pluginScanner = scanner
	s.pluginCounter = counter
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

	var pluginCounts map[int64]domain.PluginCount
	if s.pluginCounter != nil {
		pluginCounts, err = s.pluginCounter.PluginCountsByProject(ctx)
		if err != nil {
			return nil, domain.NewDatabaseError("Could not count plugins", err.Error())
		}
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
			ID:                 p.ID,
			Name:               p.Name,
			Path:               p.Path,
			Status:             p.Status,
			Providers:          providers,
			SkillCount:         skillCount,
			WarningCount:       warningCount,
			LastScannedAt:      p.LastScannedAt,
			PluginEnabledCount: pluginCounts[p.ID].Enabled,
			PluginTotalCount:   pluginCounts[p.ID].Total,
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

// RemoveProject soft-removes a project by setting its status to removed.
// Returns validation_error if the project does not exist or is already removed,
// and database_error for any underlying persistence failure.
func (s *ProjectService) RemoveProject(ctx context.Context, projectID int64) (*ProjectRemoveResult, error) {
	existing, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not fetch project", err.Error())
	}
	if existing == nil {
		return nil, domain.NewValidationError(
			"Project not found",
			fmt.Sprintf("projectId %d does not exist or is already removed", projectID),
		)
	}

	ok, err := s.projectRepo.MarkRemoved(ctx, projectID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not remove project", err.Error())
	}
	if !ok {
		return nil, domain.NewValidationError(
			"Project not found",
			fmt.Sprintf("projectId %d does not exist or is already removed", projectID),
		)
	}

	return &ProjectRemoveResult{Removed: true}, nil
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

	// ValidateProjectPath uses os.Stat, which succeeds even when the directory
	// cannot be opened. PathInfo additionally tries os.Open to verify readability.
	pi, err := s.fs.PathInfo(project.Path)
	if err != nil || !pi.Readable {
		return s.commitTerminalDirect(ctx, project,
			domain.ProjectStatusUnreadable, "project_unreadable",
			"Project folder is not readable: "+project.Path)
	}

	// M3c2b2: provider detection, entry classification, full scan commit.
	progress("detecting_providers", 0, 0, "")

	hosts, err := s.hostLister.ListAll(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load skill host folders", err.Error())
	}
	hostSummaries, err := s.buildHostSummaries(ctx, hosts)
	if err != nil {
		return nil, err
	}

	// Resolve effective project-scope paths (override ?? builtin) once per scan.
	var effectivePathsMap map[string]providers.ProjectScopePaths
	if s.pathResolver != nil {
		var resolveErr error
		effectivePathsMap, resolveErr = s.pathResolver.ProjectPaths(ctx)
		if resolveErr != nil {
			return nil, resolveErr
		}
	}

	adapters := s.providerRegistry.All()
	providerResults := make([]repositories.ProviderScanResult, 0, len(adapters))
	var projectWarnings []domain.Warning

	for _, adapter := range adapters {
		paths, ok := effectivePathsMap[adapter.Key()]
		if !ok {
			paths = adapter.DefaultProjectPaths()
		}
		result, detectErr := adapter.Detect(project.Path, paths, s.fs)
		if detectErr != nil {
			continue // non-fatal: skip this provider
		}

		pd, err := s.providerDefRepo.GetByKey(ctx, adapter.Key())
		if err != nil {
			return nil, domain.NewDatabaseError("Could not look up provider definition", err.Error())
		}
		if pd == nil {
			continue // provider not seeded in DB; skip
		}

		// Collect provider-scoped warnings. Project-scoped "no_provider_detected" warnings
		// are suppressed here and emitted once after all adapters run (see below).
		rescan := "rescan"
		var providerWarnings []domain.Warning
		for _, aw := range result.Warnings {
			if aw.Code == "no_provider_detected" {
				continue // aggregated below
			}
			w := domain.Warning{
				ScopeType: aw.ScopeType,
				Severity:  aw.Severity,
				Code:      aw.Code,
				Message:   aw.Message,
				ActionKey: &rescan,
			}
			if aw.ScopeType == domain.WarningScopeProject {
				projectWarnings = append(projectWarnings, w)
			} else {
				providerWarnings = append(providerWarnings, w)
			}
		}

		// Provider not detected: do not create a ProviderScanResult.
		if !result.Present {
			continue
		}

		progress("classifying_entries", 0, 0, "")

		installs := make([]repositories.InstallScanResult, 0, len(result.Entries))
		for _, entry := range result.Entries {
			classified := ClassifyAdapterEntry(entry, hostSummaries)
			installs = append(installs, repositories.InstallScanResult{
				SkillID:                   classified.SkillID,
				SkillName:                 entry.Name,
				InstallMode:               classified.Mode,
				InstallStatus:             classified.Status,
				ProjectSkillPath:          entry.Path,
				SourceSkillPath:           classified.SourceSkillPath,
				SymlinkTargetPath:         classified.SymlinkTargetPath,
				InstalledFromHostFolderID: classified.InstalledFromHostFolderID,
				Warning:                   installWarning(classified.Status),
			})
		}

		var detectedPath *string
		if result.DetectedPath != "" {
			dp := result.DetectedPath
			detectedPath = &dp
		}
		var skillsPathPtr *string
		if result.SkillsPath != "" {
			sp := result.SkillsPath
			skillsPathPtr = &sp
		}

		providerResults = append(providerResults, repositories.ProviderScanResult{
			ProviderDefinitionID: pd.ID,
			DetectedPath:         detectedPath,
			SkillsPath:           skillsPathPtr,
			DetectionStatus:      result.DetectionStatus,
			Installs:             installs,
			Warnings:             providerWarnings,
		})
	}

	// Emit one project-level no_provider_detected only when all adapters failed to detect.
	if len(providerResults) == 0 {
		rescan := "rescan"
		projectWarnings = append(projectWarnings, domain.Warning{
			ScopeType: domain.WarningScopeProject,
			Severity:  domain.WarningSeverityWarning,
			Code:      "no_provider_detected",
			Message:   "No provider detected in this project",
			ActionKey: &rescan,
		})
	}

	if err := s.scanRepo.CommitProjectScan(ctx, project.ID, providerResults, projectWarnings, time.Now()); err != nil {
		return nil, domain.NewDatabaseError("Could not commit project scan", err.Error())
	}

	if s.pluginScanner != nil {
		progress("scanning_plugins", 0, 0, "")
		if err := s.pluginScanner.ScanProjectLayers(ctx, project, progress); err != nil {
			// F3: skills already committed — return the summary WITH the error so the runner
			// persists it as operation metadata (partial failure), instead of discarding it.
			return buildScanSummary(providerResults, projectWarnings), err
		}
	}

	progress("done", 0, 0, "")
	return buildScanSummary(providerResults, projectWarnings), nil
}

// projectScanSummary is returned by scanProjectInternal and stored in operations.metadata_json.
type projectScanSummary struct {
	ProvidersFound    int `json:"providersFound"`
	EntriesClassified int `json:"entriesClassified"`
	WarningsCreated   int `json:"warningsCreated"`
}

func buildScanSummary(provs []repositories.ProviderScanResult, projectWarnings []domain.Warning) *projectScanSummary {
	entries := 0
	warnings := len(projectWarnings)
	for _, p := range provs {
		entries += len(p.Installs)
		warnings += len(p.Warnings)
		for _, inst := range p.Installs {
			if inst.Warning != nil {
				warnings++
			}
		}
	}
	return &projectScanSummary{
		ProvidersFound:    len(provs),
		EntriesClassified: entries,
		WarningsCreated:   warnings,
	}
}

// buildHostSummaries loads skills for each host and returns HostSummary slices
// with active hosts first (so ClassifyAdapterEntry resolves active before inactive).
func (s *ProjectService) buildHostSummaries(ctx context.Context, hosts []domain.SkillHostFolder) ([]HostSummary, error) {
	summaries := make([]HostSummary, 0, len(hosts))
	for _, active := range []bool{true, false} {
		for _, h := range hosts {
			isActive := h.Status == domain.SkillHostStatusActive
			if isActive != active {
				continue
			}
			skills, err := s.skillsByHostLister.ListByHost(ctx, h.ID)
			if err != nil {
				return nil, domain.NewDatabaseError("Could not load skills for host", err.Error())
			}
			summaries = append(summaries, HostSummary{
				ID:         h.ID,
				SkillsPath: h.SkillsPath,
				IsActive:   isActive,
				Skills:     skills,
			})
		}
	}
	return summaries, nil
}

// installWarning returns a Warning for install entries that need user attention,
// or nil for healthy entries (current, missing — missing is handled by reconcile).
func installWarning(status domain.InstallStatus) *domain.Warning {
	type rule struct {
		code     string
		severity domain.WarningSeverity
		action   string
	}
	var r rule
	switch status {
	case domain.InstallStatusBrokenSymlink:
		r = rule{"broken_symlink", domain.WarningSeverityWarning, "rescan"}
	case domain.InstallStatusExternalSymlink:
		r = rule{"external_symlink", domain.WarningSeverityWarning, "open_folder"}
	case domain.InstallStatusOldHost:
		r = rule{"old_host_symlink", domain.WarningSeverityWarning, "rescan"}
	case domain.InstallStatusError:
		r = rule{"entry_error", domain.WarningSeverityInfo, "open_folder"}
	default:
		return nil
	}
	action := r.action
	return &domain.Warning{
		ScopeType: domain.WarningScopeInstall,
		Severity:  r.severity,
		Code:      r.code,
		ActionKey: &action,
	}
}

func (s *ProjectService) commitTerminalPath(ctx context.Context, project *domain.Project, pathErr error) (any, error) {
	status := domain.ProjectStatusUnreadable
	code := "project_unreadable"
	msg := "Project folder is unreadable: " + project.Path

	if fe, ok := pathErr.(*filesystem.FilesystemError); ok && fe.Code == filesystem.ErrPathNotFound {
		status = domain.ProjectStatusMissing
		code = "project_missing"
		msg = "Project folder not found: " + project.Path
	}
	return s.commitTerminalDirect(ctx, project, status, code, msg)
}

func (s *ProjectService) commitTerminalDirect(ctx context.Context, project *domain.Project, status domain.ProjectStatus, code, msg string) (any, error) {
	rescan := "rescan"
	warning := domain.Warning{
		ScopeType: domain.WarningScopeProject,
		Severity:  domain.WarningSeverityWarning,
		Code:      code,
		Message:   msg,
		ActionKey: &rescan,
	}
	if err := s.scanRepo.CommitProjectTerminal(ctx, project.ID, status, &warning, time.Now()); err != nil {
		return nil, domain.NewDatabaseError("Could not commit terminal scan state", err.Error())
	}
	return nil, nil
}
