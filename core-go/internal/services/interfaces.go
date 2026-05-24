package services

import (
	"context"
	"time"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/operations"
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
