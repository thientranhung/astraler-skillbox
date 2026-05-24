package services

import (
	"context"
	"path/filepath"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
)

// ChooseHostResult is returned by ChooseHost.
type ChooseHostResult struct {
	HostID      int64
	Path        string
	SkillsPath  string
	Initialized bool
	Status      domain.SkillHostStatus
}

// SkillHostService handles skill host folder operations.
type SkillHostService struct {
	hostRepo     HostRepo
	settingsRepo AppSettingsRepo
	fs           Filesystem
	runner       OperationRunner
	scanWriter   ScanCommitter
}

func NewSkillHostService(
	hostRepo HostRepo,
	settingsRepo AppSettingsRepo,
	fs Filesystem,
	runner OperationRunner,
	scanWriter ScanCommitter,
) *SkillHostService {
	return &SkillHostService{
		hostRepo:     hostRepo,
		settingsRepo: settingsRepo,
		fs:           fs,
		runner:       runner,
		scanWriter:   scanWriter,
	}
}

// ChooseHost validates the path, ensures .agents/skills exists, and persists
// the host as active. Idempotent by path; switching host is not an error.
func (s *SkillHostService) ChooseHost(ctx context.Context, path string) (*ChooseHostResult, error) {
	if err := s.fs.ValidateHostPath(path); err != nil {
		fe, ok := err.(*filesystem.FilesystemError)
		if ok {
			return nil, domain.NewValidationError(
				"Invalid host folder path",
				string(fe.Code)+": "+fe.Message,
			)
		}
		return nil, domain.NewValidationError("Invalid host folder path", err.Error())
	}

	initialized, err := s.fs.EnsureAgentsSkills(path)
	if err != nil {
		return nil, domain.NewFilesystemError(
			"Could not create .agents/skills directory",
			err.Error(),
		)
	}

	skillsPath := filepath.Join(path, ".agents", "skills")
	name := filepath.Base(path)

	hostID, _, err := s.hostRepo.UpsertAndActivate(ctx, name, path, skillsPath)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not persist host folder", err.Error())
	}

	host, err := s.hostRepo.GetByID(ctx, hostID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not fetch host folder", err.Error())
	}

	return &ChooseHostResult{
		HostID:      hostID,
		Path:        host.Path,
		SkillsPath:  host.SkillsPath,
		Initialized: initialized,
		Status:      host.Status,
	}, nil
}

// ScanHost queues an async scan operation for the given host.
// Returns conflict_error if a scan is already running for this host.
func (s *SkillHostService) ScanHost(ctx context.Context, hostID int64) (int64, error) {
	host, err := s.hostRepo.GetByID(ctx, hostID)
	if err != nil || host == nil {
		return 0, domain.NewValidationError("Host not found", "hostId not in database")
	}

	target := operations.Target{Type: "skill_host_folder", ID: hostID}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeScan,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return s.scanHostInternal(opCtx, host, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not queue scan operation", err.Error())
	}
	return opID, nil
}

type scanSummary struct {
	SkillsFound     int `json:"skillsFound"`
	WarningsCreated int `json:"warningsCreated"`
}

func (s *SkillHostService) scanHostInternal(ctx context.Context, host *domain.SkillHostFolder, progress operations.ProgressFn) (any, error) {
	progress("reading_host_folder", 0, 0, "")

	entries, err := s.fs.ScanHostFolder(host.SkillsPath)
	if err != nil {
		return nil, domain.NewFilesystemError("Could not scan host folder", err.Error())
	}

	progress("classifying_entries", len(entries), len(entries), "")

	var skills []domain.Skill
	var warnings []domain.Warning
	hostID := host.ID

	for _, e := range entries {
		if !e.IsDir && !e.IsSymlink {
			continue // skip plain files
		}
		status := classifyEntry(e)
		skills = append(skills, domain.Skill{
			SkillHostFolderID: hostID,
			Name:              e.Name,
			RelativePath:      e.RelativePath,
			AbsolutePath:      e.AbsolutePath,
			Status:            status,
		})
		if w := warningForEntry(e, hostID); w != nil {
			warnings = append(warnings, *w)
		}
	}

	// Commit everything atomically: upsert skills, mark missing, update host
	// timestamp, clear old warnings, insert new warnings.
	if err := s.scanWriter.CommitScanResults(ctx, hostID, skills, warnings, time.Now()); err != nil {
		return nil, domain.NewDatabaseError("Could not persist scan results", err.Error())
	}

	progress("done", len(skills), len(skills), "")

	return scanSummary{SkillsFound: len(skills), WarningsCreated: len(warnings)}, nil
}

func classifyEntry(e filesystem.HostEntry) domain.SkillStatus {
	if e.Broken {
		return domain.SkillStatusUnreadable
	}
	if e.IsDir || (e.IsSymlink && !e.External) {
		return domain.SkillStatusAvailable
	}
	if e.External {
		return domain.SkillStatusAvailable // external symlink counts as available
	}
	return domain.SkillStatusUnknown
}

// warningForEntry generates a host-scope warning for broken or external symlinks.
// Warnings use WarningScopeSkillHostFolder so that SkillLibraryService.List
// and ClearByScope operate on the correct scope.
func warningForEntry(e filesystem.HostEntry, hostID int64) *domain.Warning {
	id := hostID
	action := "rescan"
	if e.Broken {
		return &domain.Warning{
			ScopeType: domain.WarningScopeSkillHostFolder,
			ScopeID:   &id,
			Severity:  domain.WarningSeverityWarning,
			Code:      "broken_symlink",
			Message:   "Skill " + e.Name + " has a broken symlink",
			ActionKey: &action,
		}
	}
	if e.External {
		return &domain.Warning{
			ScopeType: domain.WarningScopeSkillHostFolder,
			ScopeID:   &id,
			Severity:  domain.WarningSeverityWarning,
			Code:      "external_symlink",
			Message:   "Skill " + e.Name + " points outside the skills folder",
			ActionKey: &action,
		}
	}
	return nil
}
