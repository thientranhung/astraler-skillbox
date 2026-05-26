package providers

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSettingsFile writes JSON content to <dir>/settings.json and returns its path.
func writeSettingsFile(t *testing.T, dir string, content interface{}) string {
	t.Helper()
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshal settings: %v", err)
	}
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write settings file: %v", err)
	}
	return path
}

func TestScanClaudeSettingsFile_Missing(t *testing.T) {
	dir := t.TempDir()
	result := ScanClaudeSettingsFile(filepath.Join(dir, "settings.json"), dir)
	if result.Status != "missing" {
		t.Errorf("status: got %q want %q", result.Status, "missing")
	}
}

func TestScanClaudeSettingsFile_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("not json {{{"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "malformed" {
		t.Errorf("status: got %q want %q", result.Status, "malformed")
	}
}

func TestScanClaudeSettingsFile_MalformedTopLevelArray(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`[1,2,3]`), 0o644); err != nil {
		t.Fatal(err)
	}
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "malformed" {
		t.Errorf("status: got %q want %q", result.Status, "malformed")
	}
}

func TestScanClaudeSettingsFile_EnabledPluginsNonObject(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"enabledPlugins": ["foo"]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "malformed" {
		t.Errorf("status: got %q want %q", result.Status, "malformed")
	}
}

func TestScanClaudeSettingsFile_TooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	big := make([]byte, int(ClaudeSettingsMaxFileSize)+1)
	big[0] = '{'
	big[len(big)-1] = '}'
	if err := os.WriteFile(path, big, 0o644); err != nil {
		t.Fatal(err)
	}
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "too_large" {
		t.Errorf("status: got %q want %q", result.Status, "too_large")
	}
}

func TestScanClaudeSettingsFile_PathEscape(t *testing.T) {
	dir := t.TempDir()
	otherDir := t.TempDir()
	escapedPath := filepath.Join(otherDir, "settings.json")
	result := ScanClaudeSettingsFile(escapedPath, dir)
	if result.Status != "path_escape" {
		t.Errorf("status: got %q want %q", result.Status, "path_escape")
	}
}

func TestScanClaudeSettingsFile_Symlink(t *testing.T) {
	dir := t.TempDir()
	realFile := filepath.Join(t.TempDir(), "real.json")
	if err := os.WriteFile(realFile, []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(dir, "settings.json")
	if err := os.Symlink(realFile, symlinkPath); err != nil {
		t.Skip("symlink creation not supported")
	}
	result := ScanClaudeSettingsFile(symlinkPath, dir)
	if result.Status != "symlink" {
		t.Errorf("status: got %q want %q", result.Status, "symlink")
	}
}

func TestScanClaudeSettingsFile_EmptyObject(t *testing.T) {
	dir := t.TempDir()
	path := writeSettingsFile(t, dir, map[string]interface{}{})
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Plugins) != 0 {
		t.Errorf("plugins: got %d want 0", len(result.Plugins))
	}
}

func TestScanClaudeSettingsFile_ValidPlugins(t *testing.T) {
	dir := t.TempDir()
	path := writeSettingsFile(t, dir, map[string]interface{}{
		"enabledPlugins": map[string]interface{}{
			"my-plugin@npm":     true,
			"other-plugin@npm":  false,
			"pkg@my-registry":   true,
		},
	})
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Plugins) != 3 {
		t.Errorf("plugins count: got %d want 3", len(result.Plugins))
	}
	// Find enabled plugin
	found := false
	for _, p := range result.Plugins {
		if p.PluginName == "my-plugin" && p.MarketplaceName == "npm" && p.Enabled {
			found = true
		}
		if p.PluginName == "other-plugin" && p.MarketplaceName == "npm" && p.Enabled {
			t.Error("other-plugin@npm should be disabled")
		}
	}
	if !found {
		t.Error("my-plugin@npm enabled not found")
	}
}

func TestScanClaudeSettingsFile_NonBoolValueSkipped(t *testing.T) {
	dir := t.TempDir()
	path := writeSettingsFile(t, dir, map[string]interface{}{
		"enabledPlugins": map[string]interface{}{
			"valid@npm":   true,
			"invalid@npm": "yes", // non-bool
		},
	})
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Plugins) != 1 {
		t.Errorf("plugins: got %d want 1 (non-bool skipped)", len(result.Plugins))
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for non-bool value")
	}
}

func TestScanClaudeSettingsFile_InvalidPluginKeySkipped(t *testing.T) {
	dir := t.TempDir()
	path := writeSettingsFile(t, dir, map[string]interface{}{
		"enabledPlugins": map[string]interface{}{
			"no-at-sign":  true, // missing @
			"@npm":        true, // empty plugin name
			"plugin@":     true, // empty marketplace
			"valid@npm":   true,
		},
	})
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Plugins) != 1 {
		t.Errorf("plugins: got %d want 1 (invalid keys skipped)", len(result.Plugins))
	}
	if len(result.Warnings) < 3 {
		t.Errorf("warnings: got %d want >= 3", len(result.Warnings))
	}
}

func TestScanClaudeSettingsFile_NameLengthCap(t *testing.T) {
	dir := t.TempDir()
	longName := strings.Repeat("a", ClaudeSettingsMaxNameLen+1)
	path := writeSettingsFile(t, dir, map[string]interface{}{
		"enabledPlugins": map[string]interface{}{
			longName + "@npm": true,
		},
	})
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Plugins) != 0 {
		t.Errorf("plugins: got %d want 0 (long name skipped)", len(result.Plugins))
	}
}

func TestScanClaudeSettingsFile_PluginCountCap(t *testing.T) {
	dir := t.TempDir()
	pluginsExact := make(map[string]interface{}, ClaudeSettingsMaxPlugins+5)
	for i := 0; i < ClaudeSettingsMaxPlugins+5; i++ {
		pluginsExact[fmt.Sprintf("plugin-%d@npm", i)] = true
	}
	path := writeSettingsFile(t, dir, map[string]interface{}{
		"enabledPlugins": pluginsExact,
	})
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Plugins) > ClaudeSettingsMaxPlugins {
		t.Errorf("plugins: got %d want <= %d", len(result.Plugins), ClaudeSettingsMaxPlugins)
	}
	hasWarning := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "truncated") {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Error("expected truncation warning")
	}
}

func TestScanClaudeSettingsFile_MarketplaceCap(t *testing.T) {
	dir := t.TempDir()
	mps := make([]interface{}, ClaudeSettingsMaxMarketplaces+5)
	for i := range mps {
		mps[i] = map[string]interface{}{
			"name": fmt.Sprintf("marketplace-%d", i),
			"type": "github",
		}
	}
	path := writeSettingsFile(t, dir, map[string]interface{}{
		"extraKnownMarketplaces": mps,
	})
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Marketplaces) > ClaudeSettingsMaxMarketplaces {
		t.Errorf("marketplaces: got %d want <= %d", len(result.Marketplaces), ClaudeSettingsMaxMarketplaces)
	}
}

func TestScanClaudeSettingsFile_MarketplaceSourceTypes(t *testing.T) {
	dir := t.TempDir()
	path := writeSettingsFile(t, dir, map[string]interface{}{
		"extraKnownMarketplaces": []interface{}{
			map[string]interface{}{"name": "gh-mp", "type": "github", "githubOrg": "anthropics", "githubRepo": "plugins"},
			map[string]interface{}{"name": "git-mp", "type": "git", "url": "https://github.com/example/plugins"},
			map[string]interface{}{"name": "dir-mp", "type": "directory", "path": "/some/path"},
			map[string]interface{}{"name": "unknown-mp", "type": "weirdType"},
		},
	})
	result := ScanClaudeSettingsFile(path, dir)
	if result.Status != "ok" {
		t.Errorf("status: got %q want %q", result.Status, "ok")
	}
	if len(result.Marketplaces) != 4 {
		t.Fatalf("marketplaces: got %d want 4", len(result.Marketplaces))
	}
	mpByName := map[string]ClaudeMarketplaceDecl{}
	for _, m := range result.Marketplaces {
		mpByName[m.MarketplaceName] = m
	}
	if mpByName["gh-mp"].SourceType != "github" {
		t.Errorf("gh-mp sourceType: got %q want github", mpByName["gh-mp"].SourceType)
	}
	if mpByName["gh-mp"].SourceSummary != "anthropics/plugins" {
		t.Errorf("gh-mp summary: got %q want anthropics/plugins", mpByName["gh-mp"].SourceSummary)
	}
	if mpByName["unknown-mp"].SourceType != "unknown" {
		t.Errorf("unknown-mp sourceType: got %q want unknown", mpByName["unknown-mp"].SourceType)
	}
}

func TestParseClaudePluginKey(t *testing.T) {
	cases := []struct {
		key      string
		wantName string
		wantMkt  string
		wantOK   bool
	}{
		{"plugin@npm", "plugin", "npm", true},
		{"my-plugin@my-registry", "my-plugin", "my-registry", true},
		{"a@b@c", "a@b", "c", true},   // last @ used
		{"nope", "", "", false},         // no @
		{"@npm", "", "", false},          // empty plugin name
		{"plugin@", "", "", false},       // empty marketplace
		{"@", "", "", false},             // both empty
	}
	for _, tc := range cases {
		p, m, ok := parseClaudePluginKey(tc.key)
		if ok != tc.wantOK {
			t.Errorf("key %q: ok got %v want %v", tc.key, ok, tc.wantOK)
			continue
		}
		if ok && (p != tc.wantName || m != tc.wantMkt) {
			t.Errorf("key %q: got %q@%q want %q@%q", tc.key, p, m, tc.wantName, tc.wantMkt)
		}
	}
}
