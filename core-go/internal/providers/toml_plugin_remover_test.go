package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveTOMLPlugin_NoOpWhenFileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := RemoveTOMLPlugin(path, dir, "my-plugin", "my-market"); err != nil {
		t.Fatalf("expected no error when file missing, got: %v", err)
	}
}

func TestRemoveTOMLPlugin_RemovesDottedKey(t *testing.T) {
	dir := t.TempDir()
	content := `[plugins]
"foo@bar".enabled = true
"keep@it".enabled = false
`
	path := writeSettings(t, dir, "config.toml", content)
	if err := RemoveTOMLPlugin(path, dir, "foo", "bar"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "foo@bar") {
		t.Error("foo@bar should have been removed")
	}
	if !strings.Contains(string(data), "keep@it") {
		t.Error("keep@it should be preserved")
	}
}

func TestRemoveTOMLPlugin_RemovesTableSection(t *testing.T) {
	dir := t.TempDir()
	content := `[plugins."foo@bar"]
enabled = true

[plugins."keep@it"]
enabled = false
`
	path := writeSettings(t, dir, "config.toml", content)
	if err := RemoveTOMLPlugin(path, dir, "foo", "bar"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "foo@bar") {
		t.Error("foo@bar section should have been removed")
	}
	if !strings.Contains(string(data), "keep@it") {
		t.Error("keep@it should be preserved")
	}
}

func TestRemoveTOMLPlugin_NoOpWhenKeyAbsent(t *testing.T) {
	dir := t.TempDir()
	content := `[plugins]
"other@mkt".enabled = true
`
	path := writeSettings(t, dir, "config.toml", content)
	if err := RemoveTOMLPlugin(path, dir, "missing", "mkt"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "other@mkt") {
		t.Error("other@mkt should be preserved")
	}
}

func TestRemoveTOMLPlugin_PathEscape(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	path := filepath.Join(other, "config.toml")
	err := RemoveTOMLPlugin(path, dir, "p", "m")
	if err == nil {
		t.Error("expected path_escape error, got nil")
	}
}
