# Slice 2E Provider Terminology And Install Targets Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace user-facing `Generic Agents` wording with `Shared Agent Skills (.agents)` while keeping `generic_agents` as the stable internal provider key, and add read-only install target metadata for the next symlink slice.

**Architecture:** Provider definitions remain the persisted detection source. Install targets are non-persisted, core-only metadata derived in code so the future install flow can choose `.agents/skills` or `.claude/skills` without renaming providers, changing JSON-RPC contracts, or writing files in this slice.

**Tech Stack:** Go core, SQLite migrations, JSON-RPC contract tests, React renderer tests where labels are asserted.

---

## File Map

- `core-go/migrations/000004_shared_agent_display_names.up.sql`: update provider display names and database version.
- `core-go/migrations/000004_shared_agent_display_names.down.sql`: restore previous display names and database version.
- `core-go/internal/repositories/migration_000004_test.go`: verify display-name seed and stable provider keys.
- `core-go/internal/providers/install_targets.go`: read-only core install target metadata, not exposed through JSON-RPC in 2E.
- `core-go/internal/providers/install_targets_test.go`: verify target IDs, provider keys, paths, and compatible labels.
- `core-go/internal/rpc/handlers/project_contract_test.go`: update display-name fixtures while keeping `generic_agents`.
- `core-go/internal/rpc/handlers/project_handler_test.go`: assert list/detail responses expose shared display name and stable key.
- `docs/superpowers/specs/2026-05-25-skillbox-slice-2e-provider-terminology-install-targets-design.md`: already created source of truth.

## Task 1: Seed User-Facing Provider Display Names

**Files:**
- Create: `core-go/migrations/000004_shared_agent_display_names.up.sql`
- Create: `core-go/migrations/000004_shared_agent_display_names.down.sql`
- Create: `core-go/internal/repositories/migration_000004_test.go`
- Modify: `core-go/internal/repositories/app_settings_repo_test.go`

- [ ] **Step 1: Add failing migration test**

Create `core-go/internal/repositories/migration_000004_test.go`:

```go
package repositories

import "testing"

func TestMigration000004_SharedAgentDisplayNames(t *testing.T) {
	db := NewTestDB(t)

	cases := []struct {
		key  string
		name string
	}{
		{"generic_agents", "Shared Agent Skills (.agents)"},
		{"claude", "Claude (.claude)"},
	}

	for _, c := range cases {
		var got string
		if err := db.QueryRow("SELECT display_name FROM provider_definitions WHERE key=?", c.key).Scan(&got); err != nil {
			t.Fatalf("query display_name for %s: %v", c.key, err)
		}
		if got != c.name {
			t.Errorf("%s display_name: got %q want %q", c.key, got, c.name)
		}
	}
}
```

- [ ] **Step 2: Run test and confirm failure**

Run:

```bash
cd core-go && go test ./internal/repositories -run TestMigration000004_SharedAgentDisplayNames -count=1
```

Expected: FAIL because `Generic Agents` and `Claude` are still seeded.

- [ ] **Step 3: Add migration**

Create `core-go/migrations/000004_shared_agent_display_names.up.sql`:

```sql
-- 000004_shared_agent_display_names.up.sql
UPDATE provider_definitions
SET display_name = 'Shared Agent Skills (.agents)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'generic_agents' AND display_name = 'Generic Agents';

UPDATE provider_definitions
SET display_name = 'Claude (.claude)', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'claude' AND display_name = 'Claude';

UPDATE app_settings SET database_version = 4, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1;
```

Create `core-go/migrations/000004_shared_agent_display_names.down.sql`:

```sql
-- 000004_shared_agent_display_names.down.sql
UPDATE provider_definitions
SET display_name = 'Generic Agents', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'generic_agents' AND display_name = 'Shared Agent Skills (.agents)';

UPDATE provider_definitions
SET display_name = 'Claude', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE key = 'claude' AND display_name = 'Claude (.claude)';

UPDATE app_settings SET database_version = 3, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now') WHERE id = 1;
```

- [ ] **Step 4: Update database version test**

In `core-go/internal/repositories/app_settings_repo_test.go`, change expected `DatabaseVersion` from `3` to `4`.

- [ ] **Step 5: Run repository tests**

Run:

```bash
cd core-go && go test ./internal/repositories -count=1
```

Expected: PASS.

## Task 2: Add Read-Only Install Target Metadata

**Files:**
- Create: `core-go/internal/providers/install_targets.go`
- Create: `core-go/internal/providers/install_targets_test.go`

- [ ] **Step 1: Add failing tests**

Create `core-go/internal/providers/install_targets_test.go`:

```go
package providers_test

import (
	"reflect"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/providers"
)

func TestInstallTargets(t *testing.T) {
	targets := providers.InstallTargets()
	if len(targets) != 2 {
		t.Fatalf("target count: got %d want 2", len(targets))
	}

	shared := targets[0]
	if shared.ID != "shared_agents" {
		t.Errorf("shared ID: got %q", shared.ID)
	}
	if shared.ProviderKey != providers.GenericAgentsKey {
		t.Errorf("shared provider key: got %q", shared.ProviderKey)
	}
	if shared.DisplayName != "Shared Agent Skills (.agents)" {
		t.Errorf("shared display name: got %q", shared.DisplayName)
	}
	if shared.RelativeSkillsPath != providers.GenericAgentsSkillsPath {
		t.Errorf("shared path: got %q", shared.RelativeSkillsPath)
	}
	if !reflect.DeepEqual(shared.CompatibleLabels, []string{"Codex", "Antigravity", "compatible agents"}) {
		t.Errorf("shared compatible labels: got %#v", shared.CompatibleLabels)
	}

	claude := targets[1]
	if claude.ID != providers.ClaudeKey {
		t.Errorf("claude ID: got %q", claude.ID)
	}
	if claude.ProviderKey != providers.ClaudeKey {
		t.Errorf("claude provider key: got %q", claude.ProviderKey)
	}
	if claude.DisplayName != "Claude (.claude)" {
		t.Errorf("claude display name: got %q", claude.DisplayName)
	}
	if claude.RelativeSkillsPath != providers.ClaudeSkillsPath {
		t.Errorf("claude path: got %q", claude.RelativeSkillsPath)
	}
}

func TestInstallTargetByProviderKey(t *testing.T) {
	target, ok := providers.InstallTargetByProviderKey(providers.GenericAgentsKey)
	if !ok {
		t.Fatal("expected target for generic_agents")
	}
	if target.ID != "shared_agents" {
		t.Errorf("target ID: got %q want shared_agents", target.ID)
	}

	if _, ok := providers.InstallTargetByProviderKey("codex"); ok {
		t.Fatal("codex must not be exposed as a detected provider target in Slice 2E")
	}
}
```

- [ ] **Step 2: Run test and confirm failure**

Run:

```bash
cd core-go && go test ./internal/providers -run 'TestInstallTarget' -count=1
```

Expected: FAIL because install target helpers do not exist.

- [ ] **Step 3: Add implementation**

Create `core-go/internal/providers/install_targets.go`:

```go
package providers

// InstallTarget is read-only core metadata for future install flows.
// It is not persisted, not exposed through JSON-RPC in Slice 2E, and must not replace provider keys.
type InstallTarget struct {
	ID                 string
	ProviderKey        string
	DisplayName        string
	RelativeSkillsPath string
	CompatibleLabels   []string
}

func InstallTargets() []InstallTarget {
	return []InstallTarget{
		{
			ID:                 "shared_agents",
			ProviderKey:        GenericAgentsKey,
			DisplayName:        "Shared Agent Skills (.agents)",
			RelativeSkillsPath: GenericAgentsSkillsPath,
			CompatibleLabels:   []string{"Codex", "Antigravity", "compatible agents"},
		},
		{
			ID:                 ClaudeKey,
			ProviderKey:        ClaudeKey,
			DisplayName:        "Claude (.claude)",
			RelativeSkillsPath: ClaudeSkillsPath,
			CompatibleLabels:   []string{"Claude"},
		},
	}
}

func InstallTargetByProviderKey(providerKey string) (InstallTarget, bool) {
	for _, target := range InstallTargets() {
		if target.ProviderKey == providerKey {
			return target, true
		}
	}
	return InstallTarget{}, false
}
```

- [ ] **Step 4: Run provider tests**

Run:

```bash
cd core-go && go test ./internal/providers -count=1
```

Expected: PASS.

## Task 3: Update RPC Fixtures And Label Assertions

**Files:**
- Modify: `core-go/internal/rpc/handlers/project_contract_test.go`
- Modify: `core-go/internal/rpc/handlers/project_handler_test.go`

- [ ] **Step 1: Update contract fixtures**

In `project_contract_test.go`, keep every `ProviderKey: "generic_agents"` or `Key: "generic_agents"` unchanged. Replace display names:

```go
DisplayName: "Shared Agent Skills (.agents)",
```

- [ ] **Step 2: Strengthen handler response assertions**

In `TestProjectListHandler_WithProjects`, extend the anonymous response provider struct so the test asserts key and display name:

```go
Providers []struct {
	Key         string `json:"key"`
	DisplayName string `json:"displayName"`
} `json:"providers"`
```

After existing assertions, add:

```go
if resp.Projects[0].Providers[0].Key != "generic_agents" {
	t.Errorf("provider key: got %q want generic_agents", resp.Projects[0].Providers[0].Key)
}
if resp.Projects[0].Providers[0].DisplayName != "Shared Agent Skills (.agents)" {
	t.Errorf("provider displayName: got %q", resp.Projects[0].Providers[0].DisplayName)
}
```

In `TestProjectGetHandler_Success`, use a typed provider response and assert the same key/display pair.

- [ ] **Step 3: Update stub data**

In `project_handler_test.go`, update stub provider display names:

```go
ProviderDisplayName: "Shared Agent Skills (.agents)",
```

- [ ] **Step 4: Run handler tests**

Run:

```bash
cd core-go && go test ./internal/rpc/handlers -count=1
```

Expected: PASS.

## Task 4: Renderer And Wording Verification

**Files:**
- No required code files unless the search finds active renderer-visible `Generic Agents` text.

- [ ] **Step 1: Search active renderer text**

Run:

```bash
rg "Generic Agents" apps/desktop/renderer core-go/internal/rpc core-go/internal/services core-go/internal/domain
```

Expected: only migration down SQL and intentional legacy assertions should contain `Generic Agents`; renderer-visible source should not.

- [ ] **Step 2: Search current architecture docs with scoped exceptions**

Run:

```bash
rg "Generic Agents" docs/09-ui-wireframes.md docs/superpowers/specs docs/superpowers/plans
```

Expected: older historical slice docs may still mention `Generic Agents`; the new 2E spec/plan should only mention it when describing the term being replaced or legacy migration behavior.

- [ ] **Step 3: Update active UI docs if needed**

If `docs/09-ui-wireframes.md` still presents current/future UI labels as `Generic Agents`, replace those user-facing examples with `Shared Agent Skills (.agents)`. Do not rewrite historical Slice 2A/2D specs.

## Task 5: Final Verification And Commit

**Files:**
- All files touched above.

- [ ] **Step 1: Run full Go tests**

Run:

```bash
cd core-go && go test ./...
```

Expected: PASS.

- [ ] **Step 2: Run desktop contract check**

Run:

```bash
cd apps/desktop && pnpm check:contracts-drift
```

Expected: PASS.

- [ ] **Step 3: Run desktop typecheck and tests**

Run:

```bash
cd apps/desktop && pnpm typecheck && pnpm test
```

Expected: PASS.

- [ ] **Step 4: Run build and diff check**

Run:

```bash
cd apps/desktop && pnpm build
cd ../.. && git diff --check
```

Expected: PASS.

- [ ] **Step 5: Commit**

Run:

```bash
git add core-go/migrations/000004_shared_agent_display_names.up.sql \
  core-go/migrations/000004_shared_agent_display_names.down.sql \
  core-go/internal/repositories/migration_000004_test.go \
  core-go/internal/repositories/app_settings_repo_test.go \
  core-go/internal/providers/install_targets.go \
  core-go/internal/providers/install_targets_test.go \
  core-go/internal/rpc/handlers/project_contract_test.go \
  core-go/internal/rpc/handlers/project_handler_test.go
git commit -m "Add shared agent install target terminology"
```

## Self-Review

- Spec coverage: covers stable `generic_agents` key, display-name correction, read-only target metadata, no filesystem writes, and tests.
- Placeholder scan: no TODO/TBD placeholders remain.
- Type consistency: `shared_agents`, `generic_agents`, `.agents/skills`, `claude`, and `.claude/skills` are used consistently.
