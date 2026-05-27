package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeSettings(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	return path
}

func readSettings(t *testing.T, path string) map[string]json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal settings: %v", err)
	}
	return m
}

func readEnabledPlugins(t *testing.T, path string) map[string]bool {
	t.Helper()
	top := readSettings(t, path)
	var plugins map[string]bool
	if raw, ok := top["enabledPlugins"]; ok {
		if err := json.Unmarshal(raw, &plugins); err != nil {
			t.Fatalf("unmarshal enabledPlugins: %v", err)
		}
	}
	if plugins == nil {
		plugins = map[string]bool{}
	}
	return plugins
}

func TestWriteJSONPluginEnabled_CreatesFileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := WriteJSONPluginEnabled(path, dir, "my-plugin", "my-market", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	plugins := readEnabledPlugins(t, path)
	if plugins["my-plugin@my-market"] != true {
		t.Errorf("expected true, got %v", plugins["my-plugin@my-market"])
	}
}

func TestWriteJSONPluginEnabled_TogglesExistingPlugin(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, "settings.json", `{"enabledPlugins":{"foo@bar":true}}`)
	if err := WriteJSONPluginEnabled(path, dir, "foo", "bar", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	plugins := readEnabledPlugins(t, path)
	if plugins["foo@bar"] != false {
		t.Errorf("expected false, got %v", plugins["foo@bar"])
	}
}

func TestWriteJSONPluginEnabled_PreservesUnrelatedKeys(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, "settings.json", `{"someOtherKey":"keep-me","enabledPlugins":{}}`)
	if err := WriteJSONPluginEnabled(path, dir, "p", "m", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	top := readSettings(t, path)
	if _, ok := top["someOtherKey"]; !ok {
		t.Error("someOtherKey was removed from settings file")
	}
}

func TestWriteJSONPluginEnabled_PathEscape(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	path := filepath.Join(other, "settings.json")
	err := WriteJSONPluginEnabled(path, dir, "p", "m", true)
	if err == nil {
		t.Error("expected path_escape error, got nil")
	}
}

func TestWriteJSONPluginEnabled_SymlinkFile(t *testing.T) {
	dir := t.TempDir()
	target := writeSettings(t, dir, "real.json", `{}`)
	link := filepath.Join(dir, "settings.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := WriteJSONPluginEnabled(link, dir, "p", "m", true)
	if err == nil {
		t.Error("expected symlink error, got nil")
	}
}

func TestWriteJSONPluginEnabled_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, "settings.json", `{bad json`)
	err := WriteJSONPluginEnabled(path, dir, "p", "m", true)
	if err == nil {
		t.Error("expected malformed error, got nil")
	}
}

func TestWriteJSONPluginEnabled_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	subDir := filepath.Join(dir, "gemini", "antigravity-cli")
	path := filepath.Join(subDir, "settings.json")
	if err := WriteJSONPluginEnabled(path, subDir, "p", "m", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestWriteJSONPluginEnabled_AddsPluginToExistingMap(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, "settings.json", `{"enabledPlugins":{"existing@market":true}}`)
	if err := WriteJSONPluginEnabled(path, dir, "new-plugin", "new-market", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	plugins := readEnabledPlugins(t, path)
	if plugins["existing@market"] != true {
		t.Errorf("existing plugin was modified unexpectedly")
	}
	if plugins["new-plugin@new-market"] != false {
		t.Errorf("new plugin not written correctly")
	}
}
