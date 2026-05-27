package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

// WriteJSONPluginEnabled mutates enabledPlugins["pluginName@marketplaceName"] in a
// Claude/Antigravity-style settings.json file. It applies the same safety preflight
// as the scanner: path confinement, symlink rejection on parent and file, and
// 1 MiB file size cap. If the file is missing it is created with a minimal JSON
// object. All other top-level keys are preserved as raw bytes.
//
// Returns a filesystem_error-compatible error string; callers may wrap it with
// domain.NewFilesystemError.
func WriteJSONPluginEnabled(filePath, allowedDir, pluginName, marketplaceName string, enabled bool) error {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	// Path confinement: filePath must be within allowedDir.
	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return &pluginWriteError{"path_escape", "settings file path escapes allowed directory"}
	}

	parentDir := filepath.Dir(cleanFile)

	// If the parent directory doesn't exist yet, create it safely.
	if _, err := os.Lstat(parentDir); os.IsNotExist(err) {
		if err := filesystem.EnsureDirSafe(parentDir); err != nil {
			return &pluginWriteError{"unwritable", "could not create settings directory"}
		}
	}

	// Symlink check on parent directory.
	if lfi, err := os.Lstat(parentDir); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		return &pluginWriteError{"symlink", "parent directory is a symlink"}
	}

	// Check for file.
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

	updated, err := applyJSONPluginEnabled(existingData, pluginName, marketplaceName, enabled)
	if err != nil {
		return err
	}

	if err := filesystem.WriteFileAtomic(cleanFile, updated, 0o644); err != nil {
		return &pluginWriteError{"unwritable", "could not write settings file"}
	}
	return nil
}

// applyJSONPluginEnabled returns new JSON bytes with the single plugin key toggled.
// It preserves all other top-level keys as raw bytes using map[string]json.RawMessage.
func applyJSONPluginEnabled(existing []byte, pluginName, marketplaceName string, enabled bool) ([]byte, error) {
	pluginKey := pluginName + "@" + marketplaceName

	// Parse the top-level object, preserving all keys as raw bytes.
	top := make(map[string]json.RawMessage)
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &top); err != nil {
			return nil, &pluginWriteError{"malformed", "settings file is not valid JSON"}
		}
	}

	// Decode or create enabledPlugins.
	plugins := make(map[string]bool)
	if raw, ok := top["enabledPlugins"]; ok {
		if err := json.Unmarshal(raw, &plugins); err != nil {
			// Non-object enabledPlugins is malformed per scanner semantics.
			return nil, &pluginWriteError{"malformed", "enabledPlugins is not a JSON object"}
		}
	}

	plugins[pluginKey] = enabled

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

// pluginWriteError is a simple error that carries a status tag for error mapping.
type pluginWriteError struct {
	status  string
	message string
}

func (e *pluginWriteError) Error() string {
	return e.status + ": " + e.message
}

func (e *pluginWriteError) Status() string {
	return e.status
}
