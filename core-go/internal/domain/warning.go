package domain

import "time"

type WarningScopeType string

const (
	WarningScopeApp                     WarningScopeType = "app"
	WarningScopeSkillHostFolder         WarningScopeType = "skill_host_folder"
	WarningScopeSkill                   WarningScopeType = "skill"
	WarningScopeProject                 WarningScopeType = "project"
	WarningScopeProjectProvider         WarningScopeType = "project_provider"
	WarningScopeInstall                 WarningScopeType = "install"
	WarningScopeGlobalProviderLocation  WarningScopeType = "global_provider_location"
	WarningScopeGlobalInstall           WarningScopeType = "global_install"
)

func (s WarningScopeType) String() string { return string(s) }

type WarningSeverity string

const (
	WarningSeverityInfo     WarningSeverity = "info"
	WarningSeverityWarning  WarningSeverity = "warning"
	WarningSeverityError    WarningSeverity = "error"
	WarningSeverityBlocking WarningSeverity = "blocking"
)

type Warning struct {
	ID              int64
	ScopeType       WarningScopeType
	ScopeID         *int64
	Severity        WarningSeverity
	Code            string
	Message         string
	ActionKey       *string
	SourceOperationID *int64
	IsResolved      bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ResolvedAt      *time.Time
}
