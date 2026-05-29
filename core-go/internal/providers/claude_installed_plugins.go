package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const (
	claudeInstalledPluginsMaxFileSize = ClaudeSettingsMaxFileSize // reuse 1 MiB cap
	claudeInstalledPluginsMaxEntries  = 2000                      // reasonable upper bound
)

// ClaudeInstalledPluginsScan is the result of scanning ~/.claude/plugins/installed_plugins.json.
// Status mirrors the same set used by ClaudeSettingsScan.
type ClaudeInstalledPluginsScan struct {
	Status   string
	Entries  []ClaudeInstalledPluginEntry
	Warnings []string
}

// ClaudeInstalledPluginEntry is a single plugin→scope→version record from installed_plugins.json.
type ClaudeInstalledPluginEntry struct {
	PluginKey   string  // "pluginName@marketplaceName"
	Scope       string  // "user" | "project" | "local"
	ProjectPath string  // absolute path; set for scope "project" and "local", empty for "user"
	Version     *string // nil when JSON null or field absent; "unknown" is a valid literal
}

// ScanClaudeInstalledPluginsFile reads and parses ~/.claude/plugins/installed_plugins.json.
// filePath must be confined within allowedDir (which should be the Claude config root, e.g.
// ~/.claude — same root as the settings.json that triggered this scan). Symlinks, size
// violations, and path escapes are rejected, mirroring ScanClaudeSettingsFile security posture.
func ScanClaudeInstalledPluginsFile(filePath, allowedDir string) ClaudeInstalledPluginsScan {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	// Path confinement: reject if filePath escapes allowedDir
	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return ClaudeInstalledPluginsScan{Status: "path_escape"}
	}

	// Symlink check on parent directory
	parentDir := filepath.Dir(cleanFile)
	if lfi, err := os.Lstat(parentDir); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		return ClaudeInstalledPluginsScan{Status: "symlink"}
	}

	fi, err := os.Lstat(cleanFile)
	if err != nil {
		if os.IsNotExist(err) {
			return ClaudeInstalledPluginsScan{Status: "missing"}
		}
		return ClaudeInstalledPluginsScan{Status: "unreadable"}
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return ClaudeInstalledPluginsScan{Status: "symlink"}
	}
	if fi.Size() > claudeInstalledPluginsMaxFileSize {
		return ClaudeInstalledPluginsScan{Status: "too_large"}
	}

	data, err := os.ReadFile(cleanFile)
	if err != nil {
		return ClaudeInstalledPluginsScan{Status: "unreadable"}
	}

	// Top-level must be an object
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return ClaudeInstalledPluginsScan{Status: "malformed"}
	}

	result := ClaudeInstalledPluginsScan{Status: "ok"}

	// "plugins" is optional; absence is treated as 0 entries (not an error).
	pluginsRaw, exists := raw["plugins"]
	if !exists {
		return result
	}

	// "plugins" must be an object keyed by "pluginName@marketplaceName"
	var pluginsMap map[string]json.RawMessage
	if err := json.Unmarshal(pluginsRaw, &pluginsMap); err != nil {
		result.Warnings = append(result.Warnings, "installed_plugins.json: plugins field is not an object; skipping versions")
		return result
	}

	count := 0
	for key, scopesRaw := range pluginsMap {
		if count >= claudeInstalledPluginsMaxEntries {
			result.Warnings = append(result.Warnings, "installed_plugins.json: entries truncated at 2000")
			break
		}
		// Each value is an array of scope objects
		var scopes []json.RawMessage
		if err := json.Unmarshal(scopesRaw, &scopes); err != nil {
			continue // tolerate unexpected shapes
		}
		for _, scopeRaw := range scopes {
			var obj struct {
				Scope       string      `json:"scope"`
				ProjectPath string      `json:"projectPath"` // present for "project" and "local" scopes
				Version     interface{} `json:"version"`     // may be string, null, or absent
			}
			if err := json.Unmarshal(scopeRaw, &obj); err != nil {
				continue
			}
			entry := ClaudeInstalledPluginEntry{
				PluginKey:   key,
				Scope:       obj.Scope,
				ProjectPath: obj.ProjectPath,
			}
			// Version field: string → keep; null or absent → nil (E3a: JSON null treated as nil)
			if s, ok := obj.Version.(string); ok && s != "" {
				entry.Version = &s
			}
			result.Entries = append(result.Entries, entry)
			count++
		}
	}

	return result
}

// BuildVersionMap returns a map of "pluginKey" → version for entries with scope == "user".
// Callers use this to annotate plugin entries from settings.json.
func BuildVersionMap(scan ClaudeInstalledPluginsScan) map[string]*string {
	m := make(map[string]*string, len(scan.Entries))
	if scan.Status != "ok" {
		return m
	}
	for _, e := range scan.Entries {
		if e.Scope == "user" {
			// Keep the first user-scope entry; later ones are not expected but ignored
			if _, exists := m[e.PluginKey]; !exists {
				m[e.PluginKey] = e.Version
			}
		}
	}
	return m
}

// BuildProjectVersionMap returns a map of "pluginKey" → version for entries with
// scope "project" or "local" whose projectPath matches the given project directory.
// projectPath is cleaned with filepath.Clean before comparison.
func BuildProjectVersionMap(scan ClaudeInstalledPluginsScan, projectPath string) map[string]*string {
	m := make(map[string]*string)
	if scan.Status != "ok" || projectPath == "" {
		return m
	}
	cleanProject := filepath.Clean(projectPath)
	for _, e := range scan.Entries {
		if (e.Scope == "project" || e.Scope == "local") && filepath.Clean(e.ProjectPath) == cleanProject {
			if _, exists := m[e.PluginKey]; !exists {
				m[e.PluginKey] = e.Version
			}
		}
	}
	return m
}
