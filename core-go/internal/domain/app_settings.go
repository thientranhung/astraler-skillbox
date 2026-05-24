package domain

import "time"

type AppSettings struct {
	ID                      int64
	ActiveSkillHostFolderID *int64
	DefaultInstallMode      string
	DatabaseVersion         int
	CreatedAt               time.Time
	UpdatedAt               time.Time
}
