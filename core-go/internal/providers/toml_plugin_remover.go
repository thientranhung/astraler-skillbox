package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

// RemoveTOMLPlugin removes a plugin entry from a Codex-style config.toml file.
// It handles both dotted keys (`"key".enabled = true`) and table sections
// (`[plugins."key"]`). If the file does not exist or the key is absent, it is a no-op.
func RemoveTOMLPlugin(filePath, allowedDir, pluginName, marketplaceName string) error {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return &pluginWriteError{"path_escape", "settings file path escapes allowed directory"}
	}

	fi, err := os.Lstat(cleanFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return &pluginWriteError{"unreadable", "could not stat settings file"}
	}

	parentDir := filepath.Dir(cleanFile)
	if lfi, err := os.Lstat(parentDir); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		return &pluginWriteError{"symlink", "parent directory is a symlink"}
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return &pluginWriteError{"symlink", "settings file is a symlink"}
	}
	if fi.Size() > ClaudeSettingsMaxFileSize {
		return &pluginWriteError{"too_large", "settings file exceeds 1 MiB"}
	}

	existingData, err := os.ReadFile(cleanFile)
	if err != nil {
		return &pluginWriteError{"unreadable", "could not read settings file"}
	}

	updated, changed := applyTOMLPluginRemove(string(existingData), pluginName, marketplaceName)
	if !changed {
		return nil
	}

	if err := filesystem.WriteFileAtomic(cleanFile, []byte(updated), 0o644); err != nil {
		return &pluginWriteError{"unwritable", "could not write settings file"}
	}
	return nil
}

// applyTOMLPluginRemove removes lines related to a plugin key from TOML content.
// Returns the updated content and whether any change was made.
func applyTOMLPluginRemove(content, pluginName, marketplaceName string) (string, bool) {
	pluginKey := pluginName + "@" + marketplaceName
	quotedKey := fmt.Sprintf(`"%s"`, pluginKey)

	lines := strings.Split(content, "\n")
	var result []string
	changed := false

	// Build regexes for this plugin key.
	// Match [plugins."key"] table header.
	tableSectionRe := regexp.MustCompile(
		fmt.Sprintf(`^\s*\[plugins\.%s\]\s*(?:#.*)?$`, regexp.QuoteMeta(quotedKey)),
	)
	// Match "key".enabled = ... or "key" = { ... } dotted/inline forms.
	dottedKeyRe := regexp.MustCompile(
		fmt.Sprintf(`^\s*%s\s*[.=]`, regexp.QuoteMeta(quotedKey)),
	)

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Case 1: [plugins."key"] table section — remove until next header.
		if tableSectionRe.MatchString(line) {
			changed = true
			i++ // skip the header
			// Skip all lines until the next header or EOF.
			for i < len(lines) {
				if tomlAnyHeaderRe.MatchString(lines[i]) {
					break
				}
				i++
			}
			// Remove trailing blank lines before next section.
			for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
				result = result[:len(result)-1]
			}
			continue
		}

		// Case 2: dotted key under [plugins] — "key".enabled = ... or "key" = { ... }
		if dottedKeyRe.MatchString(line) {
			changed = true
			i++
			continue
		}

		result = append(result, line)
		i++
	}

	if !changed {
		return content, false
	}

	return strings.Join(result, "\n"), true
}
