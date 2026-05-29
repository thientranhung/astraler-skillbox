package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func ptr(s string) *string { return &s }

func TestScanClaudeInstalledPluginsFile(t *testing.T) {
	t.Run("ok with user-scope entries", func(t *testing.T) {
		dir := t.TempDir()
		pluginsDir := filepath.Join(dir, "plugins")
		if err := os.MkdirAll(pluginsDir, 0755); err != nil {
			t.Fatal(err)
		}
		content := `{
			"version": 2,
			"plugins": {
				"my-plugin@my-marketplace": [
					{"scope": "user", "version": "1.2.3", "installPath": "/some/path"}
				],
				"other-plugin@official": [
					{"scope": "project", "version": "2.0.0"},
					{"scope": "user", "version": "unknown"}
				]
			}
		}`
		path := filepath.Join(pluginsDir, "installed_plugins.json")
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		got := ScanClaudeInstalledPluginsFile(path, dir)
		if got.Status != "ok" {
			t.Fatalf("status = %q, want ok", got.Status)
		}
		vm := BuildVersionMap(got)
		if v, ok := vm["my-plugin@my-marketplace"]; !ok || v == nil || *v != "1.2.3" {
			t.Errorf("my-plugin@my-marketplace version = %v, want '1.2.3'", v)
		}
		if v, ok := vm["other-plugin@official"]; !ok || v == nil || *v != "unknown" {
			t.Errorf("other-plugin@official version = %v, want 'unknown'", v)
		}
		// project-scope should NOT appear in BuildVersionMap
		if len(vm) != 2 {
			t.Errorf("version map len = %d, want 2", len(vm))
		}
	})

	t.Run("missing file", func(t *testing.T) {
		dir := t.TempDir()
		got := ScanClaudeInstalledPluginsFile(filepath.Join(dir, "plugins", "installed_plugins.json"), dir)
		if got.Status != "missing" {
			t.Errorf("status = %q, want missing", got.Status)
		}
	})

	t.Run("malformed JSON", func(t *testing.T) {
		dir := t.TempDir()
		pluginsDir := filepath.Join(dir, "plugins")
		_ = os.MkdirAll(pluginsDir, 0755)
		path := filepath.Join(pluginsDir, "installed_plugins.json")
		_ = os.WriteFile(path, []byte(`{not valid json`), 0644)
		got := ScanClaudeInstalledPluginsFile(path, dir)
		if got.Status != "malformed" {
			t.Errorf("status = %q, want malformed", got.Status)
		}
	})

	t.Run("too large", func(t *testing.T) {
		dir := t.TempDir()
		pluginsDir := filepath.Join(dir, "plugins")
		_ = os.MkdirAll(pluginsDir, 0755)
		path := filepath.Join(pluginsDir, "installed_plugins.json")
		big := make([]byte, claudeInstalledPluginsMaxFileSize+1)
		for i := range big {
			big[i] = 'x'
		}
		_ = os.WriteFile(path, big, 0644)
		got := ScanClaudeInstalledPluginsFile(path, dir)
		if got.Status != "too_large" {
			t.Errorf("status = %q, want too_large", got.Status)
		}
	})

	t.Run("symlink rejected", func(t *testing.T) {
		dir := t.TempDir()
		pluginsDir := filepath.Join(dir, "plugins")
		_ = os.MkdirAll(pluginsDir, 0755)
		real := filepath.Join(dir, "real.json")
		_ = os.WriteFile(real, []byte(`{}`), 0644)
		link := filepath.Join(pluginsDir, "installed_plugins.json")
		_ = os.Symlink(real, link)
		got := ScanClaudeInstalledPluginsFile(link, dir)
		if got.Status != "symlink" {
			t.Errorf("status = %q, want symlink", got.Status)
		}
	})

	t.Run("path escape rejected", func(t *testing.T) {
		dir := t.TempDir()
		other := t.TempDir()
		path := filepath.Join(other, "installed_plugins.json")
		_ = os.WriteFile(path, []byte(`{}`), 0644)
		got := ScanClaudeInstalledPluginsFile(path, dir)
		if got.Status != "path_escape" {
			t.Errorf("status = %q, want path_escape", got.Status)
		}
	})

	t.Run("missing plugins key = ok with 0 entries", func(t *testing.T) {
		dir := t.TempDir()
		pluginsDir := filepath.Join(dir, "plugins")
		_ = os.MkdirAll(pluginsDir, 0755)
		path := filepath.Join(pluginsDir, "installed_plugins.json")
		_ = os.WriteFile(path, []byte(`{"version": 2}`), 0644)
		got := ScanClaudeInstalledPluginsFile(path, dir)
		if got.Status != "ok" {
			t.Errorf("status = %q, want ok", got.Status)
		}
		if len(got.Entries) != 0 {
			t.Errorf("entries len = %d, want 0", len(got.Entries))
		}
	})

	t.Run("JSON null version treated as nil", func(t *testing.T) {
		dir := t.TempDir()
		pluginsDir := filepath.Join(dir, "plugins")
		_ = os.MkdirAll(pluginsDir, 0755)
		path := filepath.Join(pluginsDir, "installed_plugins.json")
		content := `{"plugins": {"plugin@mkt": [{"scope": "user", "version": null}]}}`
		_ = os.WriteFile(path, []byte(content), 0644)
		got := ScanClaudeInstalledPluginsFile(path, dir)
		if got.Status != "ok" {
			t.Fatalf("status = %q, want ok", got.Status)
		}
		vm := BuildVersionMap(got)
		if v, ok := vm["plugin@mkt"]; !ok {
			t.Error("key missing from version map")
		} else if v != nil {
			t.Errorf("version = %v, want nil for JSON null", v)
		}
	})

	t.Run("version field absent treated as nil", func(t *testing.T) {
		dir := t.TempDir()
		pluginsDir := filepath.Join(dir, "plugins")
		_ = os.MkdirAll(pluginsDir, 0755)
		path := filepath.Join(pluginsDir, "installed_plugins.json")
		content := `{"plugins": {"plugin@mkt": [{"scope": "user", "installPath": "/x"}]}}`
		_ = os.WriteFile(path, []byte(content), 0644)
		got := ScanClaudeInstalledPluginsFile(path, dir)
		vm := BuildVersionMap(got)
		if v := vm["plugin@mkt"]; v != nil {
			t.Errorf("version = %v, want nil when field absent", v)
		}
	})

	t.Run("BuildVersionMap ignores non-user scopes", func(t *testing.T) {
		dir := t.TempDir()
		pluginsDir := filepath.Join(dir, "plugins")
		_ = os.MkdirAll(pluginsDir, 0755)
		path := filepath.Join(pluginsDir, "installed_plugins.json")
		content := `{"plugins": {"p@m": [{"scope": "project", "version": "3.0.0"}]}}`
		_ = os.WriteFile(path, []byte(content), 0644)
		got := ScanClaudeInstalledPluginsFile(path, dir)
		vm := BuildVersionMap(got)
		if _, ok := vm["p@m"]; ok {
			t.Error("project-scope entry should not appear in BuildVersionMap")
		}
	})

	t.Run("BuildVersionMap empty on non-ok status", func(t *testing.T) {
		scan := ClaudeInstalledPluginsScan{Status: "missing"}
		vm := BuildVersionMap(scan)
		if len(vm) != 0 {
			t.Errorf("len = %d, want 0 for non-ok status", len(vm))
		}
	})
}
