package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	ClaudeSettingsMaxFileSize     int64 = 1 * 1024 * 1024 // 1 MiB
	ClaudeSettingsMaxPlugins            = 1000
	ClaudeSettingsMaxMarketplaces       = 200
	ClaudeSettingsMaxNameLen            = 256
	ClaudeSettingsMaxSummaryLen         = 512
)

// ClaudeSettingsScan is the result of scanning one Claude settings.json file.
// Status is one of: ok | missing | unreadable | malformed | too_large | symlink | path_escape.
type ClaudeSettingsScan struct {
	Status       string
	Plugins      []ClaudePluginDecl
	Marketplaces []ClaudeMarketplaceDecl
	Warnings     []string
}

// ClaudePluginDecl is a single entry from enabledPlugins.
type ClaudePluginDecl struct {
	PluginName      string
	MarketplaceName string
	Enabled         bool
}

// ClaudeMarketplaceDecl is a single entry from extraKnownMarketplaces.
type ClaudeMarketplaceDecl struct {
	MarketplaceName string
	SourceType      string // github | git | directory | url | settings | hostPattern | unknown
	SourceSummary   string // bounded, sanitized; never raw bytes
}

// ScanClaudeSettingsFile reads and parses a Claude settings.json file with security bounds.
// filePath must be confined within allowedDir. Symlinks, size violations, and path escapes are rejected.
// Raw file content is never stored or logged.
func ScanClaudeSettingsFile(filePath, allowedDir string) ClaudeSettingsScan {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	// Path confinement: reject if filePath escapes allowedDir
	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return ClaudeSettingsScan{Status: "path_escape"}
	}

	// Symlink check on parent directory
	parentDir := filepath.Dir(cleanFile)
	if lfi, err := os.Lstat(parentDir); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		return ClaudeSettingsScan{Status: "symlink"}
	}

	// Lstat to detect symlinks without following them
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

	// Top-level JSON must be an object
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return ClaudeSettingsScan{Status: "malformed"}
	}

	result := ClaudeSettingsScan{Status: "ok"}

	// enabledPlugins must be object of string -> bool; non-object → malformed
	if ep, exists := raw["enabledPlugins"]; exists {
		epMap, ok := ep.(map[string]interface{})
		if !ok {
			return ClaudeSettingsScan{Status: "malformed"}
		}
		count := 0
		for key, val := range epMap {
			if count >= ClaudeSettingsMaxPlugins {
				result.Warnings = append(result.Warnings, "enabledPlugins truncated at 1000 entries")
				break
			}
			boolVal, ok := val.(bool)
			if !ok {
				result.Warnings = append(result.Warnings, fmt.Sprintf("skipped non-bool value for key %q", key))
				continue
			}
			pName, mName, ok := parseClaudePluginKey(key)
			if !ok {
				result.Warnings = append(result.Warnings, fmt.Sprintf("skipped invalid plugin key %q: must be name@marketplace", key))
				continue
			}
			if len(pName) > ClaudeSettingsMaxNameLen || len(mName) > ClaudeSettingsMaxNameLen {
				result.Warnings = append(result.Warnings, fmt.Sprintf("skipped plugin key %q: name or marketplace exceeds %d chars", key, ClaudeSettingsMaxNameLen))
				continue
			}
			result.Plugins = append(result.Plugins, ClaudePluginDecl{
				PluginName:      pName,
				MarketplaceName: mName,
				Enabled:         boolVal,
			})
			count++
		}
	}

	// extraKnownMarketplaces is an optional array of objects
	if ekm, exists := raw["extraKnownMarketplaces"]; exists {
		ekmSlice, ok := ekm.([]interface{})
		if ok {
			count := 0
			for _, item := range ekmSlice {
				if count >= ClaudeSettingsMaxMarketplaces {
					result.Warnings = append(result.Warnings, "extraKnownMarketplaces truncated at 200 entries")
					break
				}
				mp, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				name, _ := mp["name"].(string)
				if name == "" || len(name) > ClaudeSettingsMaxNameLen {
					continue
				}
				sourceType := claudeNormalizeSourceType(mp)
				sourceSummary := claudeBuildSummary(mp)
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
	}

	return result
}

// parseClaudePluginKey splits "pluginName@marketplaceName" at the last @.
// Returns false if either part is empty or @ is missing.
func parseClaudePluginKey(key string) (pluginName, marketplaceName string, ok bool) {
	idx := strings.LastIndex(key, "@")
	if idx <= 0 || idx >= len(key)-1 {
		return "", "", false
	}
	return key[:idx], key[idx+1:], true
}

func claudeNormalizeSourceType(mp map[string]interface{}) string {
	t, _ := mp["type"].(string)
	switch t {
	case "github", "git", "directory", "url", "settings", "hostPattern":
		return t
	default:
		return "unknown"
	}
}

func claudeBuildSummary(mp map[string]interface{}) string {
	t, _ := mp["type"].(string)
	switch t {
	case "github":
		org, _ := mp["githubOrg"].(string)
		repo, _ := mp["githubRepo"].(string)
		if org != "" && repo != "" {
			return fmt.Sprintf("%s/%s", org, repo)
		}
	case "git", "url":
		u, _ := mp["url"].(string)
		return u
	case "directory":
		p, _ := mp["path"].(string)
		return p
	}
	return ""
}
