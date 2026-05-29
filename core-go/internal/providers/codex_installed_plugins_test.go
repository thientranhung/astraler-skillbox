package providers_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/providers"
)

// buildCacheDir builds a fake ~/.codex/plugins/cache layout under root.
// structure: map[marketplace]map[pluginName]struct{ versionDir, pluginJSON string }
// pluginJSON == "" means no plugin.json is written.
func buildCacheDir(t *testing.T, root string, layout map[string]map[string]struct {
	versionDir string
	pluginJSON string
}) {
	t.Helper()
	for marketplace, plugins := range layout {
		for plugin, cfg := range plugins {
			dirPath := filepath.Join(root, marketplace, plugin, cfg.versionDir)
			if err := os.MkdirAll(dirPath, 0o755); err != nil {
				t.Fatal(err)
			}
			if cfg.pluginJSON != "" {
				if err := os.WriteFile(filepath.Join(dirPath, "plugin.json"), []byte(cfg.pluginJSON), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}

func TestScanCodexCacheDir_SemverPlugin(t *testing.T) {
	root := t.TempDir()
	buildCacheDir(t, root, map[string]map[string]struct {
		versionDir string
		pluginJSON string
	}{
		"stitch-skills": {
			"stitch-design": {
				versionDir: "1.0.0",
				pluginJSON: `{"name":"stitch-design","version":"1.0.0"}`,
			},
		},
	})

	scan := providers.ScanCodexCacheDir(root, root)
	if scan.Status != "ok" {
		t.Fatalf("expected ok, got %q", scan.Status)
	}
	if len(scan.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(scan.Entries))
	}
	e := scan.Entries[0]
	if e.PluginKey != "stitch-design@stitch-skills" {
		t.Errorf("unexpected key %q", e.PluginKey)
	}
	if e.Version == nil || *e.Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %v", e.Version)
	}
}

func TestScanCodexCacheDir_SHAPlugin(t *testing.T) {
	root := t.TempDir()
	buildCacheDir(t, root, map[string]map[string]struct {
		versionDir string
		pluginJSON string
	}{
		"openai-curated": {
			"superpowers": {
				versionDir: "9b3c8689",
				pluginJSON: "", // no plugin.json
			},
		},
	})

	scan := providers.ScanCodexCacheDir(root, root)
	if scan.Status != "ok" {
		t.Fatalf("expected ok, got %q", scan.Status)
	}
	if len(scan.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(scan.Entries))
	}
	e := scan.Entries[0]
	if e.Version == nil || *e.Version != "9b3c8689" {
		t.Errorf("expected SHA version 9b3c8689, got %v", e.Version)
	}
}

func TestScanCodexCacheDir_Missing(t *testing.T) {
	root := t.TempDir()
	// Pass a non-existent subdir as cacheDir.
	scan := providers.ScanCodexCacheDir(filepath.Join(root, "plugins", "cache"), root)
	if scan.Status != "missing" {
		t.Errorf("expected missing, got %q", scan.Status)
	}
	// BuildCodexVersionMap on non-ok scan returns empty map, not nil.
	m := providers.BuildCodexVersionMap(scan)
	if len(m) != 0 {
		t.Errorf("expected empty map on missing scan")
	}
}

func TestScanCodexCacheDir_MalformedPluginJSON(t *testing.T) {
	root := t.TempDir()
	buildCacheDir(t, root, map[string]map[string]struct {
		versionDir string
		pluginJSON string
	}{
		"stitch-skills": {
			"stitch-build": {
				versionDir: "1.0.0",
				pluginJSON: `not-valid-json{{{`,
			},
		},
	})

	scan := providers.ScanCodexCacheDir(root, root)
	if scan.Status != "ok" {
		t.Fatalf("expected ok, got %q", scan.Status)
	}
	if len(scan.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(scan.Entries))
	}
	e := scan.Entries[0]
	// Falls back to dir name when plugin.json is malformed.
	if e.Version == nil || *e.Version != "1.0.0" {
		t.Errorf("expected fallback to dir name 1.0.0, got %v", e.Version)
	}
}

func TestScanCodexCacheDir_SymlinkReject(t *testing.T) {
	root := t.TempDir()
	// Create a real plugin and a symlink plugin dir.
	buildCacheDir(t, root, map[string]map[string]struct {
		versionDir string
		pluginJSON string
	}{
		"openai-curated": {
			"github": {
				versionDir: "9b3c8689",
				pluginJSON: "",
			},
		},
	})
	// Create a symlink in place of a plugin dir.
	symlinkTarget := t.TempDir()
	symlinkPlugin := filepath.Join(root, "openai-curated", "evil-symlink")
	if err := os.Symlink(symlinkTarget, symlinkPlugin); err != nil {
		t.Skip("cannot create symlink:", err)
	}

	scan := providers.ScanCodexCacheDir(root, root)
	if scan.Status != "ok" {
		t.Fatalf("expected ok, got %q", scan.Status)
	}
	// Symlink plugin must be excluded; only the real plugin is returned.
	for _, e := range scan.Entries {
		if e.PluginKey == "evil-symlink@openai-curated" {
			t.Error("symlink plugin must be excluded from entries")
		}
	}
	// Check a warning was recorded.
	found := false
	for _, w := range scan.Warnings {
		if w != "" {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one warning for the rejected symlink")
	}
}

func TestScanCodexCacheDir_PathEscape(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	scan := providers.ScanCodexCacheDir(other, root)
	if scan.Status != "path_escape" {
		t.Errorf("expected path_escape, got %q", scan.Status)
	}
}

func TestBuildCodexVersionMap_MultiplePlugins(t *testing.T) {
	root := t.TempDir()
	buildCacheDir(t, root, map[string]map[string]struct {
		versionDir string
		pluginJSON string
	}{
		"stitch-skills": {
			"stitch-design": {
				versionDir: "1.0.0",
				pluginJSON: `{"version":"1.0.0"}`,
			},
			"stitch-build": {
				versionDir: "1.0.0",
				pluginJSON: `{"version":"1.0.0"}`,
			},
		},
		"openai-curated": {
			"github": {
				versionDir: "9b3c8689",
				pluginJSON: "",
			},
		},
	})

	scan := providers.ScanCodexCacheDir(root, root)
	m := providers.BuildCodexVersionMap(scan)

	wantKeys := []struct {
		key     string
		version string
	}{
		{"stitch-design@stitch-skills", "1.0.0"},
		{"stitch-build@stitch-skills", "1.0.0"},
		{"github@openai-curated", "9b3c8689"},
	}
	for _, w := range wantKeys {
		v, ok := m[w.key]
		if !ok {
			t.Errorf("missing key %q in version map", w.key)
			continue
		}
		if v == nil || *v != w.version {
			t.Errorf("key %q: expected %q, got %v", w.key, w.version, v)
		}
	}
}
