package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// readTOMLPluginEnabled reads the enabled value for a plugin from a TOML file using
// the same unmarshalling as the scanner.
func readTOMLPluginEnabled(t *testing.T, path, pluginKey string) (bool, bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readTOMLPluginEnabled read: %v", err)
	}
	var raw map[string]interface{}
	if err := toml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("readTOMLPluginEnabled unmarshal: %v", err)
	}
	pluginsRaw, ok := raw["plugins"]
	if !ok {
		return false, false
	}
	plugins, ok := pluginsRaw.(map[string]interface{})
	if !ok {
		return false, false
	}
	entry, ok := plugins[pluginKey]
	if !ok {
		return false, false
	}
	table, ok := entry.(map[string]interface{})
	if !ok {
		return false, false
	}
	enabled, ok := table["enabled"].(bool)
	return enabled, ok
}

// writeTOML writes a TOML string to dir/filename and returns the path.
func writeTOML(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeTOML: %v", err)
	}
	return path
}

// ---- applyTOMLPluginEnabled unit tests (no filesystem) ----

func TestApplyTOML_SectionTable_UpdatesEnabled(t *testing.T) {
	input := `# Global config
[settings]
theme = "dark"

[plugins."my-plugin@npm"]
# This plugin is great
enabled = false
version = "1.0"

[other]
key = "val"
`
	out, err := applyTOMLPluginEnabled([]byte(input), "my-plugin", "npm", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "enabled = true") {
		t.Errorf("expected enabled = true in output:\n%s", text)
	}
	// Comment must be preserved.
	if !strings.Contains(text, "# This plugin is great") {
		t.Errorf("comment lost:\n%s", text)
	}
	// Other sections untouched.
	if !strings.Contains(text, `theme = "dark"`) {
		t.Errorf("unrelated key lost:\n%s", text)
	}
	if !strings.Contains(text, `version = "1.0"`) {
		t.Errorf("sibling key lost:\n%s", text)
	}
}

func TestApplyTOML_SectionTable_PreservesInlineComment(t *testing.T) {
	input := "[plugins.\"p@m\"]\nenabled = true # keep me\n"
	out, err := applyTOMLPluginEnabled([]byte(input), "p", "m", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "enabled = false # keep me") {
		t.Errorf("inline comment lost or value not updated:\n%s", text)
	}
}

func TestApplyTOML_SectionTable_InsertsEnabledWhenMissing(t *testing.T) {
	input := "[plugins.\"p@m\"]\nother = 42\n"
	out, err := applyTOMLPluginEnabled([]byte(input), "p", "m", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "enabled = true") {
		t.Errorf("enabled key not inserted:\n%s", text)
	}
	if !strings.Contains(text, "other = 42") {
		t.Errorf("sibling key lost:\n%s", text)
	}
}

func TestApplyTOML_InlineTable_UpdatesEnabled(t *testing.T) {
	input := `[plugins]
"my-plugin@npm" = { enabled = false, extra = "x" }
`
	out, err := applyTOMLPluginEnabled([]byte(input), "my-plugin", "npm", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "enabled = true") {
		t.Errorf("enabled not updated:\n%s", text)
	}
	if !strings.Contains(text, `extra = "x"`) {
		t.Errorf("sibling key in inline table lost:\n%s", text)
	}
}

func TestApplyTOML_InlineTable_AppendsEnabledWhenMissing(t *testing.T) {
	input := `[plugins]
"p@m" = { other = "val" }
`
	out, err := applyTOMLPluginEnabled([]byte(input), "p", "m", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "enabled = false") {
		t.Errorf("enabled not appended:\n%s", text)
	}
	if !strings.Contains(text, `other = "val"`) {
		t.Errorf("existing key lost:\n%s", text)
	}
	// Validate TOML still parses correctly.
	var check map[string]interface{}
	if err := toml.Unmarshal(out, &check); err != nil {
		t.Errorf("result not valid TOML: %v\n%s", err, text)
	}
}

func TestApplyTOML_DottedKey_UpdatesEnabled(t *testing.T) {
	input := `[plugins]
"my-plugin@npm".enabled = false
"other@m".enabled = true
`
	out, err := applyTOMLPluginEnabled([]byte(input), "my-plugin", "npm", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := string(out)
	// Verify via TOML parse that my-plugin@npm is now enabled.
	var raw map[string]interface{}
	if err := toml.Unmarshal(out, &raw); err != nil {
		t.Fatalf("result not valid TOML: %v\n%s", err, text)
	}
	plugins := raw["plugins"].(map[string]interface{})
	myPlugin := plugins["my-plugin@npm"].(map[string]interface{})
	if myPlugin["enabled"] != true {
		t.Errorf("expected enabled=true for my-plugin@npm, got %v", myPlugin["enabled"])
	}
	// Other plugin untouched.
	other := plugins["other@m"].(map[string]interface{})
	if other["enabled"] != true {
		t.Errorf("other plugin should remain enabled=true")
	}
}

func TestApplyTOML_InsertionUnderPluginsSection(t *testing.T) {
	input := `[plugins]
"existing@m" = { enabled = true }
`
	out, err := applyTOMLPluginEnabled([]byte(input), "new-plugin", "npm", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := string(out)

	// The new entry must appear under [plugins].
	if !strings.Contains(text, `"new-plugin@npm" = { enabled = false }`) {
		t.Errorf("new entry not inserted under [plugins]:\n%s", text)
	}
	// Existing entry must be preserved.
	if !strings.Contains(text, `"existing@m" = { enabled = true }`) {
		t.Errorf("existing entry lost:\n%s", text)
	}
	// Validate TOML.
	var check map[string]interface{}
	if err := toml.Unmarshal(out, &check); err != nil {
		t.Errorf("result not valid TOML: %v\n%s", err, text)
	}
}

func TestApplyTOML_AppendWhenNoSection(t *testing.T) {
	input := `# Codex config
version = 2
`
	out, err := applyTOMLPluginEnabled([]byte(input), "my-plugin", "npm", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := string(out)

	// Original content preserved.
	if !strings.Contains(text, "version = 2") {
		t.Errorf("original content lost:\n%s", text)
	}
	// New section appended.
	if !strings.Contains(text, `[plugins."my-plugin@npm"]`) {
		t.Errorf("section not appended:\n%s", text)
	}
	if !strings.Contains(text, "enabled = true") {
		t.Errorf("enabled not appended:\n%s", text)
	}
	// Validate TOML.
	var check map[string]interface{}
	if err := toml.Unmarshal(out, &check); err != nil {
		t.Errorf("result not valid TOML: %v\n%s", err, text)
	}
}

func TestApplyTOML_EmptyFile_Append(t *testing.T) {
	out, err := applyTOMLPluginEnabled(nil, "p", "m", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var check map[string]interface{}
	if err := toml.Unmarshal(out, &check); err != nil {
		t.Fatalf("result not valid TOML: %v\n%s", err, string(out))
	}
	plugins := check["plugins"].(map[string]interface{})
	entry := plugins["p@m"].(map[string]interface{})
	if entry["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", entry["enabled"])
	}
}

func TestApplyTOML_MalformedInput_ReturnsError(t *testing.T) {
	_, err := applyTOMLPluginEnabled([]byte("not valid [[[toml"), "p", "m", true)
	if err == nil {
		t.Fatal("expected error for malformed TOML, got nil")
	}
	we, ok := err.(*pluginWriteError)
	if !ok || we.Status() != "malformed" {
		t.Errorf("expected malformed error, got %v", err)
	}
}

// ---- WriteTOMLPluginEnabled filesystem integration tests ----

func TestWriteTOMLPluginEnabled_CreatesFileMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := WriteTOMLPluginEnabled(path, dir, "my-plugin", "npm", true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	enabled, found := readTOMLPluginEnabled(t, path, "my-plugin@npm")
	if !found || !enabled {
		t.Errorf("expected enabled=true, found=%v enabled=%v", found, enabled)
	}
}

func TestWriteTOMLPluginEnabled_TogglesExistingPlugin(t *testing.T) {
	dir := t.TempDir()
	path := writeTOML(t, dir, "config.toml", `[plugins."p@m"]
enabled = true
`)
	if err := WriteTOMLPluginEnabled(path, dir, "p", "m", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	enabled, found := readTOMLPluginEnabled(t, path, "p@m")
	if !found || enabled {
		t.Errorf("expected enabled=false, found=%v enabled=%v", found, enabled)
	}
}

func TestWriteTOMLPluginEnabled_PathEscape_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	escapedPath := filepath.Join(dir, "..", "escape.toml")
	err := WriteTOMLPluginEnabled(escapedPath, dir, "p", "m", true)
	if err == nil {
		t.Fatal("expected path escape error, got nil")
	}
	we, ok := err.(*pluginWriteError)
	if !ok || we.Status() != "path_escape" {
		t.Errorf("expected path_escape error, got %v", err)
	}
}

func TestWriteTOMLPluginEnabled_Symlink_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.toml")
	if err := os.WriteFile(real, []byte("[plugins]\n"), 0o644); err != nil {
		t.Fatalf("write real: %v", err)
	}
	link := filepath.Join(dir, "config.toml")
	if err := os.Symlink(real, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	err := WriteTOMLPluginEnabled(link, dir, "p", "m", true)
	if err == nil {
		t.Fatal("expected symlink error, got nil")
	}
	we, ok := err.(*pluginWriteError)
	if !ok || we.Status() != "symlink" {
		t.Errorf("expected symlink error, got %v", err)
	}
}

func TestWriteTOMLPluginEnabled_PreservesCommentsAndFormatting(t *testing.T) {
	dir := t.TempDir()
	input := `# Codex configuration
version = 2

[plugins."my-plugin@npm"]
# plugin description
enabled = true  # active
`
	path := writeTOML(t, dir, "config.toml", input)
	if err := WriteTOMLPluginEnabled(path, dir, "my-plugin", "npm", false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(path)
	text := string(data)
	if !strings.Contains(text, "# Codex configuration") {
		t.Errorf("top comment lost:\n%s", text)
	}
	if !strings.Contains(text, "# plugin description") {
		t.Errorf("plugin comment lost:\n%s", text)
	}
	if !strings.Contains(text, "enabled = false  # active") {
		t.Errorf("inline comment lost or value not updated:\n%s", text)
	}
	if !strings.Contains(text, "version = 2") {
		t.Errorf("unrelated key lost:\n%s", text)
	}
}
