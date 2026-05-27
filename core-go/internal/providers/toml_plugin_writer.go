package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

// package-level regexes that don't depend on pluginKey.
var (
	tomlAnyHeaderRe     = regexp.MustCompile(`^\s*\[`)
	tomlPluginsHeaderRe = regexp.MustCompile(`^\s*\[plugins\]\s*(?:#.*)?$`)
	tomlEnabledLineRe   = regexp.MustCompile(`^(\s*enabled\s*=\s*)(true|false)(\s*(?:#.*)?)$`)
)

// WriteTOMLPluginEnabled mutates the enabled state of a plugin in a Codex-style
// config.toml file, preserving comments, ordering, and all unrelated keys.
// Applies the same safety preflight as WriteJSONPluginEnabled.
func WriteTOMLPluginEnabled(filePath, allowedDir, pluginName, marketplaceName string, enabled bool) error {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return &pluginWriteError{"path_escape", "settings file path escapes allowed directory"}
	}

	parentDir := filepath.Dir(cleanFile)

	if _, err := os.Lstat(parentDir); os.IsNotExist(err) {
		if err := filesystem.EnsureDirSafe(parentDir); err != nil {
			return &pluginWriteError{"unwritable", "could not create settings directory"}
		}
	}

	if lfi, err := os.Lstat(parentDir); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		return &pluginWriteError{"symlink", "parent directory is a symlink"}
	}

	fi, err := os.Lstat(cleanFile)
	if err != nil && !os.IsNotExist(err) {
		return &pluginWriteError{"unreadable", "could not stat settings file"}
	}

	var existingData []byte
	if fi != nil {
		if fi.Mode()&os.ModeSymlink != 0 {
			return &pluginWriteError{"symlink", "settings file is a symlink"}
		}
		if fi.Size() > ClaudeSettingsMaxFileSize {
			return &pluginWriteError{"too_large", "settings file exceeds 1 MiB"}
		}
		existingData, err = os.ReadFile(cleanFile)
		if err != nil {
			return &pluginWriteError{"unreadable", "could not read settings file"}
		}
	}

	updated, err := applyTOMLPluginEnabled(existingData, pluginName, marketplaceName, enabled)
	if err != nil {
		return err
	}

	if err := filesystem.WriteFileAtomic(cleanFile, updated, 0o644); err != nil {
		return &pluginWriteError{"unwritable", "could not write settings file"}
	}
	return nil
}

// applyTOMLPluginEnabled returns updated TOML bytes with the plugin enabled key toggled,
// preserving all comments, whitespace, and unrelated keys.
func applyTOMLPluginEnabled(existing []byte, pluginName, marketplaceName string, enabled bool) ([]byte, error) {
	pluginKey := pluginName + "@" + marketplaceName
	enabledVal := "true"
	if !enabled {
		enabledVal = "false"
	}

	// Validate existing content.
	if len(existing) > 0 {
		var raw map[string]interface{}
		if err := toml.Unmarshal(existing, &raw); err != nil {
			return nil, &pluginWriteError{"malformed", "settings file is not valid TOML"}
		}
	}

	lines := strings.Split(string(existing), "\n")

	// Case 1: look for [plugins."key"] or [plugins.'key'] section header.
	sectionIdx := tomlFindPluginSectionHeader(lines, pluginKey)
	if sectionIdx >= 0 {
		lines = tomlUpdateEnabledInSection(lines, sectionIdx, enabledVal)
	} else {
		// Case 2/3: look for [plugins] section.
		pluginsIdx := -1
		for i, line := range lines {
			if tomlPluginsHeaderRe.MatchString(line) {
				pluginsIdx = i
				break
			}
		}

		if pluginsIdx >= 0 {
			end := tomlSectionEnd(lines, pluginsIdx+1)
			dottedRe := tomlDottedKeyEnabledRe(pluginKey)
			inlineRe := tomlInlineTableKeyRe(pluginKey)

			found := false
			for i := pluginsIdx + 1; i < end; i++ {
				if dottedRe.MatchString(lines[i]) {
					lines[i] = tomlReplaceDottedEnabled(lines[i], enabledVal)
					found = true
					break
				}
				if inlineRe.MatchString(lines[i]) {
					lines[i] = tomlUpdateInlineEnabled(lines[i], enabledVal)
					found = true
					break
				}
			}

			if !found {
				// Insert immediately after [plugins] header.
				newLine := fmt.Sprintf("%q = { enabled = %s }", pluginKey, enabledVal)
				lines = tomlInsertAfterLine(lines, pluginsIdx, newLine)
			}
		} else {
			// Append new section at end.
			lines = tomlAppendSection(lines, pluginKey, enabledVal)
		}
	}

	result := []byte(strings.Join(lines, "\n"))

	// Validate the result is valid TOML.
	var check map[string]interface{}
	if err := toml.Unmarshal(result, &check); err != nil {
		return nil, &pluginWriteError{"internal", fmt.Sprintf("produced invalid TOML: %v", err)}
	}

	return result, nil
}

// tomlFindPluginSectionHeader finds [plugins."key"] or [plugins.'key'] and returns
// its line index, or -1 if not found.
func tomlFindPluginSectionHeader(lines []string, pluginKey string) int {
	esc := regexp.QuoteMeta(pluginKey)
	re := regexp.MustCompile(`^\s*\[plugins\.(?:"` + esc + `"|'` + esc + `')\]\s*(?:#.*)?$`)
	for i, line := range lines {
		if re.MatchString(line) {
			return i
		}
	}
	return -1
}

// tomlSectionEnd returns the index of the first line at or after start that begins
// a new TOML table header, or len(lines) if none.
func tomlSectionEnd(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		if tomlAnyHeaderRe.MatchString(lines[i]) {
			return i
		}
	}
	return len(lines)
}

// tomlUpdateEnabledInSection updates (or inserts) the enabled = val key inside the
// section that starts at headerIdx.
func tomlUpdateEnabledInSection(lines []string, headerIdx int, val string) []string {
	end := tomlSectionEnd(lines, headerIdx+1)
	for i := headerIdx + 1; i < end; i++ {
		if tomlEnabledLineRe.MatchString(lines[i]) {
			lines[i] = tomlEnabledLineRe.ReplaceAllString(lines[i], "${1}"+val+"${3}")
			return lines
		}
	}
	// Not found: insert right after header.
	return tomlInsertAfterLine(lines, headerIdx, "enabled = "+val)
}

// tomlDottedKeyEnabledRe matches lines like `"key".enabled = val` under [plugins].
func tomlDottedKeyEnabledRe(pluginKey string) *regexp.Regexp {
	esc := regexp.QuoteMeta(pluginKey)
	return regexp.MustCompile(`^\s*(?:"` + esc + `"|'` + esc + `')\.enabled\s*=`)
}

// tomlInlineTableKeyRe matches lines like `"key" = { ... }` under [plugins].
func tomlInlineTableKeyRe(pluginKey string) *regexp.Regexp {
	esc := regexp.QuoteMeta(pluginKey)
	return regexp.MustCompile(`^\s*(?:"` + esc + `"|'` + esc + `')\s*=\s*\{`)
}

// tomlReplaceDottedEnabled replaces the value in a dotted key line like
// `"key".enabled = true  # comment` → `"key".enabled = false  # comment`.
func tomlReplaceDottedEnabled(line, val string) string {
	re := regexp.MustCompile(`(\.enabled\s*=\s*)(true|false)(\s*(?:#.*)?)$`)
	return re.ReplaceAllString(line, "${1}"+val+"${3}")
}

// tomlUpdateInlineEnabled updates or inserts enabled inside an inline table value.
func tomlUpdateInlineEnabled(line, val string) string {
	re := regexp.MustCompile(`(\benabled\s*=\s*)(true|false)`)
	if re.MatchString(line) {
		return re.ReplaceAllString(line, "${1}"+val)
	}
	// enabled not present: insert before closing brace.
	i := strings.LastIndex(line, "}")
	if i < 0 {
		return line
	}
	openBrace := strings.Index(line, "{")
	if openBrace < 0 {
		return line
	}
	inner := strings.TrimSpace(line[openBrace+1 : i])
	if inner == "" {
		return line[:i] + "enabled = " + val + " }"
	}
	return line[:i] + ", enabled = " + val + " }"
}

// tomlInsertAfterLine returns lines with newLine inserted after index idx.
func tomlInsertAfterLine(lines []string, idx int, newLine string) []string {
	result := make([]string, 0, len(lines)+1)
	result = append(result, lines[:idx+1]...)
	result = append(result, newLine)
	result = append(result, lines[idx+1:]...)
	return result
}

// tomlAppendSection appends a new [plugins."key"] section to lines.
func tomlAppendSection(lines []string, pluginKey, val string) []string {
	// Ensure content ends with a newline (lines slice ends with empty string from Split).
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		// Empty file.
		return []string{
			fmt.Sprintf("[plugins.%q]", pluginKey),
			"enabled = " + val,
			"",
		}
	}
	// Ensure trailing newline before the new section.
	if lines[len(lines)-1] != "" {
		lines = append(lines, "")
	}
	// Blank separator line between existing content and new section.
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("[plugins.%q]", pluginKey))
	lines = append(lines, "enabled = "+val)
	lines = append(lines, "")
	return lines
}
