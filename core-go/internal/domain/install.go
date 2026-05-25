package domain

import "time"

type InstallMode string

const (
	InstallModeSymlink   InstallMode = "symlink"
	InstallModeRsyncCopy InstallMode = "rsync_copy"
	InstallModeDirect    InstallMode = "direct"
)

func (m InstallMode) String() string { return string(m) }

type InstallStatus string

const (
	InstallStatusCurrent         InstallStatus = "current"
	InstallStatusOutdated        InstallStatus = "outdated"
	InstallStatusMissing         InstallStatus = "missing"
	InstallStatusBrokenSymlink   InstallStatus = "broken_symlink"
	InstallStatusOldHost         InstallStatus = "old_host"
	InstallStatusExternalSymlink InstallStatus = "external_symlink"
	InstallStatusConflict        InstallStatus = "conflict"
	InstallStatusNeedsSync       InstallStatus = "needs_sync"
	InstallStatusError           InstallStatus = "error"
)

func (s InstallStatus) String() string { return string(s) }

type Install struct {
	ID                        int64
	ProjectProviderID         int64
	SkillID                   *int64
	SkillName                 string
	InstallMode               InstallMode
	InstallStatus             InstallStatus
	ProjectSkillPath          string
	SourceSkillPath           *string
	SymlinkTargetPath         *string
	InstalledFromHostFolderID *int64
	InstalledVersion          *string
	InstalledCommit           *string
	InstalledChecksum         *string
	LastSyncedAt              *time.Time
	LastScannedAt             *time.Time
	CreatedAt                 time.Time
	UpdatedAt                 time.Time
}
