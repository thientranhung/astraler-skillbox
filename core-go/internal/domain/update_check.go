package domain

import "time"

// UpdateCheckCacheEntry is a single row in plugin_update_check_cache.
type UpdateCheckCacheEntry struct {
	ProviderKey     string
	PluginName      string
	MarketplaceName string
	SourceURL       string
	SourceRef       string
	InstalledSHA    string
	InstalledVersion string
	RemoteSHA       string
	RemoteLatestTag string
	UpdateAvailable  *bool // nil = unknown
	CheckedAt       time.Time
	Error           string
}

// UpdateCheckPluginResult is returned to the renderer per-plugin.
type UpdateCheckPluginResult struct {
	ProviderKey     string  `json:"providerKey"`
	PluginName      string  `json:"pluginName"`
	MarketplaceName string  `json:"marketplaceName"`
	UpdateAvailable *bool   `json:"updateAvailable"`
	LatestVersion   *string `json:"latestVersion"`
	LastCheckedAt   *string `json:"lastCheckedAt"`
	Error           string  `json:"error,omitempty"`
}
