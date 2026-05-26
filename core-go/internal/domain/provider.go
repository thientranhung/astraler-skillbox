package domain

import "time"

type ProviderStatus string

const (
	ProviderStatusSupported    ProviderStatus = "supported"
	ProviderStatusExperimental ProviderStatus = "experimental"
	ProviderStatusUnsupported  ProviderStatus = "unsupported"
	ProviderStatusDisabled     ProviderStatus = "disabled"
)

func (s ProviderStatus) String() string { return string(s) }

type DetectionStatus string

const (
	DetectionStatusDetected         DetectionStatus = "detected"
	DetectionStatusConfigured       DetectionStatus = "configured"
	DetectionStatusMissing          DetectionStatus = "missing"
	DetectionStatusUnsupported      DetectionStatus = "unsupported"
	DetectionStatusInvalidStructure DetectionStatus = "invalid_structure"
	DetectionStatusFormatUnknown    DetectionStatus = "format_unknown"
)

func (s DetectionStatus) String() string { return string(s) }

type ProviderDefinition struct {
	ID                 int64
	Key                string
	DisplayName        string
	ProviderType       string
	IconKey            *string
	Status             ProviderStatus
	CanCreateStructure bool
	HasGlobalLevel     bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type ProjectProvider struct {
	ID                   int64
	ProjectID            int64
	ProviderDefinitionID int64
	DetectedPath         *string
	SkillsPath           *string
	DetectionStatus      DetectionStatus
	LastScannedAt        *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// ProjectProviderSummary is a read-only view of a project_provider row joined with
// provider_definitions and an install entry count. Used by list and detail queries.
type ProjectProviderSummary struct {
	ProjectProviderID   int64
	ProviderKey         string
	ProviderDisplayName string
	ProviderStatus      ProviderStatus
	DetectionStatus     DetectionStatus
	DetectedPath        *string
	SkillsPath          *string
	EntryCount          int
}

// ProviderPathCandidate is a single path candidate for a provider, as stored in
// provider_path_candidates. Scope and purpose classify how it is used.
type ProviderPathCandidate struct {
	ID                 int64
	ProviderDefinitionID int64
	RelativePath       string
	Scope              string // "project" or "global"
	Purpose            string // "detect", "skills", "config", "commands"
	Priority           int
	VerificationStatus string // "verified", "assumed", "experimental"
}

// ProviderRegistryEntry is the full read-only view of a provider definition with
// its path candidates. Used by provider.list.
type ProviderRegistryEntry struct {
	Definition ProviderDefinition
	Candidates []ProviderPathCandidate
}
