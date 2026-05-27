package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCodexConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestScanCodexConfigFile_Missing(t *testing.T) {
	dir := t.TempDir()
	result := ScanCodexConfigFile(filepath.Join(dir, "config.toml"), dir)
	if result.Status != "missing" {
		t.Fatalf("status: got %q want missing", result.Status)
	}
}

func TestScanCodexConfigFile_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := writeCodexConfig(t, dir, "[plugins.")
	result := ScanCodexConfigFile(path, dir)
	if result.Status != "malformed" {
		t.Fatalf("status: got %q want malformed", result.Status)
	}
}

func TestScanCodexConfigFile_PathEscape(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	path := writeCodexConfig(t, other, "")
	result := ScanCodexConfigFile(path, dir)
	if result.Status != "path_escape" {
		t.Fatalf("status: got %q want path_escape", result.Status)
	}
}

func TestScanCodexConfigFile_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := writeCodexConfig(t, dir, "")
	link := filepath.Join(dir, "link.toml")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	result := ScanCodexConfigFile(link, dir)
	if result.Status != "symlink" {
		t.Fatalf("status: got %q want symlink", result.Status)
	}
}

func TestScanCodexConfigFile_Plugins(t *testing.T) {
	dir := t.TempDir()
	path := writeCodexConfig(t, dir, `
[plugins."github@openai-curated"]
enabled = true

[plugins."stitch-build@stitch-skills"]
enabled = false
`)

	result := ScanCodexConfigFile(path, dir)
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
	if got["github@openai-curated"] != true {
		t.Errorf("github@openai-curated enabled: got false want true")
	}
	if got["stitch-build@stitch-skills"] != false {
		t.Errorf("stitch-build@stitch-skills enabled: got true want false")
	}
}

func TestScanCodexConfigFile_Marketplaces(t *testing.T) {
	dir := t.TempDir()
	path := writeCodexConfig(t, dir, `
[marketplaces.openai-curated]
type = "github"
source = "openai/codex"
`)

	result := ScanCodexConfigFile(path, dir)
	if result.Status != "ok" {
		t.Fatalf("status: got %q want ok", result.Status)
	}
	if len(result.Marketplaces) != 1 {
		t.Fatalf("marketplaces: got %d want 1", len(result.Marketplaces))
	}
	if result.Marketplaces[0].MarketplaceName != "openai-curated" {
		t.Errorf("marketplace name: got %q", result.Marketplaces[0].MarketplaceName)
	}
	if result.Marketplaces[0].SourceType != "github" {
		t.Errorf("source type: got %q want github", result.Marketplaces[0].SourceType)
	}
}

func TestScanCodexConfigFile_MarketplaceUnknownTypeIsNormalized(t *testing.T) {
	dir := t.TempDir()
	path := writeCodexConfig(t, dir, `
[marketplaces.custom]
type = "not-a-real-type-with-user-data"
source = "safe-summary"
`)

	result := ScanCodexConfigFile(path, dir)
	if result.Status != "ok" {
		t.Fatalf("status: got %q want ok", result.Status)
	}
	if len(result.Marketplaces) != 1 {
		t.Fatalf("marketplaces: got %d want 1", len(result.Marketplaces))
	}
	if result.Marketplaces[0].SourceType != "unknown" {
		t.Errorf("source type: got %q want unknown", result.Marketplaces[0].SourceType)
	}
}

func TestScanCodexConfigFile_WarningsContainNoRawKeys(t *testing.T) {
	dir := t.TempDir()
	rawKey := "secret-plugin@"
	path := writeCodexConfig(t, dir, `
[plugins."secret-plugin@"]
enabled = true

[plugins."other@market"]
enabled = "yes"
`)

	result := ScanCodexConfigFile(path, dir)
	if result.Status != "ok" {
		t.Fatalf("status: got %q want ok", result.Status)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("expected warnings")
	}
	for _, w := range result.Warnings {
		if strings.Contains(w, rawKey) || strings.Contains(w, "other@market") {
			t.Fatalf("warning leaked raw key %q: %q", rawKey, w)
		}
	}
}
