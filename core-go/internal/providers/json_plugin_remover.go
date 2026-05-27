package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

// RemoveJSONPlugin removes the plugin key from enabledPlugins in a Claude/Antigravity-
// style settings.json file. If the file does not exist or the key is absent, it is a
// no-op. Applies the same safety preflight as WriteJSONPluginEnabled.
func RemoveJSONPlugin(filePath, allowedDir, pluginName, marketplaceName string) error {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return &pluginWriteError{"path_escape", "settings file path escapes allowed directory"}
	}

	// If file does not exist, nothing to remove.
	fi, err := os.Lstat(cleanFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return &pluginWriteError{"unreadable", "could not stat settings file"}
	}

	// Symlink checks.
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

	updated, err := applyJSONPluginRemove(existingData, pluginName, marketplaceName)
	if err != nil {
		return err
	}
	if updated == nil {
		return nil // key was absent, nothing to write
	}

	if err := filesystem.WriteFileAtomic(cleanFile, updated, 0o644); err != nil {
		return &pluginWriteError{"unwritable", "could not write settings file"}
	}
	return nil
}

// applyJSONPluginRemove removes the plugin key from enabledPlugins and returns the
// updated JSON bytes. Returns nil, nil if the key is absent (no write needed).
func applyJSONPluginRemove(existing []byte, pluginName, marketplaceName string) ([]byte, error) {
	pluginKey := pluginName + "@" + marketplaceName

	top := make(map[string]json.RawMessage)
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &top); err != nil {
			return nil, &pluginWriteError{"malformed", "settings file is not valid JSON"}
		}
	}

	raw, ok := top["enabledPlugins"]
	if !ok {
		return nil, nil // no enabledPlugins section at all
	}

	plugins := make(map[string]bool)
	if err := json.Unmarshal(raw, &plugins); err != nil {
		return nil, &pluginWriteError{"malformed", "enabledPlugins is not a JSON object"}
	}

	if _, exists := plugins[pluginKey]; !exists {
		return nil, nil // key already absent
	}

	delete(plugins, pluginKey)

	rawPlugins, err := json.Marshal(plugins)
	if err != nil {
		return nil, &pluginWriteError{"internal", "could not marshal enabledPlugins"}
	}
	top["enabledPlugins"] = json.RawMessage(rawPlugins)

	out, err := json.MarshalIndent(top, "", "  ")
	if err != nil {
		return nil, &pluginWriteError{"internal", "could not marshal settings file"}
	}
	return append(out, '\n'), nil
}
