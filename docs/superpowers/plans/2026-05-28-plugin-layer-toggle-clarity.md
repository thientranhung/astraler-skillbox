# Plugin Layer Toggle Clarity — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-layer plugin toggles in the project detail screen so users can see and control project vs user layer independently, add a `providerPlugin.removeOverride` command for clearing project-layer overrides, and clarify button labels on the global plugins screen.

**Architecture:** The backend adds a `RemoveOverride` service method that removes a plugin key from a project-layer settings file (JSON or TOML), mirroring `SetPluginEnabled` but deleting instead of toggling. The frontend redesigns the project detail plugin table to show separate Project (3-state cycle) and User (2-state toggle) columns derived from existing `layerBreakdown` data. No new DB tables or changes to scan/resolution logic.

**Tech Stack:** Go (service + RPC handler), JSON Schema contract, TypeScript React (TanStack Query hook, table redesign), Tailwind CSS.

---

## File Structure

### Backend (Go)
- **Create:** `core-go/internal/providers/json_plugin_remover.go` — `RemoveJSONPlugin` function: removes a plugin key from `enabledPlugins` in a JSON settings file
- **Create:** `core-go/internal/providers/json_plugin_remover_test.go` — tests for `RemoveJSONPlugin`
- **Create:** `core-go/internal/providers/toml_plugin_remover.go` — `RemoveTOMLPlugin` function: removes a plugin section/key from a TOML settings file
- **Create:** `core-go/internal/providers/toml_plugin_remover_test.go` — tests for `RemoveTOMLPlugin`
- **Modify:** `core-go/internal/services/provider_plugin_service.go` — add `pluginRemoverFn` type, `removerFor` method, `RemoveOverride` method, `removeOverrideProjectInternal` method
- **Create:** `core-go/internal/rpc/handlers/provider_plugin_remove_override.go` — new RPC handler
- **Modify:** `core-go/internal/app/wire.go` — register new handler

### Contract
- **Create:** `shared/api-contracts/methods/providerPlugin.removeOverride.json` — new contract
- **Modify:** `shared/api-contracts/index.json` — add codegen entry

### Frontend
- **Create:** `apps/desktop/renderer/src/features/provider-plugins/use-remove-provider-plugin-override.ts` — new TanStack Query mutation hook
- **Modify:** `apps/desktop/renderer/src/lib/core-client/methods.ts` — add `removeProviderPluginOverride` binding
- **Modify:** `apps/desktop/electron/main/core-process/method-allowlist.ts` — add `providerPlugin.removeOverride`
- **Modify:** `apps/desktop/renderer/src/screens/project-detail-screen.tsx` — redesign `ProjectPluginSection` table
- **Modify:** `apps/desktop/renderer/src/screens/plugins-screen.tsx` — change button labels

---

## Task 1: JSON Plugin Remover

**Files:**
- Create: `core-go/internal/providers/json_plugin_remover.go`
- Test: `core-go/internal/providers/json_plugin_remover_test.go`

- [ ] **Step 1: Write the failing test for RemoveJSONPlugin — creates file when missing**

```go
// core-go/internal/providers/json_plugin_remover_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd core-go && go test ./internal/providers/ -run TestRemoveJSONPlugin -v`
Expected: FAIL — `RemoveJSONPlugin` undefined

- [ ] **Step 3: Implement RemoveJSONPlugin**

```go
// core-go/internal/providers/json_plugin_remover.go
package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

// RemoveJSONPlugin removes the plugin key from enabledPlugins in a Claude/Antigravity-
// style settings.json file. If the file does not exist or the key is absent, it is a
// no-op. Applies the same safety preflight as WriteJSONPluginEnabled.
func RemoveJSONPlugin(filePath, allowedDir, pluginName, marketplaceName string) error {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return &pluginWriteError{"path_escape", "settings file path escapes allowed directory"}
	}

	// If file does not exist, nothing to remove.
	fi, err := os.Lstat(cleanFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return &pluginWriteError{"unreadable", "could not stat settings file"}
	}

	// Symlink checks.
	parentDir := filepath.Dir(cleanFile)
	if lfi, err := os.Lstat(parentDir); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		return &pluginWriteError{"symlink", "parent directory is a symlink"}
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return &pluginWriteError{"symlink", "settings file is a symlink"}
	}
	if fi.Size() > ClaudeSettingsMaxFileSize {
		return &pluginWriteError{"too_large", "settings file exceeds 1 MiB"}
	}

	existingData, err := os.ReadFile(cleanFile)
	if err != nil {
		return &pluginWriteError{"unreadable", "could not read settings file"}
	}

	updated, err := applyJSONPluginRemove(existingData, pluginName, marketplaceName)
	if err != nil {
		return err
	}
	if updated == nil {
		return nil // key was absent, nothing to write
	}

	if err := filesystem.WriteFileAtomic(cleanFile, updated, 0o644); err != nil {
		return &pluginWriteError{"unwritable", "could not write settings file"}
	}
	return nil
}

// applyJSONPluginRemove removes the plugin key from enabledPlugins and returns the
// updated JSON bytes. Returns nil, nil if the key is absent (no write needed).
func applyJSONPluginRemove(existing []byte, pluginName, marketplaceName string) ([]byte, error) {
	pluginKey := pluginName + "@" + marketplaceName

	top := make(map[string]json.RawMessage)
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &top); err != nil {
			return nil, &pluginWriteError{"malformed", "settings file is not valid JSON"}
		}
	}

	raw, ok := top["enabledPlugins"]
	if !ok {
		return nil, nil // no enabledPlugins section at all
	}

	plugins := make(map[string]bool)
	if err := json.Unmarshal(raw, &plugins); err != nil {
		return nil, &pluginWriteError{"malformed", "enabledPlugins is not a JSON object"}
	}

	if _, exists := plugins[pluginKey]; !exists {
		return nil, nil // key already absent
	}

	delete(plugins, pluginKey)

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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/providers/ -run TestRemoveJSONPlugin -v`
Expected: PASS (all 5 tests)

- [ ] **Step 5: Commit**

```bash
git add core-go/internal/providers/json_plugin_remover.go core-go/internal/providers/json_plugin_remover_test.go
git commit -m "feat: add RemoveJSONPlugin for removing plugin keys from JSON settings"
```

---

## Task 2: TOML Plugin Remover

**Files:**
- Create: `core-go/internal/providers/toml_plugin_remover.go`
- Test: `core-go/internal/providers/toml_plugin_remover_test.go`

- [ ] **Step 1: Write the failing test for RemoveTOMLPlugin**

```go
// core-go/internal/providers/toml_plugin_remover_test.go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd core-go && go test ./internal/providers/ -run TestRemoveTOMLPlugin -v`
Expected: FAIL — `RemoveTOMLPlugin` undefined

- [ ] **Step 3: Implement RemoveTOMLPlugin**

```go
// core-go/internal/providers/toml_plugin_remover.go
package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/astraler/skillbox/core-go/internal/filesystem"
)

// RemoveTOMLPlugin removes a plugin entry from a Codex-style config.toml file.
// It handles both dotted keys (`"key".enabled = true`) and table sections
// (`[plugins."key"]`). If the file does not exist or the key is absent, it is a no-op.
func RemoveTOMLPlugin(filePath, allowedDir, pluginName, marketplaceName string) error {
	cleanFile := filepath.Clean(filePath)
	cleanDir := filepath.Clean(allowedDir)

	if cleanFile != cleanDir && !strings.HasPrefix(cleanFile, cleanDir+string(os.PathSeparator)) {
		return &pluginWriteError{"path_escape", "settings file path escapes allowed directory"}
	}

	fi, err := os.Lstat(cleanFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return &pluginWriteError{"unreadable", "could not stat settings file"}
	}

	parentDir := filepath.Dir(cleanFile)
	if lfi, err := os.Lstat(parentDir); err == nil && lfi.Mode()&os.ModeSymlink != 0 {
		return &pluginWriteError{"symlink", "parent directory is a symlink"}
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return &pluginWriteError{"symlink", "settings file is a symlink"}
	}
	if fi.Size() > ClaudeSettingsMaxFileSize {
		return &pluginWriteError{"too_large", "settings file exceeds 1 MiB"}
	}

	existingData, err := os.ReadFile(cleanFile)
	if err != nil {
		return &pluginWriteError{"unreadable", "could not read settings file"}
	}

	updated, changed := applyTOMLPluginRemove(string(existingData), pluginName, marketplaceName)
	if !changed {
		return nil
	}

	if err := filesystem.WriteFileAtomic(cleanFile, []byte(updated), 0o644); err != nil {
		return &pluginWriteError{"unwritable", "could not write settings file"}
	}
	return nil
}

// applyTOMLPluginRemove removes lines related to a plugin key from TOML content.
// Returns the updated content and whether any change was made.
func applyTOMLPluginRemove(content, pluginName, marketplaceName string) (string, bool) {
	pluginKey := pluginName + "@" + marketplaceName
	quotedKey := fmt.Sprintf(`"%s"`, pluginKey)

	lines := strings.Split(content, "\n")
	var result []string
	changed := false

	// Build regexes for this plugin key.
	// Match [plugins."key"] table header.
	tableSectionRe := regexp.MustCompile(
		fmt.Sprintf(`^\s*\[plugins\.%s\]\s*(?:#.*)?$`, regexp.QuoteMeta(quotedKey)),
	)
	// Match "key".enabled = ... or "key" = { ... } dotted/inline forms.
	dottedKeyRe := regexp.MustCompile(
		fmt.Sprintf(`^\s*%s\s*[.=]`, regexp.QuoteMeta(quotedKey)),
	)

	i := 0
	for i < len(lines) {
		line := lines[i]

		// Case 1: [plugins."key"] table section — remove until next header.
		if tableSectionRe.MatchString(line) {
			changed = true
			i++ // skip the header
			// Skip all lines until the next header or EOF.
			for i < len(lines) {
				if tomlAnyHeaderRe.MatchString(lines[i]) {
					break
				}
				i++
			}
			// Remove trailing blank lines before next section.
			for len(result) > 0 && strings.TrimSpace(result[len(result)-1]) == "" {
				result = result[:len(result)-1]
			}
			continue
		}

		// Case 2: dotted key under [plugins] — "key".enabled = ... or "key" = { ... }
		if dottedKeyRe.MatchString(line) {
			changed = true
			i++
			continue
		}

		result = append(result, line)
		i++
	}

	if !changed {
		return content, false
	}

	return strings.Join(result, "\n"), true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/providers/ -run TestRemoveTOMLPlugin -v`
Expected: PASS (all 5 tests)

- [ ] **Step 5: Commit**

```bash
git add core-go/internal/providers/toml_plugin_remover.go core-go/internal/providers/toml_plugin_remover_test.go
git commit -m "feat: add RemoveTOMLPlugin for removing plugin keys from TOML settings"
```

---

## Task 3: RemoveOverride Service Method

**Files:**
- Modify: `core-go/internal/services/provider_plugin_service.go`
- Test: `core-go/internal/services/provider_plugin_service_test.go`

The service method follows the same pattern as `SetPluginEnabled` but uses the remover functions instead of writers.

- [ ] **Step 1: Add `pluginRemoverFn` type and `removerFor` method**

Add after the existing `pluginWriterFn` type (around line 31) in `provider_plugin_service.go`:

```go
// pluginRemoverFn is the signature for plugin file removers (JSON and TOML).
type pluginRemoverFn func(filePath, allowedDir, pluginName, marketplaceName string) error
```

Add a `pluginRemover` and `tomlRemover` field to `ProviderPluginService` struct (after `tomlWriter` on line 42):

```go
	pluginRemover pluginRemoverFn
	tomlRemover   pluginRemoverFn
```

In `NewProviderPluginService`, add after the `tomlWriter` init (around line 59):

```go
		pluginRemover: providers.RemoveJSONPlugin,
		tomlRemover:   providers.RemoveTOMLPlugin,
```

Add the `removerFor` method after `writerFor` (after line 68):

```go
// removerFor returns the appropriate file remover for the given provider.
func (s *ProviderPluginService) removerFor(providerKey string) pluginRemoverFn {
	if providerKey == "codex" {
		return s.tomlRemover
	}
	return s.pluginRemover
}
```

- [ ] **Step 2: Run existing tests to verify nothing broke**

Run: `cd core-go && go test ./internal/services/ -v`
Expected: PASS — all existing tests still pass

- [ ] **Step 3: Add RemoveOverride method and removeOverrideProjectInternal**

Add after `SetPluginEnabled` method (after line 347) in `provider_plugin_service.go`:

```go
// RemoveOverride removes a plugin's declaration from a project-layer settings file,
// returning it to "not set" so the user layer takes effect. Only layer="project" is
// supported.
func (s *ProviderPluginService) RemoveOverride(
	ctx context.Context,
	providerKey, pluginName, marketplaceName, layer string,
	projectID int64,
) (int64, error) {
	// Validate provider.
	switch providerKey {
	case "claude", "antigravity_cli", "codex":
		// OK
	default:
		return 0, domain.NewValidationError(
			"Unknown provider",
			fmt.Sprintf("providerKey %q does not support plugin writes", providerKey),
		)
	}

	// Only project layer is supported for removeOverride.
	if layer != "project" {
		return 0, domain.NewValidationError(
			"Only project layer is supported",
			fmt.Sprintf("layer %q is not supported for removeOverride; only project layer is allowed", layer),
		)
	}

	if pluginName == "" || marketplaceName == "" {
		return 0, domain.NewValidationError("Plugin name and marketplace are required", "pluginName and marketplaceName must be non-empty")
	}

	defs, err := s.pluginProviderDefsAllowMissing(ctx)
	if err != nil {
		return 0, err
	}
	var targetDef *pluginProviderDef
	for i := range defs {
		if defs[i].Provider.Key == providerKey {
			targetDef = &defs[i]
			break
		}
	}
	if targetDef == nil {
		return 0, domain.NewValidationError(
			"Provider not configured",
			fmt.Sprintf("provider %q not found in database", providerKey),
		)
	}

	def := *targetDef

	project, err := s.projRepo.GetByID(ctx, projectID)
	if err != nil {
		return 0, domain.NewDatabaseError("Could not fetch project", err.Error())
	}
	if project == nil {
		return 0, domain.NewValidationError("Project not found", fmt.Sprintf("projectId %d does not exist", projectID))
	}

	filePath := def.ProjectFilePath(project.Path)
	if !confinedPath(project.Path, filePath) {
		return 0, domain.NewValidationError(
			"Path confinement violation",
			fmt.Sprintf("resolved path %q is outside project directory %q", filepath.Clean(filePath), filepath.Clean(project.Path)),
		)
	}

	remover := s.removerFor(providerKey)
	target := operations.Target{Type: "provider_plugin_project", ID: projectID}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeScan,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return nil, s.removeOverrideProjectInternal(opCtx, def, project, pluginName, marketplaceName, remover, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not start plugin remove operation", err.Error())
	}
	return opID, nil
}

func (s *ProviderPluginService) removeOverrideProjectInternal(
	ctx context.Context,
	def pluginProviderDef,
	project *domain.Project,
	pluginName, marketplaceName string,
	remover pluginRemoverFn,
	progress operations.ProgressFn,
) error {
	filePath := def.ProjectFilePath(project.Path)
	if !confinedPath(project.Path, filePath) {
		return domain.NewValidationError(
			"Path confinement violation",
			fmt.Sprintf("resolved path %q is outside project directory %q", filepath.Clean(filePath), filepath.Clean(project.Path)),
		)
	}
	allowedDir := def.ProjectAllowedDir(project.Path)

	progress("removing_plugin_override", 0, 1, "")
	if err := remover(filePath, allowedDir, pluginName, marketplaceName); err != nil {
		return domain.NewFilesystemError("Could not remove plugin override", err.Error())
	}
	progress("removing_plugin_override", 1, 1, def.Provider.Key)

	return s.scanProjectInternal(ctx, project, []pluginProviderDef{def}, progress)
}
```

- [ ] **Step 4: Run existing tests to verify nothing broke**

Run: `cd core-go && go test ./internal/services/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add core-go/internal/services/provider_plugin_service.go
git commit -m "feat: add RemoveOverride service method for clearing project-layer plugin overrides"
```

---

## Task 4: RemoveOverride RPC Handler

**Files:**
- Create: `core-go/internal/rpc/handlers/provider_plugin_remove_override.go`
- Modify: `core-go/internal/app/wire.go`

- [ ] **Step 1: Write the failing test**

Add to `core-go/internal/rpc/handlers/provider_plugin_handler_test.go`:

```go
// ---- stub for removeOverride ----

type stubPluginRemoveOverrideSvc struct {
	opID      int64
	err       error
	lastCall  removeOverrideCall
}

type removeOverrideCall struct {
	providerKey     string
	pluginName      string
	marketplaceName string
	layer           string
	projectID       int64
}

func (s *stubPluginRemoveOverrideSvc) RemoveOverride(_ context.Context, providerKey, pluginName, marketplaceName, layer string, projectID int64) (int64, error) {
	s.lastCall = removeOverrideCall{providerKey, pluginName, marketplaceName, layer, projectID}
	return s.opID, s.err
}

// ---- removeOverride tests ----

func TestProviderPluginRemoveOverrideHandler_Success(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{opID: 99}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	var resp struct {
		OperationID int64 `json:"operationId"`
	}
	params := map[string]interface{}{
		"providerKey":     "claude",
		"pluginName":      "test-plugin",
		"marketplaceName": "npm",
		"layer":           "project",
		"projectId":       5,
	}
	if err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, &resp); err != nil {
		t.Fatalf("removeOverride: %v", err)
	}
	if resp.OperationID != 99 {
		t.Errorf("operationId: got %d want 99", resp.OperationID)
	}
	if svc.lastCall.providerKey != "claude" {
		t.Errorf("providerKey: got %q want claude", svc.lastCall.providerKey)
	}
}

func TestProviderPluginRemoveOverrideHandler_MissingProviderKey(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	params := map[string]interface{}{
		"pluginName":      "test",
		"marketplaceName": "npm",
		"layer":           "project",
		"projectId":       5,
	}
	err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, nil)
	if err == nil {
		t.Fatal("expected error for missing providerKey")
	}
}

func TestProviderPluginRemoveOverrideHandler_InvalidLayer(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	params := map[string]interface{}{
		"providerKey":     "claude",
		"pluginName":      "test",
		"marketplaceName": "npm",
		"layer":           "user",
		"projectId":       5,
	}
	err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, nil)
	if err == nil {
		t.Fatal("expected error for non-project layer")
	}
}

func TestProviderPluginRemoveOverrideHandler_MissingProjectId(t *testing.T) {
	svc := &stubPluginRemoveOverrideSvc{}
	cli := startServer(t, handler.Map{"providerPlugin.removeOverride": handlers.NewProviderPluginRemoveOverrideHandler(svc)})

	params := map[string]interface{}{
		"providerKey":     "claude",
		"pluginName":      "test",
		"marketplaceName": "npm",
		"layer":           "project",
	}
	err := cli.CallResult(context.Background(), "providerPlugin.removeOverride", params, nil)
	if err == nil {
		t.Fatal("expected error for missing projectId")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd core-go && go test ./internal/rpc/handlers/ -run TestProviderPluginRemoveOverride -v`
Expected: FAIL — `NewProviderPluginRemoveOverrideHandler` undefined

- [ ] **Step 3: Implement the handler**

```go
// core-go/internal/rpc/handlers/provider_plugin_remove_override.go
package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type providerPluginRemoveOverrideSvc interface {
	RemoveOverride(ctx context.Context, providerKey, pluginName, marketplaceName, layer string, projectID int64) (int64, error)
}

type providerPluginRemoveOverrideRequest struct {
	ProviderKey     string `json:"providerKey"`
	PluginName      string `json:"pluginName"`
	MarketplaceName string `json:"marketplaceName"`
	Layer           string `json:"layer"`
	ProjectID       int64  `json:"projectId"`
}

type providerPluginRemoveOverrideResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewProviderPluginRemoveOverrideHandler(svc providerPluginRemoveOverrideSvc) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p providerPluginRemoveOverrideRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}
		if p.ProviderKey == "" {
			return nil, wrapError(domain.NewValidationError("providerKey is required", "providerKey field is empty"))
		}
		if p.PluginName == "" {
			return nil, wrapError(domain.NewValidationError("pluginName is required", "pluginName field is empty"))
		}
		if p.MarketplaceName == "" {
			return nil, wrapError(domain.NewValidationError("marketplaceName is required", "marketplaceName field is empty"))
		}
		if p.Layer != "project" {
			return nil, wrapError(domain.NewValidationError(
				"Only project layer is supported",
				"layer must be project for removeOverride",
			))
		}
		if p.ProjectID == 0 {
			return nil, wrapError(domain.NewValidationError(
				"projectId is required",
				"projectId must be non-zero",
			))
		}

		opID, err := svc.RemoveOverride(ctx, p.ProviderKey, p.PluginName, p.MarketplaceName, p.Layer, p.ProjectID)
		if err != nil {
			return nil, wrapError(err)
		}
		return providerPluginRemoveOverrideResponse{OperationID: opID}, nil
	})
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/rpc/handlers/ -run TestProviderPluginRemoveOverride -v`
Expected: PASS (all 4 tests)

- [ ] **Step 5: Register handler in wire.go**

In `core-go/internal/app/wire.go`, add after the `providerPlugin.setEnabled` line (line 56):

```go
			"providerPlugin.removeOverride": rpchandlers.NewProviderPluginRemoveOverrideHandler(providerPluginSvc),
```

- [ ] **Step 6: Run all Go tests**

Run: `cd core-go && go test ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add core-go/internal/rpc/handlers/provider_plugin_remove_override.go core-go/internal/rpc/handlers/provider_plugin_handler_test.go core-go/internal/app/wire.go
git commit -m "feat: add providerPlugin.removeOverride RPC handler and wire registration"
```

---

## Task 5: Contract + Codegen + Frontend Wiring

**Files:**
- Create: `shared/api-contracts/methods/providerPlugin.removeOverride.json`
- Modify: `shared/api-contracts/index.json`
- Modify: `apps/desktop/electron/main/core-process/method-allowlist.ts`
- Modify: `apps/desktop/renderer/src/lib/core-client/methods.ts`

- [ ] **Step 1: Create the contract JSON**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ProviderPluginRemoveOverrideMethod",
  "description": "Contract for providerPlugin.removeOverride JSON-RPC method. Removes a plugin's declaration from the project-layer settings file, returning it to 'not set' so the user layer takes effect. Only layer=project is supported.",
  "oneOf": [
    { "$ref": "#/definitions/ProviderPluginRemoveOverrideRequest" },
    { "$ref": "#/definitions/ProviderPluginRemoveOverrideResponse" }
  ],
  "definitions": {
    "ProviderPluginRemoveOverrideRequest": {
      "title": "ProviderPluginRemoveOverrideRequest",
      "description": "Params for providerPlugin.removeOverride.",
      "type": "object",
      "properties": {
        "providerKey": {
          "type": "string",
          "description": "Provider key (e.g. claude, antigravity_cli, codex)."
        },
        "pluginName": {
          "type": "string",
          "description": "Plugin name (the part before @ in name@marketplace)"
        },
        "marketplaceName": {
          "type": "string",
          "description": "Marketplace name (the part after @ in name@marketplace)"
        },
        "layer": {
          "type": "string",
          "enum": ["project"],
          "description": "Settings layer. Only project is supported for removeOverride."
        },
        "projectId": {
          "type": "integer",
          "description": "The project ID whose settings file will be modified."
        }
      },
      "required": ["providerKey", "pluginName", "marketplaceName", "layer", "projectId"],
      "additionalProperties": false
    },
    "ProviderPluginRemoveOverrideResponse": {
      "title": "ProviderPluginRemoveOverrideResponse",
      "description": "Immediate response — the remove and rescan run asynchronously. Errors: validation_error (1001) for unknown/unsupported provider or layer; filesystem_error (1002) if the settings file cannot be written; conflict_error (1005) if a scan or write is already running.",
      "type": "object",
      "properties": {
        "operationId": {
          "type": "integer",
          "description": "ID of the created remove+rescan operation"
        }
      },
      "required": ["operationId"],
      "additionalProperties": false
    }
  }
}
```

Write to: `shared/api-contracts/methods/providerPlugin.removeOverride.json`

- [ ] **Step 2: Add codegen entry to index.json**

Add to the end of the `schemas` array in `shared/api-contracts/index.json`:

```json
    { "input": "methods/providerPlugin.removeOverride.json", "output": "methods/provider-plugin-remove-override.ts" }
```

- [ ] **Step 3: Run codegen**

Run: `cd apps/desktop && pnpm generate:contracts`
Expected: generates `shared/generated/methods/provider-plugin-remove-override.ts` and updates `shared/generated/index.ts`

- [ ] **Step 4: Run contract drift check**

Run: `cd apps/desktop && pnpm check:contracts-drift`
Expected: PASS (no drift)

- [ ] **Step 5: Add to method allowlist**

In `apps/desktop/electron/main/core-process/method-allowlist.ts`, add `"providerPlugin.removeOverride"` after `"providerPlugin.setEnabled"`:

```typescript
  "providerPlugin.removeOverride",
```

- [ ] **Step 6: Add to methods.ts**

In `apps/desktop/renderer/src/lib/core-client/methods.ts`, add the import for the new types:

```typescript
  ProviderPluginRemoveOverrideRequest,
  ProviderPluginRemoveOverrideResponse,
```

Add the method binding after `setProviderPluginEnabled`:

```typescript
  removeProviderPluginOverride: (req: ProviderPluginRemoveOverrideRequest) =>
    invoke<ProviderPluginRemoveOverrideResponse>("providerPlugin.removeOverride", req),
```

- [ ] **Step 7: Commit**

```bash
git add shared/api-contracts/methods/providerPlugin.removeOverride.json shared/api-contracts/index.json shared/generated/ apps/desktop/electron/main/core-process/method-allowlist.ts apps/desktop/renderer/src/lib/core-client/methods.ts
git commit -m "feat: add providerPlugin.removeOverride contract, codegen, and frontend wiring"
```

---

## Task 6: useRemoveProviderPluginOverride Hook

**Files:**
- Create: `apps/desktop/renderer/src/features/provider-plugins/use-remove-provider-plugin-override.ts`

- [ ] **Step 1: Create the hook**

This hook follows the exact same pattern as `useSetProviderPluginEnabled` but calls `removeProviderPluginOverride` instead.

```typescript
// apps/desktop/renderer/src/features/provider-plugins/use-remove-provider-plugin-override.ts
import { useState, useRef, useCallback, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../lib/core-client/progress.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { OperationProgressNotification, ProviderPluginRemoveOverrideRequest } from "@contracts/index.js";

function isTerminal(status: OperationProgressNotification["status"]): boolean {
  return status === "success" || status === "failed" || status === "cancelled";
}

export function useRemoveProviderPluginOverride() {
  const queryClient = useQueryClient();
  const [operationId, setOperationId] = useState<number | null>(null);
  const unsubRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    return () => {
      unsubRef.current?.();
      unsubRef.current = null;
    };
  }, []);

  const mutation = useMutation({
    mutationFn: async (req: ProviderPluginRemoveOverrideRequest) => {
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.removeProviderPluginOverride(req);
        return { operationId: result.operationId, buffered, req };
      } finally {
        tempUnsub();
      }
    },

    onSuccess: ({ operationId: opId, buffered, req }) => {
      const terminalInBuffer = [...buffered]
        .reverse()
        .find((e) => e.operationId === opId && isTerminal(e.status));

      if (terminalInBuffer != null) {
        if (terminalInBuffer.status === "success") {
          toast.success("Project override removed");
        } else if (terminalInBuffer.status === "failed") {
          toast.error(
            `Override removal failed${terminalInBuffer.message ? `: ${terminalInBuffer.message}` : ""}`,
          );
        }
        void queryClient.invalidateQueries({ queryKey: queryKeys.providerPlugins.list() });
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(req.projectId) });
        return;
      }

      const toastId = toast.loading("Removing project override…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (event.status === "success") {
          toast.success("Project override removed", { id: toastId });
        } else if (event.status === "failed") {
          toast.error(
            `Override removal failed${event.message ? `: ${event.message}` : ""}`,
            { id: toastId },
          );
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        }

        if (isTerminal(event.status)) {
          void queryClient.invalidateQueries({ queryKey: queryKeys.providerPlugins.list() });
          void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(req.projectId) });
          setOperationId(null);
          unsub();
          unsubRef.current = null;
        }
      });

      unsubRef.current = unsub;
      setOperationId(opId);
    },

    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : "Unknown error";
      toast.error(`Override removal failed: ${msg}`);
    },
  });

  const clearOperation = useCallback(() => {
    unsubRef.current?.();
    unsubRef.current = null;
    setOperationId(null);
  }, []);

  return { ...mutation, operationId, clearOperation };
}
```

- [ ] **Step 2: Verify typecheck passes**

Run: `cd apps/desktop && pnpm typecheck`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add apps/desktop/renderer/src/features/provider-plugins/use-remove-provider-plugin-override.ts
git commit -m "feat: add useRemoveProviderPluginOverride React hook"
```

---

## Task 7: Redesign ProjectPluginSection Table

**Files:**
- Modify: `apps/desktop/renderer/src/screens/project-detail-screen.tsx`

This is the core UI change. Replace the existing plugin table columns (Plugin | Marketplace | Effective | Provenance | Action) with (Plugin | Marketplace | Project | User | Effective).

- [ ] **Step 1: Add imports for the new hook**

In `project-detail-screen.tsx`, add the import:

```typescript
import { useRemoveProviderPluginOverride } from "../features/provider-plugins/use-remove-provider-plugin-override.js";
```

- [ ] **Step 2: Add helper functions for layer state extraction**

Add these helper functions inside the file (before `ProjectPluginSection`):

```typescript
type ProjectLayerState = "enabled" | "disabled" | "not-set";

function getLayerDeclaration(
  layerBreakdown: Array<{ layer: string; scanStatus: string; declaration: string | null }>,
  layer: string,
): string | null {
  const entry = layerBreakdown.find((lb) => lb.layer === layer);
  return entry?.declaration ?? null;
}

function projectLayerState(
  layerBreakdown: Array<{ layer: string; scanStatus: string; declaration: string | null }>,
): ProjectLayerState {
  const decl = getLayerDeclaration(layerBreakdown, "project");
  if (decl === "enabled") return "enabled";
  if (decl === "disabled") return "disabled";
  return "not-set";
}

function projectStateBadgeClass(state: ProjectLayerState): string {
  switch (state) {
    case "enabled": return "bg-green-100 text-green-700";
    case "disabled": return "bg-zinc-100 text-zinc-500";
    case "not-set": return "";
  }
}

function projectStateLabel(state: ProjectLayerState): string {
  switch (state) {
    case "enabled": return "enabled";
    case "disabled": return "disabled";
    case "not-set": return "—";
  }
}
```

- [ ] **Step 3: Rewrite ProjectPluginSection with per-layer columns**

Replace the effective plugins table (the `{projectView.plugins.length > 0 && (` block, roughly lines 307–358) with:

```tsx
          {projectView.plugins.length > 0 && (
            <div className="overflow-x-auto rounded border border-zinc-200">
              <table className="w-full text-left">
                <thead className="border-b border-zinc-200 bg-zinc-50">
                  <tr>
                    <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Plugin</th>
                    <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Marketplace</th>
                    {canToggle && (
                      <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Project</th>
                    )}
                    {canToggle && (
                      <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">User</th>
                    )}
                    <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Effective</th>
                  </tr>
                </thead>
                <tbody>
                  {projectView.plugins.map((p, i) => {
                    const isLocalOverride = p.provenanceLayer === "local";
                    const projState = projectLayerState(p.layerBreakdown);
                    const userDecl = getLayerDeclaration(p.layerBreakdown, "user");
                    const isUserEnabled = userDecl === "enabled";
                    const projectHasValue = projState !== "not-set";

                    return (
                      <tr key={i} className="border-b border-zinc-100 hover:bg-zinc-50">
                        <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">{p.pluginName}</td>
                        <td className="px-3 py-1.5 text-xs text-zinc-500">{p.marketplaceName || "—"}</td>

                        {/* Project column — 3-state cycle */}
                        {canToggle && (
                          <td className="px-3 py-1.5 text-xs">
                            {isLocalOverride ? (
                              <span className="text-xs text-zinc-400 opacity-40">overridden</span>
                            ) : (
                              <button
                                onClick={() => {
                                  if (projState === "not-set") {
                                    handleToggleProjectPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, true);
                                  } else if (projState === "enabled") {
                                    handleToggleProjectPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, false);
                                  } else {
                                    handleRemoveProjectOverride(projectView.providerKey, p.pluginName, p.marketplaceName);
                                  }
                                }}
                                disabled={isOperationInFlight}
                                title={
                                  projState === "not-set"
                                    ? "Click to enable at project level"
                                    : projState === "enabled"
                                      ? "Click to disable at project level"
                                      : "Click to clear project override"
                                }
                                className={`rounded px-1.5 py-0.5 font-medium disabled:cursor-not-allowed disabled:opacity-40 ${
                                  projState === "not-set"
                                    ? "text-zinc-400 hover:bg-zinc-100"
                                    : projectStateBadgeClass(projState) + " hover:opacity-80"
                                }`}
                              >
                                {projectStateLabel(projState)}
                              </button>
                            )}
                          </td>
                        )}

                        {/* User column — 2-state toggle */}
                        {canToggle && (
                          <td className="px-3 py-1.5 text-xs">
                            {isLocalOverride ? (
                              <span className="text-xs text-zinc-400 opacity-40">overridden</span>
                            ) : (
                              <div className={projectHasValue ? "opacity-40" : ""}>
                                <button
                                  onClick={() => handleToggleUserPlugin(projectView.providerKey, p.pluginName, p.marketplaceName, !isUserEnabled)}
                                  disabled={isOperationInFlight}
                                  title={
                                    projectHasValue
                                      ? "Project layer overrides this setting"
                                      : isUserEnabled
                                        ? "Disable globally"
                                        : "Enable globally"
                                  }
                                  className={`rounded px-1.5 py-0.5 font-medium hover:opacity-80 disabled:cursor-not-allowed disabled:opacity-40 ${
                                    isUserEnabled
                                      ? "bg-green-100 text-green-700"
                                      : "bg-zinc-100 text-zinc-500"
                                  }`}
                                >
                                  {isUserEnabled ? "enabled" : "disabled"}
                                </button>
                              </div>
                            )}
                          </td>
                        )}

                        {/* Effective column — read-only */}
                        <td className="px-3 py-1.5 text-xs">
                          <span className={`rounded px-1.5 py-0.5 font-medium ${effectiveStatusClass(p.effectiveStatus)}`}>
                            {p.effectiveStatus}
                          </span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
```

- [ ] **Step 4: Update the component to use both hooks and add handler functions**

Update the `ProjectPluginSection` function body. After the existing `setEnabledMutation` line, add:

```typescript
  const removeOverrideMutation = useRemoveProviderPluginOverride();
  const isRemovingOverride = removeOverrideMutation.isPending || removeOverrideMutation.operationId != null;
```

Update `isOperationInFlight` to include the new mutation:

```typescript
  const isOperationInFlight = isTogglingPlugin || isRemovingOverride || scanInFlight;
```

Rename the existing `handleTogglePlugin` and add two new handlers:

```typescript
  function handleToggleProjectPlugin(providerKey: string, pluginName: string, marketplaceName: string, enabled: boolean): void {
    setEnabledMutation.mutate({ providerKey, pluginName, marketplaceName, layer: "project", projectId, enabled });
  }

  function handleToggleUserPlugin(providerKey: string, pluginName: string, marketplaceName: string, enabled: boolean): void {
    setEnabledMutation.mutate({ providerKey, pluginName, marketplaceName, layer: "user", projectId: 0, enabled });
  }

  function handleRemoveProjectOverride(providerKey: string, pluginName: string, marketplaceName: string): void {
    removeOverrideMutation.mutate({ providerKey, pluginName, marketplaceName, layer: "project", projectId });
  }
```

Remove the old `handleTogglePlugin` function.

Move `const canToggle = JSON_WRITE_PROVIDERS.has(projectView.providerKey);` from inside the `.map()` callback up to the `projectViews.map` level (where `projectView` is available), since it's now used in the `<thead>` as well.

- [ ] **Step 5: Verify typecheck**

Run: `cd apps/desktop && pnpm typecheck`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add apps/desktop/renderer/src/screens/project-detail-screen.tsx
git commit -m "feat: redesign ProjectPluginSection with per-layer Project/User columns and 3-state toggle"
```

---

## Task 8: Plugins Screen Label Change

**Files:**
- Modify: `apps/desktop/renderer/src/screens/plugins-screen.tsx`

- [ ] **Step 1: Change button labels**

In `plugins-screen.tsx`, find the `PluginToggleButton` component (around line 59–79). Change the label from:

```tsx
      {isEnabled ? "Disable" : "Enable"}
```

to:

```tsx
      {isEnabled ? "Disable globally" : "Enable globally"}
```

- [ ] **Step 2: Verify typecheck**

Run: `cd apps/desktop && pnpm typecheck`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add apps/desktop/renderer/src/screens/plugins-screen.tsx
git commit -m "feat: change plugin toggle labels to 'Disable globally'/'Enable globally'"
```

---

## Task 9: Frontend Tests

**Files:**
- Test: `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx` (or new file if this doesn't exist)

Check if the test file exists first. If not, create a minimal test file for the plugin section behavior. The key test cases per the spec:

- [ ] **Step 1: Check if test file exists and decide test location**

Run: `ls apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx 2>/dev/null || echo "not found"`

If it doesn't exist, check what test infrastructure exists for screens and create a focused test file. The tests need to verify:

1. Plugin with project=enabled, user=disabled → Project column shows "enabled", User column dimmed (opacity-40)
2. Plugin with project=not-set, user=enabled → Project column shows "—", User column active
3. Plugin with local override → both columns show "overridden"
4. Click project toggle cycle: not-set → enabled (calls setEnabled with layer=project, enabled=true)

Since the `ProjectPluginSection` is deeply nested and requires mocking `useProviderPluginList`, `useSetProviderPluginEnabled`, and `useRemoveProviderPluginOverride`, the test needs substantial setup. Write a focused integration test:

```typescript
// This test file should mock:
// - useProviderPluginList (return fixture data with layerBreakdown)
// - useSetProviderPluginEnabled (return stub mutation)
// - useRemoveProviderPluginOverride (return stub mutation)
// Then render ProjectPluginSection and assert column content.
```

The exact test implementation depends on the screen's export structure. If `ProjectPluginSection` is not exported, the test will need to render the full `ProjectDetailScreen` with route setup, or the component should be extracted.

- [ ] **Step 2: Write tests covering the 3 spec scenarios**

Create focused tests that verify:
- Layer state extraction helpers (`projectLayerState`, `getLayerDeclaration`)
- Correct badge rendering for each combination
- User column dimming when project layer has a value

- [ ] **Step 3: Run tests**

Run: `cd apps/desktop && pnpm test -- --run`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add apps/desktop/renderer/src/screens/__tests__/
git commit -m "test: add ProjectPluginSection per-layer column tests"
```

---

## Task 10: Full Verification

- [ ] **Step 1: Run all Go tests**

Run: `cd core-go && go test ./...`
Expected: PASS

- [ ] **Step 2: Run all frontend tests**

Run: `cd apps/desktop && pnpm test -- --run`
Expected: PASS

- [ ] **Step 3: Run typecheck**

Run: `cd apps/desktop && pnpm typecheck`
Expected: PASS

- [ ] **Step 4: Run contract drift check**

Run: `cd apps/desktop && pnpm check:contracts-drift`
Expected: PASS

- [ ] **Step 5: Build and smoke test**

Run: `cd apps/desktop && pnpm build`
Expected: Build succeeds

Launch the app and verify:
1. Project detail → Plugin section shows Project, User, Effective columns
2. Project column shows "—" for not-set, "enabled"/"disabled" badges for set values
3. Click cycle works: — → enabled → disabled → —
4. User column dimmed when project layer is set
5. Local override → both columns show "overridden"
6. Plugins screen → buttons say "Disable globally"/"Enable globally"
