package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func writeAntigravitySettings(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write settings: %v", err)
	}
	return path
}

func TestScanAntigravityCLISettingsFile_Missing(t *testing.T) {
	dir := t.TempDir()
	result := ScanAntigravityCLISettingsFile(filepath.Join(dir, "settings.json"), dir)
	if result.Status != "missing" {
		t.Fatalf("status: got %q want missing", result.Status)
	}
}

func TestScanAntigravityCLISettingsFile_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := writeAntigravitySettings(t, dir, "{bad json")
	result := ScanAntigravityCLISettingsFile(path, dir)
	if result.Status != "malformed" {
		t.Fatalf("status: got %q want malformed", result.Status)
	}
}

func TestScanAntigravityCLISettingsFile_PathEscape(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	path := writeAntigravitySettings(t, other, "{}")
	result := ScanAntigravityCLISettingsFile(path, dir)
	if result.Status != "path_escape" {
		t.Fatalf("status: got %q want path_escape", result.Status)
	}
}

func TestScanAntigravityCLISettingsFile_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := writeAntigravitySettings(t, dir, "{}")
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	result := ScanAntigravityCLISettingsFile(link, dir)
	if result.Status != "symlink" {
		t.Fatalf("status: got %q want symlink", result.Status)
	}
}

func TestScanAntigravityCLISettingsFile_Plugins(t *testing.T) {
	dir := t.TempDir()
	path := writeAntigravitySettings(t, dir, `{
		"enabledPlugins": {
			"stitch-build@stitch-skills": true,
			"other-plugin@my-market": false
		}
	}`)
	result := ScanAntigravityCLISettingsFile(path, dir)
	if result.Status != "ok" {
		t.Fatalf("status: got %q want ok", result.Status)
	}
	if len(result.Plugins) != 2 {
		t.Fatalf("plugins: got %d want 2", len(result.Plugins))
	}
	got := map[string]bool{}
	for _, p := range result.Plugins {
		got[p.PluginName+"@"+p.MarketplaceName] = p.Enabled
	}
	if got["stitch-build@stitch-skills"] != true {
		t.Errorf("stitch-build@stitch-skills enabled: got false want true")
	}
	if got["other-plugin@my-market"] != false {
		t.Errorf("other-plugin@my-market enabled: got true want false")
	}
}

func TestScanAntigravityCLISettingsFile_Marketplaces(t *testing.T) {
	dir := t.TempDir()
	path := writeAntigravitySettings(t, dir, `{
		"extraKnownMarketplaces": [
			{"name": "stitch-skills", "type": "github", "githubOrg": "astraler", "githubRepo": "stitch-skills"}
		]
	}`)
	result := ScanAntigravityCLISettingsFile(path, dir)
	if result.Status != "ok" {
		t.Fatalf("status: got %q want ok", result.Status)
	}
	if len(result.Marketplaces) != 1 {
		t.Fatalf("marketplaces: got %d want 1", len(result.Marketplaces))
	}
	if result.Marketplaces[0].MarketplaceName != "stitch-skills" {
		t.Errorf("marketplace name: got %q want stitch-skills", result.Marketplaces[0].MarketplaceName)
	}
	if result.Marketplaces[0].SourceType != "github" {
		t.Errorf("source type: got %q want github", result.Marketplaces[0].SourceType)
	}
}

func TestScanAntigravityCLISettingsFile_EmptyOK(t *testing.T) {
	dir := t.TempDir()
	path := writeAntigravitySettings(t, dir, `{}`)
	result := ScanAntigravityCLISettingsFile(path, dir)
	if result.Status != "ok" {
		t.Fatalf("status: got %q want ok", result.Status)
	}
	if len(result.Plugins) != 0 || len(result.Marketplaces) != 0 {
		t.Errorf("expected empty plugins and marketplaces")
	}
}
