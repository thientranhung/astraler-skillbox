# Plugin Layer Toggle Clarity

## Problem

The plugin system supports three layers with precedence local > project > user, but the UI presents a single "Disable" button with no indication of which layer is being toggled. Users cannot:

- See which layer controls a plugin's current state
- Override a user-level setting at the project level (or vice versa)
- Understand why a plugin is enabled/disabled in a given project

## Scope

- **In scope**: Project detail plugin table — add per-layer columns with toggles
- **In scope**: Plugins screen (global) — clarify button labels
- **Out of scope**: Local layer remains read-only (shown as "overridden locally")
- **Out of scope**: No changes to scan or resolution logic

## Precedence Rule

```
effective = project layer value (if set) → user layer value (fallback)
```

- Project **enabled** → effective = enabled, regardless of user layer
- Project **disabled** → effective = disabled, regardless of user layer
- Project **not set** → effective = user layer value
- Local layer overrides everything (read-only, not toggleable)

## §1 — Project Detail Plugin Table Redesign

### Current state

The plugin table in `ProjectPluginSection` has columns: Plugin | Marketplace | Effective | Provenance | Action. The Action column shows a single Disable/Enable button that always writes to project layer. There is no way to toggle the user layer from project detail, and no visibility into which layer is active.

### Target state

Replace with columns: Plugin | Marketplace | Project | User | Effective.

```
Plugin       | Marketplace | Project         | User       | Effective
─────────────┼─────────────┼─────────────────┼────────────┼──────────
superpowers  | official    | ── (not set) ── | ✓ enabled  | enabled
my-plugin    | custom      | ✕ disabled      | ✓ enabled  | disabled
other-plugin | official    | ✓ enabled       | ✕ disabled | enabled
```

### Project column — 3-state cycle toggle

The project column cycles through three states on click: **not set → enabled → disabled → not set**.

- **not set → enabled**: call `providerPlugin.setEnabled({ layer: "project", projectId, enabled: true })`
- **enabled → disabled**: call `providerPlugin.setEnabled({ layer: "project", projectId, enabled: false })`
- **disabled → not set**: call `providerPlugin.removeOverride({ layer: "project", projectId })` — new command

Visual states:
- `not set`: dimmed text "—", no background
- `enabled`: green badge "enabled"
- `disabled`: zinc badge "disabled"

### User column — 2-state toggle

Standard enable/disable toggle. Calls `providerPlugin.setEnabled({ layer: "user", enabled })`.

When project layer has a value (enabled or disabled), the User column is **visually dimmed** (opacity-40) with a tooltip "Project layer overrides this setting" — the toggle still works but user sees it has no current effect.

### Effective column — read-only

Displays the resolved status. Uses existing `effectiveStatus` from `PluginEffectiveEntry`. Read-only, no interaction.

### Local layer override

When `provenanceLayer === "local"`, both Project and User columns show "overridden" in dimmed text. No toggle available. Same behavior as current "Overridden locally" label.

## §2 — New Command: `providerPlugin.removeOverride`

### Purpose

Remove a plugin's declaration from a specific layer's settings file, effectively returning to "not set" so the lower-priority layer takes effect.

### Contract

```json
{
  "ProviderPluginRemoveOverrideRequest": {
    "type": "object",
    "properties": {
      "providerKey": { "type": "string" },
      "pluginName": { "type": "string" },
      "marketplaceName": { "type": "string" },
      "layer": { "type": "string", "enum": ["project"] },
      "projectId": { "type": "integer" }
    },
    "required": ["providerKey", "pluginName", "marketplaceName", "layer", "projectId"]
  },
  "ProviderPluginRemoveOverrideResponse": {
    "type": "object",
    "properties": {
      "operationId": { "type": "integer" }
    },
    "required": ["operationId"]
  }
}
```

Only `layer: "project"` is supported. Removing from user layer is not supported (user layer always has a value).

### Backend implementation

1. **ProviderPluginService.RemoveOverride(ctx, providerKey, pluginName, marketplaceName, layer, projectId)**
   - Read project settings JSON file
   - Remove the plugin entry from the plugins array/map
   - Write back the modified JSON
   - Rescan the layer to update DB state
   - Return operation ID

2. **RPC handler**: `provider_plugin_remove_override.go` — validates inputs, calls service, returns operation ID.

3. **Wiring**: Register in `main.go` builder chain + IPC allowlist + contract.

## §3 — Plugins Screen (Global) Label Change

Change toggle button label from "Disable"/"Enable" to "Disable globally"/"Enable globally" to make clear this affects all projects where the project layer is not set.

No layout or functional changes.

> **Update (Slice A — naming-alignment, 2026-05-28):** Label `Disable globally` / `Enable globally` đã được đổi thành `Disable` / `Enable` sau khi sidebar đã ghi rõ `Global Plugins` — chữ "globally" trở nên redundant trong context đó.

## §4 — Layer Breakdown Data

The `PluginEffectiveEntry` already contains `LayerBreakdown []PluginLayerBreakdown` with per-layer `Declaration` (enabled/disabled/nil). The frontend contract `PPProjectPlugin` already exposes `layerBreakdown`. The project detail table can derive:

- **Project column state**: find breakdown entry where `layer === "project"` → declaration === "enabled" / "disabled" / null (not set)
- **User column state**: find breakdown entry where `layer === "user"` → declaration === "enabled" / "disabled"

No backend changes needed for reading layer state — data is already available.

## §5 — Testing

### Go tests

- `provider_plugin_service_test.go`: Test `RemoveOverride` — write a plugin at project layer, call remove, verify rescan shows absent.
- `provider_plugin_remove_override_handler_test.go`: Test RPC handler validation and happy path.

### Frontend tests

- `project-detail-screen.test.tsx` or extracted plugin section test:
  - Render plugin with project=enabled, user=disabled → verify Project column shows "enabled", User column dimmed
  - Render plugin with project=not set, user=enabled → verify Project column shows "—", User column active
  - Render plugin with local override → verify both columns show "overridden"
  - Click project toggle: verify cycle not set → enabled → disabled → not set
  - Click user toggle: verify calls setEnabled with layer="user"

### Contract drift

- `pnpm check:contracts-drift` must pass after adding new contract

## §6 — Files Touched

### Backend
- `core-go/internal/domain/provider_plugin.go` — no changes expected
- `core-go/internal/services/provider_plugin_service.go` — add `RemoveOverride` method
- `core-go/internal/rpc/handlers/provider_plugin_remove_override.go` — new handler
- `core-go/cmd/skillbox-core/main.go` — register handler + capabilities
- `core-go/internal/services/provider_plugin_service_test.go` — test RemoveOverride

### Contract
- `shared/api-contracts/methods/providerPlugin.removeOverride.json` — new contract
- `shared/api-contracts/index.json` — add codegen entry
- `shared/generated/` — regenerated types

### Frontend
- `apps/desktop/renderer/src/features/provider-plugins/use-remove-provider-plugin-override.ts` — new hook
- `apps/desktop/renderer/src/screens/project-detail-screen.tsx` — redesign ProjectPluginSection table
- `apps/desktop/renderer/src/screens/plugins-screen.tsx` — change button labels
- `apps/desktop/electron/main/core-process/method-allowlist.ts` — add new method
- `apps/desktop/renderer/src/lib/core-client/methods.ts` — add new method binding

### Tests
- `core-go/internal/services/provider_plugin_service_test.go`
- `apps/desktop/renderer/src/features/provider-plugins/__tests__/use-remove-provider-plugin-override.test.tsx`
- `apps/desktop/renderer/src/screens/__tests__/project-detail-screen.test.tsx` (or plugin section test)

## Non-Goals

- No changes to plugin scan logic
- No changes to effective resolution algorithm
- No new DB tables or migrations
- No local layer toggling (remains read-only)
- No changes to project list Plugins column
