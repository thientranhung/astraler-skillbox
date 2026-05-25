package services

import (
	"context"
	"fmt"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/operations"
)

// removeSkillMetadata is the operation result for a remove-skill run, stored in
// operations.metadata_json and surfaced to the renderer on the terminal event.
type removeSkillMetadata struct {
	ProjectID     int64  `json:"projectId"`
	ProviderKey   string `json:"providerKey"`
	SkillName     string `json:"skillName"`
	RemovedPath   string `json:"removedPath"`
	AlreadyAbsent bool   `json:"alreadyAbsent"`
}

// RemoveSkill validates the request synchronously, then queues an async remove
// operation. Returns the operation ID on success.
func (s *ProjectService) RemoveSkill(ctx context.Context, projectID, installID int64) (int64, error) {
	if projectID <= 0 {
		return 0, domain.NewValidationError("Invalid project", "projectId must be positive")
	}
	if installID <= 0 {
		return 0, domain.NewValidationError("Invalid install", "installId must be positive")
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return 0, domain.NewDatabaseError("Could not fetch project", err.Error())
	}
	if project == nil {
		return 0, domain.NewValidationError("Project not found", fmt.Sprintf("projectId %d does not exist", projectID))
	}
	if project.Status != domain.ProjectStatusActive {
		return 0, domain.NewValidationError(
			"Project is not active",
			fmt.Sprintf("projectId %d has status %q; only active projects can have skills removed", projectID, project.Status),
		)
	}

	// Load the install via the project-scoped list (implicit ownership check).
	installs, err := s.installRepo.ListByProject(ctx, projectID)
	if err != nil {
		return 0, domain.NewDatabaseError("Could not load installs", err.Error())
	}
	var install *domain.Install
	for i := range installs {
		if installs[i].ID == installID {
			install = &installs[i]
			break
		}
	}
	if install == nil {
		return 0, domain.NewValidationError(
			"Install not found",
			fmt.Sprintf("installId %d does not exist in project %d", installID, projectID),
		)
	}

	// Removable precheck: symlink into the active host (status=current) only.
	if install.InstallMode != domain.InstallModeSymlink || install.InstallStatus != domain.InstallStatusCurrent {
		return 0, domain.NewValidationError(
			"Install is not removable",
			fmt.Sprintf("install %d has mode=%q status=%q; only current symlink installs can be removed in this slice",
				installID, install.InstallMode, install.InstallStatus),
		)
	}

	// Resolve and bound the path under the project root.
	path, err := s.fs.NormalizeAbs(install.ProjectSkillPath)
	if err != nil {
		return 0, domain.NewValidationError("Invalid install path", err.Error())
	}
	root, err := s.fs.NormalizeAbs(project.Path)
	if err != nil {
		return 0, domain.NewValidationError("Invalid project path", err.Error())
	}
	if !isWithin(root, path) {
		return 0, domain.NewValidationError(
			"Install path escapes project root",
			fmt.Sprintf("path %q is not within project root %q", path, root),
		)
	}

	// Resolve provider key for metadata via the project_providers summary.
	providerKey := s.providerKeyForInstall(ctx, projectID, install.ProjectProviderID)

	loaded := *install
	target := operations.Target{Type: "project", ID: projectID}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeRemoveSkill,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return s.removeSkillInternal(opCtx, project, loaded, providerKey, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not queue remove operation", err.Error())
	}
	return opID, nil
}

// providerKeyForInstall returns the provider key for the install's
// project_provider_id, or "" if it cannot be resolved (metadata is best-effort).
func (s *ProjectService) providerKeyForInstall(ctx context.Context, projectID, projectProviderID int64) string {
	summaries, err := s.ppRepo.ListByProject(ctx, projectID)
	if err != nil {
		return ""
	}
	for _, sum := range summaries {
		if sum.ProjectProviderID == projectProviderID {
			return sum.ProviderKey
		}
	}
	return ""
}

// removeSkillInternal is the async work function executed inside the operation runner.
func (s *ProjectService) removeSkillInternal(
	ctx context.Context,
	project *domain.Project,
	install domain.Install,
	providerKey string,
	progress operations.ProgressFn,
) (any, error) {
	progress("validating", 0, 0, "")

	path := install.ProjectSkillPath
	meta := removeSkillMetadata{
		ProjectID:   project.ID,
		ProviderKey: providerKey,
		SkillName:   install.SkillName,
		RemovedPath: path,
	}

	// 1. On-disk re-verification (do NOT trust the stored classification).
	facts, err := s.removeFS.ResolveEntry(path)
	if err != nil {
		return nil, domain.NewFilesystemError("Could not inspect install entry", err.Error())
	}

	alreadyAbsent := false
	switch {
	case !facts.Exists:
		alreadyAbsent = true
	case !facts.IsSymlink:
		return nil, domain.NewConflictError(
			"This entry changed on disk. Rescan the project and try again.",
			fmt.Sprintf("path %q is no longer a symlink; refusing to delete real content", path),
		)
	default:
		// It is a symlink: it must resolve inside the active host.
		activeHost, herr := s.activeHostReader.GetActive(ctx)
		if herr != nil {
			return nil, domain.NewDatabaseError("Could not load active skill host", herr.Error())
		}
		if activeHost == nil || facts.Broken || facts.ResolvedTarget == "" ||
			!isWithin(activeHost.SkillsPath, facts.ResolvedTarget) {
			return nil, domain.NewConflictError(
				"This entry changed on disk. Rescan the project and try again.",
				fmt.Sprintf("symlink %q no longer resolves inside the active host", path),
			)
		}
	}

	// 2. Unlink (skipped when already absent).
	if !alreadyAbsent {
		progress("removing_symlink", 0, 0, "")
		if err := s.removeFS.RemoveSymlink(path); err != nil {
			return nil, domain.NewFilesystemError("Could not remove skill symlink", err.Error())
		}
	}
	meta.AlreadyAbsent = alreadyAbsent

	// 3. Authoritative rescan.
	if _, rescanErr := s.scanProjectInternal(ctx, project, progress); rescanErr != nil {
		return meta, rescanErr
	}

	// 4. Hard-delete the one targeted install row.
	progress("deleting_record", 0, 0, "")
	if _, derr := s.installDeleter.DeleteByID(ctx, install.ID); derr != nil {
		return meta, domain.NewDatabaseError("Could not delete install record", derr.Error())
	}

	progress("done", 0, 0, "")
	return meta, nil
}
