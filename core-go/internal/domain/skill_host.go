package domain

import "time"

type SkillHostStatus string

const (
	SkillHostStatusActive           SkillHostStatus = "active"
	SkillHostStatusMissing          SkillHostStatus = "missing"
	SkillHostStatusUnreadable       SkillHostStatus = "unreadable"
	SkillHostStatusUnwritable       SkillHostStatus = "unwritable"
	SkillHostStatusInvalidStructure SkillHostStatus = "invalid_structure"
	SkillHostStatusEmpty            SkillHostStatus = "empty"
	SkillHostStatusInactive         SkillHostStatus = "inactive"
)

type SkillHostFolder struct {
	ID            int64
	Name          string
	Path          string
	SkillsPath    string
	Status        SkillHostStatus
	LastScannedAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
