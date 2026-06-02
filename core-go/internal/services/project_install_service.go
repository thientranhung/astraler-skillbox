package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// InstallSkills validates the request synchronously, then queues an async install
// operation via the runner. Returns the operation ID on success.
//
// Synchronous validation:
//   - skillIDs must be non-empty
//   - skillIDs must be unique positive integers
//   - providerKey must be a known install target
//   - project must exist and have status=active
//
// Returns conflict_error if an install is already running for this project.
func (s *ProjectService) InstallSkills(
	ctx context.Context,
	projectID int64,
	providerKey string,
	skillIDs []int64,
) (int64, error) {
	// Validate skillIDs: non-empty.
	if len(skillIDs) == 0 {
		return 0, domain.NewValidationError("No skills selected", "skillIDs must not be empty")
	}

	// Validate skillIDs: unique positive values.
	seen := make(map[int64]struct{}, len(skillIDs))
	for _, id := range skillIDs {
		if id <= 0 {
			return 0, domain.NewValidationError("Invalid skill ID", fmt.Sprintf("skill ID %d must be positive", id))
		}
		if _, dup := seen[id]; dup {
			return 0, domain.NewValidationError("Duplicate skill IDs", fmt.Sprintf("skill ID %d appears more than once", id))
		}
		seen[id] = struct{}{}
	}

	// Validate providerKey.
	if _, ok := providers.InstallTargetByProviderKey(providerKey); !ok {
		return 0, domain.NewValidationError(
			"Unknown provider",
			fmt.Sprintf("providerKey %q is not a known install target", providerKey),
		)
	}

	// Load project — must exist and be active.
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
	if project.Status != domain.ProjectStatusActive {
		return 0, domain.NewValidationError(
			"Project is not active",
			fmt.Sprintf("projectId %d has status %q; only active projects can receive skill installs", projectID, project.Status),
		)
	}

	// Queue the async operation.
	target := operations.Target{Type: "project", ID: projectID}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeInstallSkill,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return s.installSkillsInternal(opCtx, project, providerKey, skillIDs, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not queue install operation", err.Error())
	}
	return opID, nil
}

// installMetadata is the operation result for an install-skills run, stored in
// operations.metadata_json.
type installMetadata struct {
	Requested   int    `json:"requested"`
	Created     int    `json:"created"`
	Failed      int    `json:"failed"`
	ProviderKey string `json:"providerKey"`
}

// installSkillsInternal is the async work function executed inside the operation runner.
// It resolves the install target, validates provider/host/skill state, fail-fast checks
// for symlink conflicts, creates symlinks, and ALWAYS rescans the project once the
// symlink-write phase is reached. Filesystem failures during the write phase produce a
// metadata result plus the underlying error (operation FAILED with metadata).
func (s *ProjectService) installSkillsInternal(
	ctx context.Context,
	project *domain.Project,
	providerKey string,
	skillIDs []int64,
	progress operations.ProgressFn,
) (any, error) {
	progress("validating", 0, 0, "")

	// 1. Resolve install target.
	target, ok := providers.InstallTargetByProviderKey(providerKey)
	if !ok {
		return nil, domain.NewValidationError(
			"Unknown provider",
			fmt.Sprintf("providerKey %q is not a known install target", providerKey),
		)
	}

	// 2. Load provider definition.
	pd, err := s.providerDefRepo.GetByKey(ctx, providerKey)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not look up provider definition", err.Error())
	}
	if pd == nil {
		return nil, domain.NewValidationError(
			"Provider not found",
			fmt.Sprintf("provider definition %q does not exist", providerKey),
		)
	}
	if pd.Status != domain.ProviderStatusSupported && pd.Status != domain.ProviderStatusExperimental {
		return nil, domain.NewProviderError(
			"Provider not supported",
			fmt.Sprintf("provider %q has status %q; skills cannot be installed", providerKey, pd.Status),
		)
	}

	// 3. Locate the project_provider summary for this provider.
	summaries, err := s.ppRepo.ListByProject(ctx, project.ID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not list project providers", err.Error())
	}
	var summary *domain.ProjectProviderSummary
	for i := range summaries {
		if summaries[i].ProviderKey == providerKey {
			summary = &summaries[i]
			break
		}
	}
	if summary == nil {
		return nil, domain.NewValidationError(
			"Provider not detected in project",
			fmt.Sprintf("provider %q has no detection record for this project; scan first", providerKey),
		)
	}
	if summary.DetectionStatus != domain.DetectionStatusDetected &&
		summary.DetectionStatus != domain.DetectionStatusConfigured {
		return nil, domain.NewValidationError(
			"Provider is not ready for installs",
			fmt.Sprintf("provider %q has detection status %q; expected detected or configured", providerKey, summary.DetectionStatus),
		)
	}

	// 4. Resolve and bound the skills path under the project root.
	// Use effective skills rel: override ?? builtin (from pathResolver), fall back to target default.
	skillsRel := target.RelativeSkillsPath
	if s.pathResolver != nil {
		pathsMap, resolveErr := s.pathResolver.ProjectPaths(ctx)
		if resolveErr != nil {
			return nil, domain.NewDatabaseError("Could not resolve provider paths", resolveErr.Error())
		}
		if ep, ok := pathsMap[providerKey]; ok && ep.SkillsRel != "" {
			skillsRel = ep.SkillsRel
		}
	}
	skillsPath, err := s.fs.NormalizeAbs(filepath.Join(project.Path, skillsRel))
	if err != nil {
		return nil, domain.NewValidationError("Invalid skills path", err.Error())
	}
	root, err := s.fs.NormalizeAbs(project.Path)
	if err != nil {
		return nil, domain.NewValidationError("Invalid project path", err.Error())
	}
	if !isWithin(root, skillsPath) {
		return nil, domain.NewValidationError(
			"Skills path escapes project root",
			fmt.Sprintf("resolved skills path %q is not within project root %q", skillsPath, root),
		)
	}

	// 5. Resolve requested skills against the active host, preserving request order.
	activeHost, err := s.activeHostReader.GetActive(ctx)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load active skill host", err.Error())
	}
	if activeHost == nil {
		return nil, domain.NewValidationError(
			"No active skill host",
			"no active host folder is configured; configure one in Settings first",
		)
	}
	hostSkills, err := s.installSkillReader.ListByHost(ctx, activeHost.ID)
	if err != nil {
		return nil, domain.NewDatabaseError("Could not load host skills", err.Error())
	}
	skillByID := make(map[int64]domain.Skill, len(hostSkills))
	for _, sk := range hostSkills {
		skillByID[sk.ID] = sk
	}
	// Resolve activeHost.SkillsPath to its canonical realpath so the containment
	// check below works correctly on systems where paths have symlink prefixes
	// (e.g. /var → /private/var on macOS).
	realHostSkillsPath, evalHostErr := filepath.EvalSymlinks(activeHost.SkillsPath)
	if evalHostErr != nil {
		return nil, domain.NewDatabaseError(
			"Could not resolve host skills path",
			fmt.Sprintf("host skills path %q: %s", activeHost.SkillsPath, evalHostErr),
		)
	}

	resolved := make([]domain.Skill, 0, len(skillIDs))
	for _, id := range skillIDs {
		sk, found := skillByID[id]
		if !found {
			return nil, domain.NewValidationError(
				"Skill not found",
				fmt.Sprintf("skill ID %d is not available on the active host", id),
			)
		}
		if sk.Status != domain.SkillStatusAvailable {
			return nil, domain.NewValidationError(
				"Skill not available",
				fmt.Sprintf("skill %q (ID %d) has status %q; only available skills can be installed", sk.Name, id, sk.Status),
			)
		}
		// Defense-in-depth: resolve the skill's absolute path to its realpath before
		// installing. A host skill that is a symlink pointing outside the host skills
		// folder (external_symlink) must not produce a project install — the project
		// symlink would transitively escape the active Skill Host Folder, violating the
		// source-of-truth invariant. This check catches both correctly-classified rows
		// and any stale DB rows that still report status=available for such a skill.
		realSkillPath, evalErr := filepath.EvalSymlinks(sk.AbsolutePath)
		if evalErr != nil {
			return nil, domain.NewValidationError(
				"Skill source path cannot be resolved",
				fmt.Sprintf("skill %q source %q: %s", sk.Name, sk.AbsolutePath, evalErr),
			)
		}
		if !isWithin(realHostSkillsPath, realSkillPath) {
			return nil, domain.NewValidationError(
				"Skill escapes Skill Host Folder",
				fmt.Sprintf("skill %q resolves to %q which is outside the active Skill Host Folder; install refused", sk.Name, realSkillPath),
			)
		}
		resolved = append(resolved, sk)
	}

	// 6. Validate each skill segment and compute its link path.
	linkPaths := make([]string, len(resolved))
	for i, sk := range resolved {
		if err := validateSkillSegment(sk.Name); err != nil {
			return nil, err
		}
		linkPath := filepath.Join(skillsPath, sk.Name)
		if filepath.Dir(linkPath) != skillsPath {
			return nil, domain.NewValidationError(
				"Invalid skill link path",
				fmt.Sprintf("skill %q resolves outside the skills directory", sk.Name),
			)
		}
		linkPaths[i] = linkPath
	}

	// 7. Conflict check (fail-fast, pre-write).
	var conflicts []string
	for i, sk := range resolved {
		exists, err := s.installFS.LstatExists(linkPaths[i])
		if err != nil {
			return nil, domain.NewFilesystemError("Could not check existing skill", err.Error())
		}
		if exists {
			conflicts = append(conflicts, sk.Name)
		}
	}
	if len(conflicts) > 0 {
		return nil, domain.NewConflictError(
			"Some skills are already installed",
			fmt.Sprintf("the following skills already exist in the target folder: %s", strings.Join(conflicts, ", ")),
		)
	}

	// 8. Ensure the skills directory exists.
	pi, err := s.fs.PathInfo(skillsPath)
	if err != nil {
		return nil, domain.NewFilesystemError("Could not inspect skills folder", err.Error())
	}
	if !pi.Exists {
		if !pd.CanCreateStructure {
			displayName := pd.DisplayName
			if displayName == "" {
				displayName = pd.Key
			}
			return nil, domain.NewProviderError(
				"Skills folder missing",
				fmt.Sprintf("%s skills folder does not exist and cannot be created automatically", displayName),
			)
		}
		if err := s.installFS.EnsureDir(skillsPath); err != nil {
			return nil, domain.NewFilesystemError("Could not create skills folder", err.Error())
		}
	}

	// 9. Create symlinks; stop at the first failure.
	progress("creating_symlinks", 0, len(skillIDs), "")
	created := 0
	var createErr error
	for i, sk := range resolved {
		if err := s.installFS.CreateSymlink(sk.AbsolutePath, linkPaths[i]); err != nil {
			createErr = domain.NewFilesystemError(
				"Could not create skill symlink",
				fmt.Sprintf("failed to link skill %q: %s", sk.Name, err.Error()),
			)
			break
		}
		created++
		progress("creating_symlinks", created, len(skillIDs), "")
	}

	// 10. Always rescan once the write phase has been reached.
	_, rescanErr := s.scanProjectInternal(ctx, project, progress)

	// 11. Build metadata and return.
	failed := len(skillIDs) - created
	meta := installMetadata{
		Requested:   len(skillIDs),
		Created:     created,
		Failed:      failed,
		ProviderKey: providerKey,
	}
	if createErr != nil {
		return meta, createErr
	}
	if rescanErr != nil {
		return meta, rescanErr
	}
	return meta, nil
}
