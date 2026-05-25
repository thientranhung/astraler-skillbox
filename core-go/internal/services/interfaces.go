package services

import (
	"context"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
	"github.com/astraler/skillbox/core-go/internal/providers"
	"github.com/astraler/skillbox/core-go/internal/repositories"
)

// Filesystem abstracts the gateway for testability.
type Filesystem interface {
	ValidateHostPath(path string) error
	EnsureAgentsSkills(hostPath string) (bool, error)
	ScanHostFolder(skillsPath string) ([]filesystem.HostEntry, error)
}

// HostRepo is the minimal repository interface used by skill host services.
type HostRepo interface {
	GetByID(ctx context.Context, id int64) (*domain.SkillHostFolder, error)
	GetByPath(ctx context.Context, path string) (*domain.SkillHostFolder, error)
	UpsertAndActivate(ctx context.Context, name, path, skillsPath string) (int64, bool, error)
	UpdateStatus(ctx context.Context, id int64, status domain.SkillHostStatus) error
	UpdateLastScannedAt(ctx context.Context, id int64, t time.Time) error
}

// AppSettingsRepo is the minimal app settings repo interface.
type AppSettingsRepo interface {
	Get(ctx context.Context) (*domain.AppSettings, error)
}

// SkillRepo is the minimal repo interface for skills.
type SkillRepo interface {
	UpsertMany(ctx context.Context, hostID int64, skills []domain.Skill) error
	ListByHost(ctx context.Context, hostID int64) ([]domain.Skill, error)
	MarkMissing(ctx context.Context, hostID int64, presentIDs []int64) error
	ListIDsByHost(ctx context.Context, hostID int64) ([]int64, error)
}

// WarningRepo is the minimal warning repo interface.
type WarningRepo interface {
	Insert(ctx context.Context, w domain.Warning) (int64, error)
	ListByScope(ctx context.Context, scopeType domain.WarningScopeType, scopeID int64, includeResolved bool) ([]domain.Warning, error)
	ClearByScope(ctx context.Context, scopeType domain.WarningScopeType, scopeID int64) error
}

// ScanCommitter performs the full scan write phase atomically in one transaction:
// upsert skills, mark missing, update host timestamp, clear+insert warnings.
type ScanCommitter interface {
	CommitScanResults(ctx context.Context, hostID int64, skills []domain.Skill, warnings []domain.Warning, now time.Time) error
}

// OperationRunner is the minimal runner interface.
type OperationRunner interface {
	Start(ctx context.Context, target operations.Target, opType domain.OperationType, fn operations.WorkFn) (int64, error)
	// Cancel signals the operation to stop.
	// Returns (true, nil) if signal sent; (false, nil) if already finished;
	// (false, validation_error) if operationID not found; (false, db_error) on failure.
	Cancel(ctx context.Context, operationID int64) (bool, error)
}

// ProjectRepo is the minimal repository interface for projects.
type ProjectRepo interface {
	UpsertByPath(ctx context.Context, name, path string) (int64, bool, error)
	GetByID(ctx context.Context, id int64) (*domain.Project, error)
	List(ctx context.Context) ([]domain.Project, error)
	// MarkRemoved sets a project's status to removed.
	// Returns (true, nil) on success, (false, nil) if not found or already removed,
	// and (false, err) on a real database failure.
	MarkRemoved(ctx context.Context, id int64) (bool, error)
}

// ProjectFilesystem provides the read-only filesystem operations needed by the project service.
// filesystem.Gateway satisfies this interface and also satisfies providers.FsReader, so the
// gateway can be passed directly to adapter.Detect.
type ProjectFilesystem interface {
	ValidateProjectPath(path string) error
	NormalizeAbs(path string) (string, error)
	// PathInfo returns existence and readability facts for path (follows symlinks).
	// Used by ScanProject to detect unreadable directories that os.Stat alone misses.
	PathInfo(path string) (filesystem.PathInfo, error)
	// ListSkillEntries lists top-level entries in a skills directory (read-only).
	ListSkillEntries(skillsPath string) ([]filesystem.ProjectEntry, error)
}

// ProjectProviderRepo reads project_providers joined with definitions and entry counts.
// *repositories.ProjectProviderRepo satisfies this interface.
type ProjectProviderRepo interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.ProjectProviderSummary, error)
}

// ProjectWarningRepo reads warnings across project, project_provider, and install scopes.
type ProjectWarningRepo interface {
	CountActiveForProject(ctx context.Context, projectID int64) (int, error)
	ListActiveForProject(ctx context.Context, projectID int64) ([]domain.Warning, error)
}

// ProjectInstallRepo reads observed install entries for a project.
type ProjectInstallRepo interface {
	ListByProject(ctx context.Context, projectID int64) ([]domain.Install, error)
}

// ProjectScanCommitter writes project scan states atomically.
// *repositories.ProjectScanRepo satisfies both methods.
type ProjectScanCommitter interface {
	CommitProjectTerminal(ctx context.Context, projectID int64, status domain.ProjectStatus, warning *domain.Warning, now time.Time) error
	CommitProjectScan(ctx context.Context, projectID int64, provs []repositories.ProviderScanResult, projectWarnings []domain.Warning, now time.Time) error
}

// ProviderDefinitionRepo looks up provider definitions by key.
type ProviderDefinitionRepo interface {
	GetByKey(ctx context.Context, key string) (*domain.ProviderDefinition, error)
}

// ProviderRegistry returns all registered provider adapters.
type ProviderRegistry interface {
	All() []providers.ProviderAdapter
}

// SkillHostLister lists all skill host folders regardless of status.
type SkillHostLister interface {
	ListAll(ctx context.Context) ([]domain.SkillHostFolder, error)
}

// SkillsByHostLister reads skills for a given skill host folder.
type SkillsByHostLister interface {
	ListByHost(ctx context.Context, hostID int64) ([]domain.Skill, error)
}

// InstallFilesystem provides the filesystem operations needed for skill installation.
type InstallFilesystem interface {
	// LstatExists reports whether path exists (does not follow symlinks).
	LstatExists(path string) (bool, error)
	// EnsureDir creates path and all parents if they do not exist.
	EnsureDir(path string) error
	// CreateSymlink creates a symlink at linkPath pointing to targetPath.
	CreateSymlink(targetPath, linkPath string) error
}

// ActiveHostReader reads the currently active skill host folder.
// *repositories.SkillHostFolderRepo satisfies this interface.
type ActiveHostReader interface {
	GetActive(ctx context.Context) (*domain.SkillHostFolder, error)
}

// RemoveFilesystem provides the filesystem operations needed to remove a symlink
// install: re-verify the on-disk entry and unlink it. *filesystem.Gateway
// satisfies this interface.
type RemoveFilesystem interface {
	// ResolveEntry returns lstat + symlink-resolution facts for path.
	ResolveEntry(path string) (filesystem.EntryFacts, error)
	// RemoveSymlink unlinks the entry at path (os.Remove; non-recursive).
	RemoveSymlink(path string) error
}

// RemoveInstallDeleter hard-deletes a single install row by id.
// *repositories.InstallRepo satisfies this interface.
type RemoveInstallDeleter interface {
	DeleteByID(ctx context.Context, installID int64) (int64, error)
}
