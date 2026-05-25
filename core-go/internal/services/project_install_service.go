package services

import (
	"context"
	"fmt"

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

// installSkillsInternal is the async work function executed inside the operation runner.
// Implemented in Task 6.
func (s *ProjectService) installSkillsInternal(
	_ context.Context,
	_ *domain.Project,
	_ string,
	_ []int64,
	_ operations.ProgressFn,
) (any, error) {
	panic("not implemented")
}
