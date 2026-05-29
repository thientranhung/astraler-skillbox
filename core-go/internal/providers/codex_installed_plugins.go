package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const codexPluginJSONMaxSize = 64 * 1024 // 64 KiB — plugin.json is small

// CodexCacheEntry is a single plugin record resolved from the Codex cache dir.
type CodexCacheEntry struct {
	PluginKey string  // "pluginName@marketplaceName"
	Version   *string // nil when no version could be determined
}

// CodexCacheScan is the result of scanning ~/.codex/plugins/cache/.
type CodexCacheScan struct {
	Status   string
	Entries  []CodexCacheEntry
	Warnings []string
}

// ScanCodexCacheDir reads ~/.codex/plugins/cache/<marketplace>/<plugin>/<version-or-sha>/
// and resolves a version string for each plugin found. cacheDir must be confined within
// allowedDir (which should be ~/.codex). Applies the same security posture as
// ScanClaudeInstalledPluginsFile: path confinement, lstat-only, symlink rejection, size cap.
//
// Version resolution order per plugin:
//  1. plugin.json "version" field (authoritative, used by semver plugins).
//  2. Cache dir name verbatim (e.g. "1.0.0" or short-SHA "9b3c8689").
func ScanCodexCacheDir(cacheDir, allowedDir string) CodexCacheScan {
	cleanCache := filepath.Clean(cacheDir)
	cleanAllowed := filepath.Clean(allowedDir)

	if cleanCache != cleanAllowed && !strings.HasPrefix(cleanCache, cleanAllowed+string(os.PathSeparator)) {
		return CodexCacheScan{Status: "path_escape"}
	}

	fi, err := os.Lstat(cleanCache)
	if err != nil {
		if os.IsNotExist(err) {
			return CodexCacheScan{Status: "missing"}
		}
		return CodexCacheScan{Status: "unreadable"}
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return CodexCacheScan{Status: "symlink"}
	}
	if !fi.IsDir() {
		return CodexCacheScan{Status: "not_a_dir"}
	}

	marketplaceDirs, err := os.ReadDir(cleanCache)
	if err != nil {
		return CodexCacheScan{Status: "unreadable"}
	}

	result := CodexCacheScan{Status: "ok"}

	for _, mEntry := range marketplaceDirs {
		marketplacePath := filepath.Join(cleanCache, mEntry.Name())
		mfi, err := os.Lstat(marketplacePath)
		if err != nil || mfi.Mode()&os.ModeSymlink != 0 || !mfi.IsDir() {
			continue
		}
		marketplaceName := mEntry.Name()
		if marketplaceName == "" || len(marketplaceName) > ClaudeSettingsMaxNameLen {
			continue
		}

		pluginDirs, err := os.ReadDir(marketplacePath)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("skipped marketplace %q: readdir error", marketplaceName))
			continue
		}

		for _, pEntry := range pluginDirs {
			pluginPath := filepath.Join(marketplacePath, pEntry.Name())
			pfi, err := os.Lstat(pluginPath)
			if err != nil {
				continue
			}
			if pfi.Mode()&os.ModeSymlink != 0 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("skipped symlink: %s/%s", marketplaceName, pEntry.Name()))
				continue
			}
			if !pfi.IsDir() {
				continue
			}
			pluginName := pEntry.Name()
			if pluginName == "" || len(pluginName) > ClaudeSettingsMaxNameLen {
				continue
			}

			version := resolveCodexPluginVersion(pluginPath, &result)
			result.Entries = append(result.Entries, CodexCacheEntry{
				PluginKey: pluginName + "@" + marketplaceName,
				Version:   version,
			})
		}
	}
	return result
}

// resolveCodexPluginVersion finds the version-or-SHA subdirectory under pluginPath and
// resolves a version string. Returns nil only when no subdirectory exists.
func resolveCodexPluginVersion(pluginPath string, result *CodexCacheScan) *string {
	versionDirs, err := os.ReadDir(pluginPath)
	if err != nil {
		return nil
	}
	for _, vEntry := range versionDirs {
		versionPath := filepath.Join(pluginPath, vEntry.Name())
		vfi, err := os.Lstat(versionPath)
		if err != nil || vfi.Mode()&os.ModeSymlink != 0 || !vfi.IsDir() {
			continue
		}
		dirName := vEntry.Name()
		if dirName == "" {
			continue
		}

		// Try plugin.json first (semver plugins have it; git-source plugins do not).
		if v := readPluginJSONVersion(versionPath); v != nil {
			return v
		}

		// Fall back to dir name verbatim (semver or short-SHA).
		v := dirName
		return &v
	}
	return nil
}

// readPluginJSONVersion attempts to read "version" from plugin.json inside versionDir.
// Returns nil on any error or when the version field is absent/empty.
func readPluginJSONVersion(versionDir string) *string {
	jsonPath := filepath.Join(versionDir, "plugin.json")
	fi, err := os.Lstat(jsonPath)
	if err != nil {
		return nil
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return nil
	}
	if fi.Size() > codexPluginJSONMaxSize {
		return nil
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil
	}
	var obj struct {
		Version string `json:"version"`
	}
	if json.Unmarshal(data, &obj) != nil || obj.Version == "" {
		return nil
	}
	v := obj.Version
	return &v
}

// BuildCodexVersionMap returns a map of "pluginKey" → *version from a CodexCacheScan.
// Used by ProviderPluginService to annotate Codex plugin entries with version strings.
func BuildCodexVersionMap(scan CodexCacheScan) map[string]*string {
	m := make(map[string]*string, len(scan.Entries))
	if scan.Status != "ok" {
		return m
	}
	for _, e := range scan.Entries {
		if _, exists := m[e.PluginKey]; !exists {
			m[e.PluginKey] = e.Version
		}
	}
	return m
}
