package domain

type GlobalLocationStatus string

const (
	GlobalLocationStatusActive           GlobalLocationStatus = "active"
	GlobalLocationStatusNotConfigured    GlobalLocationStatus = "not_configured"
	GlobalLocationStatusMissing          GlobalLocationStatus = "missing"
	GlobalLocationStatusUnreadable       GlobalLocationStatus = "unreadable"
	GlobalLocationStatusInvalidStructure GlobalLocationStatus = "invalid_structure"
	GlobalLocationStatusEmpty            GlobalLocationStatus = "empty"
	GlobalLocationStatusDisabled         GlobalLocationStatus = "disabled"
)

// GlobalInstallView backs the global.list query (read model).
type GlobalInstallView struct {
	GlobalInstallID   int64
	SkillID           *int64
	SkillName         string
	Mode              InstallMode
	Status            InstallStatus
	GlobalSkillPath   string
	SourceSkillPath   *string
	SymlinkTargetPath *string
}

// GlobalLocationView backs the global.list query (read model).
type GlobalLocationView struct {
	GlobalProviderLocationID int64
	ProviderKey              string
	ProviderDisplayName      string
	ProviderStatus           string
	Path                     *string
	SkillsPath               *string
	Status                   GlobalLocationStatus
	LastScannedAt            *string
	Entries                  []GlobalInstallView
	Warnings                 []Warning
}
