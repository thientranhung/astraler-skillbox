package providers

// ScanAntigravityCLISettingsFile reads and parses an Antigravity CLI settings.json file.
// The format is identical to Claude's settings.json (JSON with enabledPlugins and
// extraKnownMarketplaces), so this delegates directly to ScanClaudeSettingsFile with
// the same security bounds.
func ScanAntigravityCLISettingsFile(filePath, allowedDir string) ClaudeSettingsScan {
	return ScanClaudeSettingsFile(filePath, allowedDir)
}
