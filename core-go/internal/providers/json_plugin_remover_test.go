package providers

import (
	"path/filepath"
	"testing"
)

func TestRemoveJSONPlugin_NoOpWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := RemoveJSONPlugin(path, dir, "my-plugin", "my-market"); err != nil {
		t.Fatalf("expected no error when file missing, got: %v", err)
	}
}

func TestRemoveJSONPlugin_RemovesExistingKey(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, "settings.json", `{"enabledPlugins":{"foo@bar":true,"keep@it":false}}`)
	if err := RemoveJSONPlugin(path, dir, "foo", "bar"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	plugins := readEnabledPlugins(t, path)
	if _, exists := plugins["foo@bar"]; exists {
		t.Error("foo@bar should have been removed")
	}
	if plugins["keep@it"] != false {
		t.Error("keep@it should be preserved")
	}
}

func TestRemoveJSONPlugin_NoOpWhenKeyAbsent(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, "settings.json", `{"enabledPlugins":{"other@mkt":true}}`)
	if err := RemoveJSONPlugin(path, dir, "missing", "mkt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	plugins := readEnabledPlugins(t, path)
	if plugins["other@mkt"] != true {
		t.Error("other@mkt should be preserved")
	}
}

func TestRemoveJSONPlugin_PreservesOtherTopLevelKeys(t *testing.T) {
	dir := t.TempDir()
	path := writeSettings(t, dir, "settings.json", `{"someKey":"val","enabledPlugins":{"rm@me":true}}`)
	if err := RemoveJSONPlugin(path, dir, "rm", "me"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	top := readSettings(t, path)
	if _, ok := top["someKey"]; !ok {
		t.Error("someKey was removed from settings file")
	}
}

func TestRemoveJSONPlugin_PathEscape(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	path := filepath.Join(other, "settings.json")
	err := RemoveJSONPlugin(path, dir, "p", "m")
	if err == nil {
		t.Error("expected path_escape error, got nil")
	}
}
