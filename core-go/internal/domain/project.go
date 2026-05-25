package domain

import "time"

type ProjectStatus string

const (
	ProjectStatusActive     ProjectStatus = "active"
	ProjectStatusMissing    ProjectStatus = "missing"
	ProjectStatusUnreadable ProjectStatus = "unreadable"
	ProjectStatusRemoved    ProjectStatus = "removed"
)

func (s ProjectStatus) String() string { return string(s) }

type Project struct {
	ID            int64
	Name          string
	Path          string
	Status        ProjectStatus
	LastScannedAt *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
