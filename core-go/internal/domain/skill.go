package domain

import "time"

type SkillStatus string

const (
	SkillStatusAvailable       SkillStatus = "available"
	SkillStatusMissing         SkillStatus = "missing"
	SkillStatusUnreadable      SkillStatus = "unreadable"
	SkillStatusLocalModified   SkillStatus = "local_modified"
	SkillStatusExternalSymlink SkillStatus = "external_symlink"
	SkillStatusUnknown         SkillStatus = "unknown"
)

type SkillProjectUsage struct {
	ProjectID           int64
	ProjectName         string
	ProjectProviderID   int64
	ProviderKey         string
	ProviderDisplayName string
	Mode                string
	Status              string
	ProjectSkillPath    string
}

type Skill struct {
	ID                int64
	SkillHostFolderID int64
	Name              string
	DisplayName       *string
	RelativePath      string
	AbsolutePath      string
	Status            SkillStatus
	SourceID          *int64
	CurrentVersion    *string
	CurrentCommit     *string
	CurrentChecksum   *string
	LastScannedAt     *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
