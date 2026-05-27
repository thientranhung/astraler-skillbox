package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ScanCodexConfigFile reads and parses a Codex config.toml file with the same
// security bounds used for Claude settings scans.
func ScanCodexConfigFile(filePath, allowedDir string) ClaudeSettingsScan {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return ClaudeSettingsScan{Status: "path_escape"}
	}

	parentDir := filepath.Dir(cleanFile)
	if lfi, err := os.Lstat(parentDir); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		return ClaudeSettingsScan{Status: "symlink"}
	}

	fi, err := os.Lstat(cleanFile)
	if err != nil {
		if os.IsNotExist(err) {
			return ClaudeSettingsScan{Status: "missing"}
		}
		return ClaudeSettingsScan{Status: "unreadable"}
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return ClaudeSettingsScan{Status: "symlink"}
	}
	if fi.Size() > ClaudeSettingsMaxFileSize {
		return ClaudeSettingsScan{Status: "too_large"}
	}

	data, err := os.ReadFile(cleanFile)
	if err != nil {
		return ClaudeSettingsScan{Status: "unreadable"}
	}

	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		return ClaudeSettingsScan{Status: "malformed"}
	}

	result := ClaudeSettingsScan{Status: "ok"}
	parseCodexPlugins(raw, &result)
	parseCodexMarketplaces(raw, &result)
	return result
}

func parseCodexPlugins(raw map[string]interface{}, result *ClaudeSettingsScan) {
	pluginsRaw, exists := raw["plugins"]
	if !exists {
		return
	}
	plugins, ok := pluginsRaw.(map[string]interface{})
	if !ok {
		result.Warnings = append(result.Warnings, "skipped plugins section: expected TOML table")
		return
	}
	count := 0
	for key, val := range plugins {
		if count >= ClaudeSettingsMaxPlugins {
			result.Warnings = append(result.Warnings, "plugins truncated at 1000 entries")
			break
		}
		table, ok := val.(map[string]interface{})
		if !ok {
			result.Warnings = append(result.Warnings, "skipped plugins entry: expected TOML table")
			continue
		}
		enabled, ok := table["enabled"].(bool)
		if !ok {
			result.Warnings = append(result.Warnings, "skipped plugins entry: enabled is not a boolean")
			continue
		}
		pluginName, marketplaceName, ok := parseClaudePluginKey(key)
		if !ok {
			result.Warnings = append(result.Warnings, "skipped plugins entry: key format must be name@marketplace")
			continue
		}
		if len(pluginName) > ClaudeSettingsMaxNameLen || len(marketplaceName) > ClaudeSettingsMaxNameLen {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipped plugins entry: name or marketplace exceeds %d chars", ClaudeSettingsMaxNameLen))
			continue
		}
		result.Plugins = append(result.Plugins, ClaudePluginDecl{
			PluginName:      pluginName,
			MarketplaceName: marketplaceName,
			Enabled:         enabled,
		})
		count++
	}
}

func parseCodexMarketplaces(raw map[string]interface{}, result *ClaudeSettingsScan) {
	marketplacesRaw, exists := raw["marketplaces"]
	if !exists {
		return
	}
	marketplaces, ok := marketplacesRaw.(map[string]interface{})
	if !ok {
		result.Warnings = append(result.Warnings, "skipped marketplaces section: expected TOML table")
		return
	}
	count := 0
	for name, val := range marketplaces {
		if count >= ClaudeSettingsMaxMarketplaces {
			result.Warnings = append(result.Warnings, "marketplaces truncated at 200 entries")
			break
		}
		if name == "" || len(name) > ClaudeSettingsMaxNameLen {
			continue
		}
		table, _ := val.(map[string]interface{})
		sourceType := "settings"
		sourceSummary := ""
		for _, k := range []string{"source", "url", "path"} {
			if s, ok := table[k].(string); ok {
				sourceSummary = s
				break
			}
		}
		if t, ok := table["type"].(string); ok && t != "" {
			sourceType = codexNormalizeSourceType(t)
		}
		if len(sourceSummary) > ClaudeSettingsMaxSummaryLen {
			sourceSummary = sourceSummary[:ClaudeSettingsMaxSummaryLen]
		}
		result.Marketplaces = append(result.Marketplaces, ClaudeMarketplaceDecl{
			MarketplaceName: name,
			SourceType:      sourceType,
			SourceSummary:   sourceSummary,
		})
		count++
	}
}

func codexNormalizeSourceType(t string) string {
	switch t {
	case "github", "git", "directory", "url", "settings", "hostPattern":
		return t
	default:
		return "unknown"
	}
}
