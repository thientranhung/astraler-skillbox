package domain

import "time"

// PluginLayerScanStatus is the outcome of scanning a single Claude settings file.
type PluginLayerScanStatus string

const (
	PluginLayerScanOK          PluginLayerScanStatus = "ok"
	PluginLayerScanMissing     PluginLayerScanStatus = "missing"
	PluginLayerScanUnreadable  PluginLayerScanStatus = "unreadable"
	PluginLayerScanMalformed   PluginLayerScanStatus = "malformed"
	PluginLayerScanTooLarge    PluginLayerScanStatus = "too_large"
	PluginLayerScanSymlink     PluginLayerScanStatus = "symlink"
	PluginLayerScanPathEscape  PluginLayerScanStatus = "path_escape"
)

// PluginSettingsLayer is the precedence level of a Claude settings file.
type PluginSettingsLayer string

const (
	PluginLayerUser    PluginSettingsLayer = "user"
	PluginLayerProject PluginSettingsLayer = "project"
	PluginLayerLocal   PluginSettingsLayer = "local"
)

// PluginDeclaration is how a plugin is declared within a settings file.
type PluginDeclaration string

const (
	PluginDeclarationEnabled  PluginDeclaration = "enabled"
	PluginDeclarationDisabled PluginDeclaration = "disabled"
)

// PluginEffectiveStatus is the resolved status after merging layers (local > project > user).
type PluginEffectiveStatus string

const (
	PluginEffectiveEnabled  PluginEffectiveStatus = "enabled"
	PluginEffectiveDisabled PluginEffectiveStatus = "disabled"
	PluginEffectiveAbsent   PluginEffectiveStatus = "absent"
	PluginEffectiveUnknown  PluginEffectiveStatus = "unknown"
)

// PluginLayerScan is a single settings-layer scan record (one row per provider+project+layer).
type PluginLayerScan struct {
	ID                   int64
	ProviderDefinitionID int64
	ProjectID            *int64 // nil for user/global layer
	SettingsLayer        PluginSettingsLayer
	ScanStatus           PluginLayerScanStatus
	SettingsFilePath     string
	LastScannedAt        time.Time
	SourceOperationID    *int64
	Warnings             []string // parse-time warnings; never raw file content; bounded
}

// PluginEntry is a single plugin declaration within a layer scan.
type PluginEntry struct {
	ID              int64
	LayerScanID     int64
	PluginName      string
	MarketplaceName string
	Declaration     PluginDeclaration
}

// PluginMarketplace is a declared extra marketplace within a layer scan.
type PluginMarketplace struct {
	ID              int64
	LayerScanID     int64
	MarketplaceName string
	SourceType      string
	SourceSummary   string
}

// PluginLayerBreakdown describes a single layer's contribution to effective state.
type PluginLayerBreakdown struct {
	Layer       PluginSettingsLayer
	ScanStatus  PluginLayerScanStatus
	Declaration *PluginDeclaration // nil = absent (scan ok but key not present in that layer)
}

// PluginEffectiveEntry is a plugin with resolved effective status and per-layer provenance.
type PluginEffectiveEntry struct {
	PluginName      string
	MarketplaceName string
	EffectiveStatus PluginEffectiveStatus
	ProvenanceLayer *PluginSettingsLayer // nil if absent or unknown
	LayerBreakdown  []PluginLayerBreakdown
}

// GlobalPluginView is the resolved view for the user (global ~/.claude/settings.json) layer.
type GlobalPluginView struct {
	ProviderKey       string
	UserLayerPath     string           // expected path, always set even before first scan
	Scan              *PluginLayerScan // nil = never scanned
	Plugins           []PluginEntry    // non-empty only if Scan != nil and ScanStatus == ok
	Marketplaces      []PluginMarketplace
	ManagedOutOfScope bool // always true in Slice 1; managed settings not implemented
}

// ProjectPluginView is the resolved view for a project (merges local + project + user layers).
type ProjectPluginView struct {
	ProjectID         int64
	ProviderKey       string
	LayerScans        []PluginLayerScan // ordered: local, project, user (omitted if absent)
	Plugins           []PluginEffectiveEntry
	Marketplaces      []PluginMarketplace
	ManagedOutOfScope bool // always true in Slice 1
}
