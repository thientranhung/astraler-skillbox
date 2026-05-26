# Provider Registry PR-2A Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add provider path overrides + reset to Settings (UI/API/storage only, no scan/install wiring).

**Architecture:** New `provider_path_overrides` table stores one override per (provider, scope, purpose) as JSON paths array. `ProviderRegistryService.List` merges overrides into builtin candidates (source="override"). Two new RPC commands (`provider.updatePaths`, `provider.resetPaths`) handle writes with path validation.

**Tech Stack:** Go + SQLite (modernc.org/sqlite), jrpc2, React + TanStack Query, Zod-free validation (domain errors), Vitest + React Testing Library.

---

## File Map

**Create:**
- `core-go/migrations/000010_provider_path_overrides.up.sql`
- `core-go/migrations/000010_provider_path_overrides.down.sql`
- `core-go/internal/repositories/migration_000010_test.go`
- `core-go/internal/repositories/provider_override_repo.go`
- `core-go/internal/repositories/provider_override_repo_test.go`
- `core-go/internal/rpc/handlers/provider_update_paths.go`
- `core-go/internal/rpc/handlers/provider_reset_paths.go`
- `core-go/internal/rpc/handlers/provider_update_reset_handler_test.go`
- `shared/api-contracts/methods/provider.updatePaths.json`
- `shared/api-contracts/methods/provider.resetPaths.json`
- `apps/desktop/renderer/src/features/providers/use-update-provider-paths.ts`
- `apps/desktop/renderer/src/features/providers/use-reset-provider-paths.ts`
- `apps/desktop/renderer/src/features/providers/__tests__/use-update-provider-paths.test.tsx`
- `apps/desktop/renderer/src/features/providers/__tests__/use-reset-provider-paths.test.tsx`
- `apps/desktop/renderer/src/features/providers/provider-paths-editor.tsx`
- `apps/desktop/renderer/src/features/providers/__tests__/provider-paths-editor.test.tsx`

**Modify:**
- `core-go/internal/domain/provider.go` — add `Source` to ProviderPathCandidate, add ProviderPathOverride
- `core-go/internal/services/interfaces.go` — add ProviderOverrideRepo interface, extend ProviderRegistryRepo with GetByKey
- `core-go/internal/services/provider_registry_service.go` — add UpdatePaths, ResetPaths, merge in List
- `core-go/internal/services/provider_registry_service_test.go` — update mock, add tests
- `core-go/internal/rpc/handlers/provider_list.go` — use candidate.Source instead of hardcoded "builtin"
- `core-go/internal/rpc/handlers/provider_list_handler_test.go` — update override source test
- `core-go/internal/app/wire.go` — register new handlers, inject overrideRepo
- `core-go/cmd/skillbox-core/main.go` — wire overrideRepo, update capabilities list
- `shared/api-contracts/index.json` — add two new contract entries
- `apps/desktop/renderer/src/lib/core-client/methods.ts` — add updateProviderPaths, resetProviderPaths
- `apps/desktop/renderer/src/screens/settings-screen.tsx` — override indicator, edit button, reset button
- `apps/desktop/renderer/src/screens/__tests__/settings-screen.test.tsx` — add override/edit/reset tests

---

## Task 1: Migration 000010 — provider_path_overrides table

**Files:**
- Create: `core-go/migrations/000010_provider_path_overrides.up.sql`
- Create: `core-go/migrations/000010_provider_path_overrides.down.sql`
- Create: `core-go/internal/repositories/migration_000010_test.go`

- [ ] **Step 1: Write migration up**

```sql
-- 000010_provider_path_overrides.up.sql
-- PR-2A: user overrides for built-in provider path candidates.
-- One override row per (provider_definition_id, scope, purpose).
-- paths_json stores a JSON array of path strings replacing builtin defaults.

CREATE TABLE IF NOT EXISTS provider_path_overrides (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_definition_id INTEGER NOT NULL REFERENCES provider_definitions(id),
    scope                 TEXT NOT NULL,
    purpose               TEXT NOT NULL,
    paths_json            TEXT NOT NULL DEFAULT '[]',
    created_at            TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at            TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    UNIQUE(provider_definition_id, scope, purpose)
);

UPDATE app_settings
   SET database_version = 10, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
```

- [ ] **Step 2: Write migration down**

```sql
-- 000010_provider_path_overrides.down.sql
DROP TABLE IF EXISTS provider_path_overrides;

UPDATE app_settings
   SET database_version = 9, updated_at = strftime('%Y-%m-%dT%H:%M:%SZ','now')
 WHERE id = 1;
```

- [ ] **Step 3: Write migration test**

```go
package repositories

import (
    "testing"
)

func TestMigration000010_TableExists(t *testing.T) {
    db := NewTestDB(t)

    var name string
    err := db.QueryRow(
        `SELECT name FROM sqlite_master WHERE type='table' AND name='provider_path_overrides'`,
    ).Scan(&name)
    if err != nil {
        t.Fatalf("provider_path_overrides table not created: %v", err)
    }
    if name != "provider_path_overrides" {
        t.Errorf("table name: got %q want provider_path_overrides", name)
    }
}

func TestMigration000010_FKConstraint(t *testing.T) {
    db := NewTestDB(t)

    _, err := db.Exec(`
        INSERT INTO provider_path_overrides (provider_definition_id, scope, purpose, paths_json)
        VALUES (99999, 'project', 'detect', '[]')
    `)
    if err == nil {
        t.Error("expected FK constraint error for nonexistent provider_definition_id, got nil")
    }
}

func TestMigration000010_UniqueConstraint(t *testing.T) {
    db := NewTestDB(t)

    var providerID int64
    if err := db.QueryRow(`SELECT id FROM provider_definitions WHERE key='claude'`).Scan(&providerID); err != nil {
        t.Fatalf("claude not found: %v", err)
    }

    _, err := db.Exec(`
        INSERT INTO provider_path_overrides (provider_definition_id, scope, purpose, paths_json)
        VALUES (?, 'project', 'detect', '["custom"]')
    `, providerID)
    if err != nil {
        t.Fatalf("first insert failed: %v", err)
    }

    _, err = db.Exec(`
        INSERT INTO provider_path_overrides (provider_definition_id, scope, purpose, paths_json)
        VALUES (?, 'project', 'detect', '["another"]')
    `, providerID)
    if err == nil {
        t.Error("expected UNIQUE constraint error on second insert, got nil")
    }
}

func TestMigration000010_DatabaseVersion(t *testing.T) {
    db := NewTestDB(t)

    var dbVersion int
    if err := db.QueryRow(`SELECT database_version FROM app_settings WHERE id=1`).Scan(&dbVersion); err != nil {
        t.Fatalf("query database_version: %v", err)
    }
    if dbVersion != 10 {
        t.Errorf("database_version: got %d want 10", dbVersion)
    }
}
```

- [ ] **Step 4: Run tests**

```bash
cd core-go && go test ./internal/repositories/... -run TestMigration000010 -v
```

Expected: 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add core-go/migrations/000010_provider_path_overrides.up.sql \
        core-go/migrations/000010_provider_path_overrides.down.sql \
        core-go/internal/repositories/migration_000010_test.go
git commit -m "Add migration 000010: provider_path_overrides table (PR-2A)"
```

---

## Task 2: Domain types update

**Files:**
- Modify: `core-go/internal/domain/provider.go`

- [ ] **Step 1: Add Source field + ProviderPathOverride struct**

In `core-go/internal/domain/provider.go`, add `Source string` to `ProviderPathCandidate` and add `ProviderPathOverride`:

```go
// ProviderPathCandidate is a single path candidate for a provider, as stored in
// provider_path_candidates. Scope and purpose classify how it is used.
type ProviderPathCandidate struct {
    ID                   int64
    ProviderDefinitionID int64
    RelativePath         string
    Scope                string // "project" or "global"
    Purpose              string // "detect", "skills", "config", "commands"
    Priority             int
    VerificationStatus   string // "verified", "assumed", "experimental"
    Source               string // "builtin", "override", "custom"
}

// ProviderPathOverride stores a user path override for a provider's (scope, purpose) slot.
// One row per (provider_definition_id, scope, purpose). Paths replaces all builtin candidates
// for that slot. Empty Paths slice means use builtin defaults.
type ProviderPathOverride struct {
    ID                   int64
    ProviderDefinitionID int64
    Scope                string
    Purpose              string
    Paths                []string
}
```

- [ ] **Step 2: Run existing tests to confirm no breakage**

```bash
cd core-go && go test ./...
```

Expected: all PASS (Source="" for existing builtin candidates — empty string won't break anything since handler currently hardcodes "builtin").

---

## Task 3: ProviderOverrideRepo — concrete repo

**Files:**
- Create: `core-go/internal/repositories/provider_override_repo.go`
- Create: `core-go/internal/repositories/provider_override_repo_test.go`

- [ ] **Step 1: Write failing tests**

```go
// core-go/internal/repositories/provider_override_repo_test.go
package repositories

import (
    "context"
    "testing"

    "github.com/astraler/skillbox/core-go/internal/domain"
)

func getClaudeProviderID(t *testing.T, db interface{ QueryRow(string, ...any) interface{ Scan(...any) error } }) int64 {
    t.Helper()
    var id int64
    // Use the *sql.DB from NewTestDB
    return id
}

// helper to get provider ID by key for tests
func providerIDByKey(t *testing.T, r *ProviderOverrideRepo, ctx context.Context, key string) int64 {
    t.Helper()
    id, err := r.GetProviderIDByKey(ctx, key)
    if err != nil {
        t.Fatalf("GetProviderIDByKey(%q): %v", key, err)
    }
    if id == 0 {
        t.Fatalf("GetProviderIDByKey(%q): not found", key)
    }
    return id
}

func TestProviderOverrideRepo_Upsert_And_ListAll(t *testing.T) {
    db := NewTestDB(t)
    r := NewProviderOverrideRepo(db)
    ctx := context.Background()

    provID := providerIDByKey(t, r, ctx, "claude")

    err := r.Upsert(ctx, domain.ProviderPathOverride{
        ProviderDefinitionID: provID,
        Scope:                "project",
        Purpose:              "detect",
        Paths:                []string{".custom-claude", ".claude-alt"},
    })
    if err != nil {
        t.Fatalf("Upsert: %v", err)
    }

    all, err := r.ListAll(ctx)
    if err != nil {
        t.Fatalf("ListAll: %v", err)
    }
    if len(all) != 1 {
        t.Fatalf("ListAll len: got %d want 1", len(all))
    }
    got := all[0]
    if got.ProviderDefinitionID != provID {
        t.Errorf("ProviderDefinitionID: got %d want %d", got.ProviderDefinitionID, provID)
    }
    if got.Scope != "project" {
        t.Errorf("Scope: got %q want project", got.Scope)
    }
    if got.Purpose != "detect" {
        t.Errorf("Purpose: got %q want detect", got.Purpose)
    }
    if len(got.Paths) != 2 || got.Paths[0] != ".custom-claude" || got.Paths[1] != ".claude-alt" {
        t.Errorf("Paths: got %v want [.custom-claude .claude-alt]", got.Paths)
    }
}

func TestProviderOverrideRepo_Upsert_ReplacesPaths(t *testing.T) {
    db := NewTestDB(t)
    r := NewProviderOverrideRepo(db)
    ctx := context.Background()

    provID := providerIDByKey(t, r, ctx, "claude")

    _ = r.Upsert(ctx, domain.ProviderPathOverride{
        ProviderDefinitionID: provID, Scope: "project", Purpose: "detect",
        Paths: []string{".first"},
    })
    _ = r.Upsert(ctx, domain.ProviderPathOverride{
        ProviderDefinitionID: provID, Scope: "project", Purpose: "detect",
        Paths: []string{".second"},
    })

    all, _ := r.ListAll(ctx)
    if len(all) != 1 {
        t.Fatalf("expected 1 override after upsert, got %d", len(all))
    }
    if len(all[0].Paths) != 1 || all[0].Paths[0] != ".second" {
        t.Errorf("expected .second after upsert, got %v", all[0].Paths)
    }
}

func TestProviderOverrideRepo_Delete_ExistingRow(t *testing.T) {
    db := NewTestDB(t)
    r := NewProviderOverrideRepo(db)
    ctx := context.Background()

    provID := providerIDByKey(t, r, ctx, "claude")

    _ = r.Upsert(ctx, domain.ProviderPathOverride{
        ProviderDefinitionID: provID, Scope: "project", Purpose: "detect",
        Paths: []string{".custom"},
    })

    deleted, err := r.Delete(ctx, provID, "project", "detect")
    if err != nil {
        t.Fatalf("Delete: %v", err)
    }
    if !deleted {
        t.Error("Delete: expected true (row existed), got false")
    }

    all, _ := r.ListAll(ctx)
    if len(all) != 0 {
        t.Errorf("after delete: expected 0 overrides, got %d", len(all))
    }
}

func TestProviderOverrideRepo_Delete_NonExistent(t *testing.T) {
    db := NewTestDB(t)
    r := NewProviderOverrideRepo(db)
    ctx := context.Background()

    provID := providerIDByKey(t, r, ctx, "claude")

    deleted, err := r.Delete(ctx, provID, "project", "detect")
    if err != nil {
        t.Fatalf("Delete: %v", err)
    }
    if deleted {
        t.Error("Delete: expected false (no row), got true")
    }
}

func TestProviderOverrideRepo_GetProviderIDByKey_KnownKey(t *testing.T) {
    db := NewTestDB(t)
    r := NewProviderOverrideRepo(db)
    ctx := context.Background()

    id, err := r.GetProviderIDByKey(ctx, "claude")
    if err != nil {
        t.Fatalf("GetProviderIDByKey: %v", err)
    }
    if id == 0 {
        t.Error("expected non-zero ID for claude")
    }
}

func TestProviderOverrideRepo_GetProviderIDByKey_UnknownKey(t *testing.T) {
    db := NewTestDB(t)
    r := NewProviderOverrideRepo(db)
    ctx := context.Background()

    id, err := r.GetProviderIDByKey(ctx, "no_such_provider")
    if err != nil {
        t.Fatalf("GetProviderIDByKey returned error for unknown: %v", err)
    }
    if id != 0 {
        t.Errorf("expected 0 for unknown key, got %d", id)
    }
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd core-go && go test ./internal/repositories/... -run TestProviderOverrideRepo -v
```

Expected: compile error — `ProviderOverrideRepo` not defined.

- [ ] **Step 3: Implement ProviderOverrideRepo**

```go
// core-go/internal/repositories/provider_override_repo.go
package repositories

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"

    "github.com/astraler/skillbox/core-go/internal/domain"
)

type ProviderOverrideRepo struct {
    db *sql.DB
}

func NewProviderOverrideRepo(db *sql.DB) *ProviderOverrideRepo {
    return &ProviderOverrideRepo{db: db}
}

// GetProviderIDByKey returns the provider_definition.id for the given key,
// or 0 if the key does not exist.
func (r *ProviderOverrideRepo) GetProviderIDByKey(ctx context.Context, key string) (int64, error) {
    var id int64
    err := r.db.QueryRowContext(ctx,
        `SELECT id FROM provider_definitions WHERE key = ?`, key,
    ).Scan(&id)
    if err == sql.ErrNoRows {
        return 0, nil
    }
    return id, err
}

// ListAll returns all overrides ordered by provider_definition_id, scope, purpose.
func (r *ProviderOverrideRepo) ListAll(ctx context.Context) ([]domain.ProviderPathOverride, error) {
    rows, err := r.db.QueryContext(ctx,
        `SELECT id, provider_definition_id, scope, purpose, paths_json
           FROM provider_path_overrides
          ORDER BY provider_definition_id ASC, scope ASC, purpose ASC`)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var result []domain.ProviderPathOverride
    for rows.Next() {
        var o domain.ProviderPathOverride
        var pathsJSON string
        if err := rows.Scan(&o.ID, &o.ProviderDefinitionID, &o.Scope, &o.Purpose, &pathsJSON); err != nil {
            return nil, err
        }
        if err := json.Unmarshal([]byte(pathsJSON), &o.Paths); err != nil {
            return nil, err
        }
        result = append(result, o)
    }
    return result, rows.Err()
}

// Upsert inserts or replaces the override for (provider_definition_id, scope, purpose).
func (r *ProviderOverrideRepo) Upsert(ctx context.Context, o domain.ProviderPathOverride) error {
    pathsJSON, err := json.Marshal(o.Paths)
    if err != nil {
        return err
    }
    now := time.Now().UTC().Format(time.RFC3339)
    _, err = r.db.ExecContext(ctx, `
        INSERT INTO provider_path_overrides (provider_definition_id, scope, purpose, paths_json, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?)
        ON CONFLICT(provider_definition_id, scope, purpose)
        DO UPDATE SET paths_json = excluded.paths_json, updated_at = excluded.updated_at
    `, o.ProviderDefinitionID, o.Scope, o.Purpose, string(pathsJSON), now, now)
    return err
}

// Delete removes the override for (providerDefinitionID, scope, purpose).
// Returns true if a row was deleted, false if none existed.
func (r *ProviderOverrideRepo) Delete(ctx context.Context, providerDefinitionID int64, scope, purpose string) (bool, error) {
    res, err := r.db.ExecContext(ctx, `
        DELETE FROM provider_path_overrides
         WHERE provider_definition_id = ? AND scope = ? AND purpose = ?
    `, providerDefinitionID, scope, purpose)
    if err != nil {
        return false, err
    }
    n, err := res.RowsAffected()
    return n > 0, err
}
```

- [ ] **Step 4: Run tests**

```bash
cd core-go && go test ./internal/repositories/... -run TestProviderOverrideRepo -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add core-go/internal/repositories/provider_override_repo.go \
        core-go/internal/repositories/provider_override_repo_test.go \
        core-go/internal/domain/provider.go
git commit -m "Add ProviderPathOverride domain type + ProviderOverrideRepo (PR-2A)"
```

---

## Task 4: Update service interfaces + ProviderRegistryService

**Files:**
- Modify: `core-go/internal/services/interfaces.go`
- Modify: `core-go/internal/services/provider_registry_service.go`
- Modify: `core-go/internal/services/provider_registry_service_test.go`

- [ ] **Step 1: Write failing tests in provider_registry_service_test.go**

Replace the entire file content:

```go
package services

import (
    "context"
    "errors"
    "testing"

    "github.com/astraler/skillbox/core-go/internal/domain"
)

// -- mocks --

type mockProviderRegistryRepo struct {
    entries []domain.ProviderRegistryEntry
    err     error
}

func (m *mockProviderRegistryRepo) ListAll(_ context.Context) ([]domain.ProviderRegistryEntry, error) {
    return m.entries, m.err
}

func (m *mockProviderRegistryRepo) GetByKey(_ context.Context, key string) (*domain.ProviderDefinition, error) {
    for _, e := range m.entries {
        if e.Definition.Key == key {
            d := e.Definition
            return &d, nil
        }
    }
    return nil, nil
}

type mockProviderOverrideRepo struct {
    overrides  []domain.ProviderPathOverride
    listErr    error
    upsertErr  error
    deleteRet  bool
    deleteErr  error
    idByKey    map[string]int64
    idByKeyErr error
}

func (m *mockProviderOverrideRepo) ListAll(_ context.Context) ([]domain.ProviderPathOverride, error) {
    return m.overrides, m.listErr
}

func (m *mockProviderOverrideRepo) Upsert(_ context.Context, o domain.ProviderPathOverride) error {
    m.overrides = append(m.overrides, o)
    return m.upsertErr
}

func (m *mockProviderOverrideRepo) Delete(_ context.Context, _ int64, _, _ string) (bool, error) {
    return m.deleteRet, m.deleteErr
}

func (m *mockProviderOverrideRepo) GetProviderIDByKey(_ context.Context, key string) (int64, error) {
    if m.idByKeyErr != nil {
        return 0, m.idByKeyErr
    }
    if m.idByKey != nil {
        return m.idByKey[key], nil
    }
    return 0, nil
}

func makeTestEntry(key, status string) domain.ProviderRegistryEntry {
    iconKey := key
    return domain.ProviderRegistryEntry{
        Definition: domain.ProviderDefinition{
            ID:           1,
            Key:          key,
            DisplayName:  key,
            ProviderType: key,
            IconKey:      &iconKey,
            Status:       domain.ProviderStatus(status),
        },
        Candidates: []domain.ProviderPathCandidate{
            {RelativePath: "." + key, Scope: "project", Purpose: "detect", Priority: 10, VerificationStatus: "assumed", Source: "builtin"},
        },
    }
}

func makeSvc(entries []domain.ProviderRegistryEntry, overrides []domain.ProviderPathOverride) *ProviderRegistryService {
    repo := &mockProviderRegistryRepo{entries: entries}
    overrideRepo := &mockProviderOverrideRepo{
        overrides: overrides,
        idByKey: func() map[string]int64 {
            m := make(map[string]int64)
            for _, e := range entries {
                m[e.Definition.Key] = e.Definition.ID
            }
            return m
        }(),
    }
    return NewProviderRegistryService(repo, overrideRepo)
}

// -- List tests --

func TestProviderRegistryService_List_ReturnsEntries(t *testing.T) {
    entries := []domain.ProviderRegistryEntry{
        makeTestEntry("generic_agents", "supported"),
        makeTestEntry("claude", "experimental"),
    }
    svc := makeSvc(entries, nil)

    got, err := svc.List(context.Background())
    if err != nil {
        t.Fatalf("List: %v", err)
    }
    if len(got) != 2 {
        t.Errorf("len: got %d want 2", len(got))
    }
    if got[0].Definition.Key != "generic_agents" {
        t.Errorf("first key: got %q want generic_agents", got[0].Definition.Key)
    }
}

func TestProviderRegistryService_List_RepoErrorWrapped(t *testing.T) {
    repo := &mockProviderRegistryRepo{err: errors.New("db gone")}
    svc := NewProviderRegistryService(repo, &mockProviderOverrideRepo{})

    _, err := svc.List(context.Background())
    if err == nil {
        t.Fatal("expected error, got nil")
    }
    var appErr *domain.AppError
    if !errors.As(err, &appErr) {
        t.Fatalf("expected *domain.AppError, got %T: %v", err, err)
    }
    if appErr.Code != domain.CodeDatabase {
        t.Errorf("error code: got %q want database_error", appErr.Code)
    }
}

func TestProviderRegistryService_List_EmptyIsNotNil(t *testing.T) {
    svc := makeSvc([]domain.ProviderRegistryEntry{}, nil)

    got, err := svc.List(context.Background())
    if err != nil {
        t.Fatalf("List: %v", err)
    }
    if got == nil {
        t.Error("expected non-nil empty slice, got nil")
    }
}

func TestProviderRegistryService_List_MergesOverride(t *testing.T) {
    entries := []domain.ProviderRegistryEntry{makeTestEntry("claude", "experimental")}
    overrides := []domain.ProviderPathOverride{
        {ProviderDefinitionID: 1, Scope: "project", Purpose: "detect", Paths: []string{".custom-claude"}},
    }
    svc := makeSvc(entries, overrides)

    got, err := svc.List(context.Background())
    if err != nil {
        t.Fatalf("List: %v", err)
    }
    if len(got) != 1 {
        t.Fatalf("entry count: got %d want 1", len(got))
    }
    cands := got[0].Candidates
    if len(cands) != 1 {
        t.Fatalf("candidate count: got %d want 1", len(cands))
    }
    if cands[0].RelativePath != ".custom-claude" {
        t.Errorf("RelativePath: got %q want .custom-claude", cands[0].RelativePath)
    }
    if cands[0].Source != "override" {
        t.Errorf("Source: got %q want override", cands[0].Source)
    }
}

func TestProviderRegistryService_List_BuiltinPreservedWhenNoOverride(t *testing.T) {
    entries := []domain.ProviderRegistryEntry{makeTestEntry("claude", "experimental")}
    svc := makeSvc(entries, nil)

    got, _ := svc.List(context.Background())
    if len(got[0].Candidates) != 1 || got[0].Candidates[0].Source != "builtin" {
        t.Errorf("expected builtin candidate, got %v", got[0].Candidates)
    }
}

// -- UpdatePaths tests --

func TestProviderRegistryService_UpdatePaths_Success(t *testing.T) {
    entries := []domain.ProviderRegistryEntry{makeTestEntry("claude", "experimental")}
    overrideRepo := &mockProviderOverrideRepo{
        idByKey: map[string]int64{"claude": 1},
    }
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{entries: entries}, overrideRepo)

    err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{".custom"})
    if err != nil {
        t.Fatalf("UpdatePaths: %v", err)
    }
}

func TestProviderRegistryService_UpdatePaths_UnknownProvider(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{}}
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    err := svc.UpdatePaths(context.Background(), "no_such", "project", "detect", []string{".path"})
    if err == nil {
        t.Fatal("expected error for unknown provider")
    }
    var ae *domain.AppError
    if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
        t.Errorf("expected validation_error, got %v", err)
    }
}

func TestProviderRegistryService_UpdatePaths_InvalidScope(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    err := svc.UpdatePaths(context.Background(), "claude", "invalid_scope", "detect", []string{".path"})
    if err == nil {
        t.Fatal("expected error for invalid scope")
    }
    var ae *domain.AppError
    if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
        t.Errorf("expected validation_error, got %v", err)
    }
}

func TestProviderRegistryService_UpdatePaths_ProjectPathWithDotDot(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{"../escape"})
    if err == nil {
        t.Fatal("expected error for path with ..")
    }
    var ae *domain.AppError
    if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
        t.Errorf("expected validation_error, got %v", err)
    }
}

func TestProviderRegistryService_UpdatePaths_ProjectPathAbsolute(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{"/absolute"})
    if err == nil {
        t.Fatal("expected error for absolute project path")
    }
    var ae *domain.AppError
    if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
        t.Errorf("expected validation_error, got %v", err)
    }
}

func TestProviderRegistryService_UpdatePaths_GlobalPathNoTilde(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    err := svc.UpdatePaths(context.Background(), "claude", "global", "skills", []string{"relative/path"})
    if err == nil {
        t.Fatal("expected error for global path without / or ~/")
    }
    var ae *domain.AppError
    if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
        t.Errorf("expected validation_error, got %v", err)
    }
}

func TestProviderRegistryService_UpdatePaths_EmptyPaths(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{"claude": 1}}
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    err := svc.UpdatePaths(context.Background(), "claude", "project", "detect", []string{})
    if err == nil {
        t.Fatal("expected error for empty paths")
    }
    var ae *domain.AppError
    if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
        t.Errorf("expected validation_error, got %v", err)
    }
}

// -- ResetPaths tests --

func TestProviderRegistryService_ResetPaths_ExistingOverride(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{
        idByKey:   map[string]int64{"claude": 1},
        deleteRet: true,
    }
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    reset, err := svc.ResetPaths(context.Background(), "claude", "project", "detect")
    if err != nil {
        t.Fatalf("ResetPaths: %v", err)
    }
    if !reset {
        t.Error("expected reset=true when override existed")
    }
}

func TestProviderRegistryService_ResetPaths_NoOverride(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{
        idByKey:   map[string]int64{"claude": 1},
        deleteRet: false,
    }
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    reset, err := svc.ResetPaths(context.Background(), "claude", "project", "detect")
    if err != nil {
        t.Fatalf("ResetPaths: %v", err)
    }
    if reset {
        t.Error("expected reset=false when no override existed")
    }
}

func TestProviderRegistryService_ResetPaths_UnknownProvider(t *testing.T) {
    overrideRepo := &mockProviderOverrideRepo{idByKey: map[string]int64{}}
    svc := NewProviderRegistryService(&mockProviderRegistryRepo{}, overrideRepo)

    _, err := svc.ResetPaths(context.Background(), "no_such", "project", "detect")
    if err == nil {
        t.Fatal("expected error for unknown provider")
    }
    var ae *domain.AppError
    if !errors.As(err, &ae) || ae.Code != domain.CodeValidation {
        t.Errorf("expected validation_error, got %v", err)
    }
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd core-go && go test ./internal/services/... -run TestProviderRegistryService -v 2>&1 | head -30
```

Expected: compile errors — `NewProviderRegistryService` signature mismatch, methods missing.

- [ ] **Step 3: Update services/interfaces.go**

Add the `ProviderOverrideRepo` interface and extend `ProviderRegistryRepo` with `GetByKey`:

```go
// ProviderRegistryRepo lists all provider definitions with their path candidates
// and looks up individual providers by key.
// *repositories.ProviderDefinitionRepo satisfies this interface.
type ProviderRegistryRepo interface {
    ListAll(ctx context.Context) ([]domain.ProviderRegistryEntry, error)
    GetByKey(ctx context.Context, key string) (*domain.ProviderDefinition, error)
}

// ProviderOverrideRepo stores user path overrides for built-in providers.
// *repositories.ProviderOverrideRepo satisfies this interface.
type ProviderOverrideRepo interface {
    ListAll(ctx context.Context) ([]domain.ProviderPathOverride, error)
    Upsert(ctx context.Context, o domain.ProviderPathOverride) error
    Delete(ctx context.Context, providerDefinitionID int64, scope, purpose string) (bool, error)
    GetProviderIDByKey(ctx context.Context, key string) (int64, error)
}
```

Note: Replace the existing `ProviderRegistryRepo` definition and add `ProviderOverrideRepo` below it.

- [ ] **Step 4: Rewrite provider_registry_service.go**

```go
package services

import (
    "context"
    "fmt"
    "path/filepath"
    "strings"

    "github.com/astraler/skillbox/core-go/internal/domain"
)

var validScopes   = map[string]bool{"project": true, "global": true}
var validPurposes = map[string]bool{"detect": true, "skills": true, "config": true, "commands": true}

// ProviderRegistryService returns the provider registry with override support.
type ProviderRegistryService struct {
    repo         ProviderRegistryRepo
    overrideRepo ProviderOverrideRepo
}

func NewProviderRegistryService(repo ProviderRegistryRepo, overrideRepo ProviderOverrideRepo) *ProviderRegistryService {
    return &ProviderRegistryService{repo: repo, overrideRepo: overrideRepo}
}

func (s *ProviderRegistryService) List(ctx context.Context) ([]domain.ProviderRegistryEntry, error) {
    entries, err := s.repo.ListAll(ctx)
    if err != nil {
        return nil, domain.NewDatabaseError("Could not load provider registry", err.Error())
    }

    overrides, err := s.overrideRepo.ListAll(ctx)
    if err != nil {
        return nil, domain.NewDatabaseError("Could not load provider path overrides", err.Error())
    }

    if len(overrides) > 0 {
        entries = mergeOverrides(entries, overrides)
    }

    return entries, nil
}

// mergeOverrides replaces builtin candidates for each (providerID, scope, purpose)
// slot that has an override. Override candidates carry Source="override".
func mergeOverrides(entries []domain.ProviderRegistryEntry, overrides []domain.ProviderPathOverride) []domain.ProviderRegistryEntry {
    type slot struct {
        provID  int64
        scope   string
        purpose string
    }
    overrideMap := make(map[slot][]string, len(overrides))
    for _, o := range overrides {
        overrideMap[slot{o.ProviderDefinitionID, o.Scope, o.Purpose}] = o.Paths
    }

    result := make([]domain.ProviderRegistryEntry, len(entries))
    for i, e := range entries {
        newCands := make([]domain.ProviderPathCandidate, 0, len(e.Candidates))
        // Track which (scope, purpose) slots have been overridden for this provider.
        overriddenSlots := map[string]bool{}
        for _, o := range overrides {
            if o.ProviderDefinitionID == e.Definition.ID {
                key := o.Scope + ":" + o.Purpose
                if !overriddenSlots[key] {
                    overriddenSlots[key] = true
                    for _, p := range o.Paths {
                        newCands = append(newCands, domain.ProviderPathCandidate{
                            ProviderDefinitionID: e.Definition.ID,
                            RelativePath:         p,
                            Scope:                o.Scope,
                            Purpose:              o.Purpose,
                            Priority:             10,
                            VerificationStatus:   "assumed",
                            Source:               "override",
                        })
                    }
                }
            }
        }
        // Add builtin candidates for non-overridden slots.
        for _, c := range e.Candidates {
            key := c.Scope + ":" + c.Purpose
            if !overriddenSlots[key] {
                c.Source = "builtin"
                newCands = append(newCands, c)
            }
        }
        result[i] = domain.ProviderRegistryEntry{
            Definition: e.Definition,
            Candidates: newCands,
        }
        _ = overrideMap // used above via overriddenSlots
    }
    return result
}

// UpdatePaths validates and persists a path override for the given (providerKey, scope, purpose).
func (s *ProviderRegistryService) UpdatePaths(ctx context.Context, providerKey, scope, purpose string, paths []string) error {
    if providerKey == "" {
        return domain.NewValidationError("Provider key is required", "providerKey must not be empty")
    }
    if !validScopes[scope] {
        return domain.NewValidationError("Invalid scope", fmt.Sprintf("scope must be 'project' or 'global', got %q", scope))
    }
    if !validPurposes[purpose] {
        return domain.NewValidationError("Invalid purpose", fmt.Sprintf("purpose must be one of detect/skills/config/commands, got %q", purpose))
    }
    if len(paths) == 0 {
        return domain.NewValidationError("Paths must not be empty", "provide at least one path, or use resetPaths to restore defaults")
    }
    for _, p := range paths {
        if err := validatePath(p, scope); err != nil {
            return err
        }
    }

    provID, err := s.overrideRepo.GetProviderIDByKey(ctx, providerKey)
    if err != nil {
        return domain.NewDatabaseError("Could not look up provider", err.Error())
    }
    if provID == 0 {
        return domain.NewValidationError("Unknown provider", fmt.Sprintf("provider key %q not found", providerKey))
    }

    if err := s.overrideRepo.Upsert(ctx, domain.ProviderPathOverride{
        ProviderDefinitionID: provID,
        Scope:                scope,
        Purpose:              purpose,
        Paths:                paths,
    }); err != nil {
        return domain.NewDatabaseError("Could not save path override", err.Error())
    }
    return nil
}

// ResetPaths removes the user override for (providerKey, scope, purpose), restoring builtin defaults.
// Returns true if an override was removed, false if none existed.
func (s *ProviderRegistryService) ResetPaths(ctx context.Context, providerKey, scope, purpose string) (bool, error) {
    if providerKey == "" {
        return false, domain.NewValidationError("Provider key is required", "providerKey must not be empty")
    }
    if !validScopes[scope] {
        return false, domain.NewValidationError("Invalid scope", fmt.Sprintf("scope must be 'project' or 'global', got %q", scope))
    }
    if !validPurposes[purpose] {
        return false, domain.NewValidationError("Invalid purpose", fmt.Sprintf("purpose must be one of detect/skills/config/commands, got %q", purpose))
    }

    provID, err := s.overrideRepo.GetProviderIDByKey(ctx, providerKey)
    if err != nil {
        return false, domain.NewDatabaseError("Could not look up provider", err.Error())
    }
    if provID == 0 {
        return false, domain.NewValidationError("Unknown provider", fmt.Sprintf("provider key %q not found", providerKey))
    }

    deleted, err := s.overrideRepo.Delete(ctx, provID, scope, purpose)
    if err != nil {
        return false, domain.NewDatabaseError("Could not reset path override", err.Error())
    }
    return deleted, nil
}

func validatePath(p, scope string) error {
    if p == "" {
        return domain.NewValidationError("Empty path", "path must not be empty")
    }
    switch scope {
    case "project":
        if strings.HasPrefix(p, "/") {
            return domain.NewValidationError("Invalid project path", fmt.Sprintf("project path must be relative, got absolute path %q", p))
        }
        clean := filepath.Clean(p)
        if strings.HasPrefix(clean, "..") {
            return domain.NewValidationError("Invalid project path", fmt.Sprintf("project path must not escape via .., got %q", p))
        }
    case "global":
        if !strings.HasPrefix(p, "/") && !strings.HasPrefix(p, "~/") {
            return domain.NewValidationError("Invalid global path", fmt.Sprintf("global path must start with / or ~/, got %q", p))
        }
    }
    return nil
}
```

- [ ] **Step 5: Run tests**

```bash
cd core-go && go test ./internal/services/... -run TestProviderRegistryService -v
```

Expected: all PASS.

- [ ] **Step 6: Run full test suite**

```bash
cd core-go && go test ./...
```

Expected: all PASS.

- [ ] **Step 7: Commit**

```bash
git add core-go/internal/services/interfaces.go \
        core-go/internal/services/provider_registry_service.go \
        core-go/internal/services/provider_registry_service_test.go
git commit -m "Extend ProviderRegistryService with UpdatePaths/ResetPaths + override merge (PR-2A)"
```

---

## Task 5: RPC Handlers for provider.updatePaths and provider.resetPaths

**Files:**
- Modify: `core-go/internal/rpc/handlers/provider_list.go`
- Create: `core-go/internal/rpc/handlers/provider_update_paths.go`
- Create: `core-go/internal/rpc/handlers/provider_reset_paths.go`
- Create: `core-go/internal/rpc/handlers/provider_update_reset_handler_test.go`
- Modify: `core-go/internal/rpc/handlers/provider_list_handler_test.go`

- [ ] **Step 1: Update provider_list.go to use candidate.Source**

Change the line `Source: "builtin",` in the candidates loop to use the domain field:

```go
// In NewProviderListHandler, change the candidates loop:
for j, c := range e.Candidates {
    source := c.Source
    if source == "" {
        source = "builtin" // backward compat for zero-value
    }
    candidates[j] = providerListPathCandidate{
        RelativePath:       c.RelativePath,
        Scope:              c.Scope,
        Purpose:            c.Purpose,
        Priority:           c.Priority,
        Source:             source,
        VerificationStatus: c.VerificationStatus,
    }
}
```

- [ ] **Step 2: Write failing handler tests**

```go
// core-go/internal/rpc/handlers/provider_update_reset_handler_test.go
package handlers_test

import (
    "context"
    "errors"
    "testing"

    "github.com/creachadair/jrpc2"
    "github.com/creachadair/jrpc2/handler"

    "github.com/astraler/skillbox/core-go/internal/domain"
    "github.com/astraler/skillbox/core-go/internal/rpc/handlers"
)

// -- stubs --

type stubProviderUpdateResetSvc struct {
    updateErr  error
    resetVal   bool
    resetErr   error
}

func (s *stubProviderUpdateResetSvc) UpdatePaths(_ context.Context, _, _, _ string, _ []string) error {
    return s.updateErr
}

func (s *stubProviderUpdateResetSvc) ResetPaths(_ context.Context, _, _, _ string) (bool, error) {
    return s.resetVal, s.resetErr
}

// -- updatePaths tests --

func TestProviderUpdatePathsHandler_Success(t *testing.T) {
    svc := &stubProviderUpdateResetSvc{}
    cli := startServer(t, handler.Map{"provider.updatePaths": handlers.NewProviderUpdatePathsHandler(svc)})

    var resp struct{ Updated bool `json:"updated"` }
    err := cli.CallResult(context.Background(), "provider.updatePaths", map[string]interface{}{
        "providerKey": "claude",
        "scope":       "project",
        "purpose":     "detect",
        "paths":       []string{".custom"},
    }, &resp)
    if err != nil {
        t.Fatalf("provider.updatePaths: %v", err)
    }
    if !resp.Updated {
        t.Error("expected updated=true on success")
    }
}

func TestProviderUpdatePathsHandler_MissingKey(t *testing.T) {
    svc := &stubProviderUpdateResetSvc{}
    cli := startServer(t, handler.Map{"provider.updatePaths": handlers.NewProviderUpdatePathsHandler(svc)})

    err := cli.CallResult(context.Background(), "provider.updatePaths", map[string]interface{}{
        "scope": "project", "purpose": "detect", "paths": []string{".custom"},
    }, nil)
    if err == nil {
        t.Fatal("expected error for missing providerKey")
    }
    var rpcErr *jrpc2.Error
    if !errors.As(err, &rpcErr) {
        t.Fatalf("expected *jrpc2.Error, got %T", err)
    }
}

func TestProviderUpdatePathsHandler_ServiceValidationError(t *testing.T) {
    svc := &stubProviderUpdateResetSvc{updateErr: domain.NewValidationError("Unknown provider", "key not found")}
    cli := startServer(t, handler.Map{"provider.updatePaths": handlers.NewProviderUpdatePathsHandler(svc)})

    err := cli.CallResult(context.Background(), "provider.updatePaths", map[string]interface{}{
        "providerKey": "no_such",
        "scope":       "project",
        "purpose":     "detect",
        "paths":       []string{".custom"},
    }, nil)
    if err == nil {
        t.Fatal("expected error")
    }
    we := extractWireError(t, err, jrpc2.Code(1001))
    if we.ae.Code != domain.CodeValidation {
        t.Errorf("code: got %q want validation_error", we.ae.Code)
    }
}

// -- resetPaths tests --

func TestProviderResetPathsHandler_ExistingOverride(t *testing.T) {
    svc := &stubProviderUpdateResetSvc{resetVal: true}
    cli := startServer(t, handler.Map{"provider.resetPaths": handlers.NewProviderResetPathsHandler(svc)})

    var resp struct{ Reset bool `json:"reset"` }
    err := cli.CallResult(context.Background(), "provider.resetPaths", map[string]interface{}{
        "providerKey": "claude",
        "scope":       "project",
        "purpose":     "detect",
    }, &resp)
    if err != nil {
        t.Fatalf("provider.resetPaths: %v", err)
    }
    if !resp.Reset {
        t.Error("expected reset=true")
    }
}

func TestProviderResetPathsHandler_NoOverride(t *testing.T) {
    svc := &stubProviderUpdateResetSvc{resetVal: false}
    cli := startServer(t, handler.Map{"provider.resetPaths": handlers.NewProviderResetPathsHandler(svc)})

    var resp struct{ Reset bool `json:"reset"` }
    err := cli.CallResult(context.Background(), "provider.resetPaths", map[string]interface{}{
        "providerKey": "claude",
        "scope":       "project",
        "purpose":     "detect",
    }, &resp)
    if err != nil {
        t.Fatalf("provider.resetPaths: %v", err)
    }
    if resp.Reset {
        t.Error("expected reset=false when no override existed")
    }
}

func TestProviderResetPathsHandler_UnknownProvider(t *testing.T) {
    svc := &stubProviderUpdateResetSvc{resetErr: domain.NewValidationError("Unknown provider", "key not found")}
    cli := startServer(t, handler.Map{"provider.resetPaths": handlers.NewProviderResetPathsHandler(svc)})

    err := cli.CallResult(context.Background(), "provider.resetPaths", map[string]interface{}{
        "providerKey": "no_such",
        "scope":       "project",
        "purpose":     "detect",
    }, nil)
    if err == nil {
        t.Fatal("expected error for unknown provider")
    }
    we := extractWireError(t, err, jrpc2.Code(1001))
    if we.ae.Code != domain.CodeValidation {
        t.Errorf("code: got %q want validation_error", we.ae.Code)
    }
}
```

- [ ] **Step 3: Run tests to confirm they fail**

```bash
cd core-go && go test ./internal/rpc/handlers/... -run "TestProviderUpdatePaths|TestProviderResetPaths" -v 2>&1 | head -20
```

Expected: compile error — handlers not defined.

- [ ] **Step 4: Implement provider_update_paths.go**

```go
// core-go/internal/rpc/handlers/provider_update_paths.go
package handlers

import (
    "context"

    "github.com/creachadair/jrpc2"
    "github.com/creachadair/jrpc2/handler"
)

type providerPathsService interface {
    UpdatePaths(ctx context.Context, providerKey, scope, purpose string, paths []string) error
    ResetPaths(ctx context.Context, providerKey, scope, purpose string) (bool, error)
}

type providerUpdatePathsRequest struct {
    ProviderKey string   `json:"providerKey"`
    Scope       string   `json:"scope"`
    Purpose     string   `json:"purpose"`
    Paths       []string `json:"paths"`
}

type providerUpdatePathsResponse struct {
    Updated bool `json:"updated"`
}

func NewProviderUpdatePathsHandler(svc providerPathsService) jrpc2.Handler {
    return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
        var p providerUpdatePathsRequest
        if err := req.UnmarshalParams(&p); err != nil {
            return nil, err
        }
        if p.ProviderKey == "" {
            return nil, wrapError(newValidationError("providerKey is required"))
        }
        if err := svc.UpdatePaths(ctx, p.ProviderKey, p.Scope, p.Purpose, p.Paths); err != nil {
            return nil, wrapError(err)
        }
        return providerUpdatePathsResponse{Updated: true}, nil
    })
}
```

- [ ] **Step 5: Implement provider_reset_paths.go**

```go
// core-go/internal/rpc/handlers/provider_reset_paths.go
package handlers

import (
    "context"

    "github.com/creachadair/jrpc2"
    "github.com/creachadair/jrpc2/handler"
)

type providerResetPathsRequest struct {
    ProviderKey string `json:"providerKey"`
    Scope       string `json:"scope"`
    Purpose     string `json:"purpose"`
}

type providerResetPathsResponse struct {
    Reset bool `json:"reset"`
}

func NewProviderResetPathsHandler(svc providerPathsService) jrpc2.Handler {
    return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
        var p providerResetPathsRequest
        if err := req.UnmarshalParams(&p); err != nil {
            return nil, err
        }
        if p.ProviderKey == "" {
            return nil, wrapError(newValidationError("providerKey is required"))
        }
        reset, err := svc.ResetPaths(ctx, p.ProviderKey, p.Scope, p.Purpose)
        if err != nil {
            return nil, wrapError(err)
        }
        return providerResetPathsResponse{Reset: reset}, nil
    })
}
```

- [ ] **Step 6: Add `newValidationError` helper to helpers.go**

Check `core-go/internal/rpc/handlers/helpers.go`. If it only has `Ping`, add to it:

```go
// In helpers.go, add:
func newValidationError(detail string) error {
    return domain.NewValidationError(detail, detail)
}
```

(Import `github.com/astraler/skillbox/core-go/internal/domain` at top of helpers.go)

- [ ] **Step 7: Run handler tests**

```bash
cd core-go && go test ./internal/rpc/handlers/... -v
```

Expected: all PASS including new handler tests.

- [ ] **Step 8: Commit**

```bash
git add core-go/internal/rpc/handlers/provider_list.go \
        core-go/internal/rpc/handlers/provider_update_paths.go \
        core-go/internal/rpc/handlers/provider_reset_paths.go \
        core-go/internal/rpc/handlers/provider_update_reset_handler_test.go \
        core-go/internal/rpc/handlers/helpers.go \
        core-go/internal/rpc/handlers/provider_list_handler_test.go
git commit -m "Add provider.updatePaths and provider.resetPaths RPC handlers (PR-2A)"
```

---

## Task 6: Wire + contracts + main update

**Files:**
- Modify: `core-go/internal/app/wire.go`
- Modify: `core-go/cmd/skillbox-core/main.go`
- Create: `shared/api-contracts/methods/provider.updatePaths.json`
- Create: `shared/api-contracts/methods/provider.resetPaths.json`
- Modify: `shared/api-contracts/index.json`

- [ ] **Step 1: Create provider.updatePaths.json**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ProviderUpdatePathsMethod",
  "description": "Contract for provider.updatePaths command. Stores user path overrides for a provider (scope+purpose) slot. Does not affect scan or install behavior — configuration metadata only (behavior integration is a later slice).",
  "oneOf": [
    { "$ref": "#/definitions/ProviderUpdatePathsRequest" },
    { "$ref": "#/definitions/ProviderUpdatePathsResponse" }
  ],
  "definitions": {
    "ProviderUpdatePathsRequest": {
      "title": "ProviderUpdatePathsRequest",
      "description": "Params for provider.updatePaths.",
      "type": "object",
      "properties": {
        "providerKey": {
          "type": "string",
          "description": "Stable provider key (e.g. claude, generic_agents)"
        },
        "scope": {
          "type": "string",
          "enum": ["project", "global"],
          "description": "Whether override applies to project or global paths"
        },
        "purpose": {
          "type": "string",
          "enum": ["detect", "skills", "config", "commands"],
          "description": "Role of the path slot being overridden"
        },
        "paths": {
          "type": "array",
          "items": { "type": "string" },
          "minItems": 1,
          "description": "Override paths. Project paths must be relative (no ..). Global paths must start with / or ~/."
        }
      },
      "required": ["providerKey", "scope", "purpose", "paths"],
      "additionalProperties": false
    },
    "ProviderUpdatePathsResponse": {
      "title": "ProviderUpdatePathsResponse",
      "description": "Result of provider.updatePaths. Errors: validation_error (1001), database_error (1004).",
      "type": "object",
      "properties": {
        "updated": {
          "type": "boolean",
          "description": "True when the override was saved successfully"
        }
      },
      "required": ["updated"],
      "additionalProperties": false
    }
  }
}
```

- [ ] **Step 2: Create provider.resetPaths.json**

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ProviderResetPathsMethod",
  "description": "Contract for provider.resetPaths command. Removes user path override for a provider (scope+purpose) slot, restoring built-in defaults.",
  "oneOf": [
    { "$ref": "#/definitions/ProviderResetPathsRequest" },
    { "$ref": "#/definitions/ProviderResetPathsResponse" }
  ],
  "definitions": {
    "ProviderResetPathsRequest": {
      "title": "ProviderResetPathsRequest",
      "description": "Params for provider.resetPaths.",
      "type": "object",
      "properties": {
        "providerKey": {
          "type": "string",
          "description": "Stable provider key"
        },
        "scope": {
          "type": "string",
          "enum": ["project", "global"],
          "description": "Scope of the slot to reset"
        },
        "purpose": {
          "type": "string",
          "enum": ["detect", "skills", "config", "commands"],
          "description": "Purpose of the slot to reset"
        }
      },
      "required": ["providerKey", "scope", "purpose"],
      "additionalProperties": false
    },
    "ProviderResetPathsResponse": {
      "title": "ProviderResetPathsResponse",
      "description": "Result of provider.resetPaths. Errors: validation_error (1001), database_error (1004).",
      "type": "object",
      "properties": {
        "reset": {
          "type": "boolean",
          "description": "True if an override existed and was removed; false if no override was stored"
        }
      },
      "required": ["reset"],
      "additionalProperties": false
    }
  }
}
```

- [ ] **Step 3: Add to shared/api-contracts/index.json**

Add these two lines to the `"schemas"` array (before the closing `]`):

```json
    { "input": "methods/provider.updatePaths.json", "output": "methods/provider-update-paths.ts" },
    { "input": "methods/provider.resetPaths.json", "output": "methods/provider-reset-paths.ts" }
```

- [ ] **Step 4: Update wire.go**

In `core-go/internal/app/wire.go`, add the override repo to `New` signature and register handlers:

```go
func New(
    hostSvc *services.SkillHostService,
    libSvc *services.SkillLibraryService,
    settingsSvc *services.SettingsService,
    runner *operations.Runner,
    projectSvc *services.ProjectService,
    dashboardSvc *services.DashboardService,
    globalSvc *services.GlobalSkillsService,
    providerRegistrySvc *services.ProviderRegistryService,
) *App {
    a := &App{
        methods: handler.Map{
            // ... existing entries ...
            "provider.list":        rpchandlers.NewProviderListHandler(providerRegistrySvc),
            "provider.updatePaths": rpchandlers.NewProviderUpdatePathsHandler(providerRegistrySvc),
            "provider.resetPaths":  rpchandlers.NewProviderResetPathsHandler(providerRegistrySvc),
        },
    }
    return a
}
```

Note: `*services.ProviderRegistryService` satisfies `providerPathsService` interface since it has `UpdatePaths` and `ResetPaths`.

- [ ] **Step 5: Update main.go**

In `core-go/cmd/skillbox-core/main.go`:

1. Create `overrideRepo` after `pdRepo`:

```go
overrideRepo := repositories.NewProviderOverrideRepo(db)
```

2. Update `NewProviderRegistryService` call:

```go
providerRegistrySvc := services.NewProviderRegistryService(pdRepo, overrideRepo)
```

3. Add `"provider.updatePaths"` and `"provider.resetPaths"` to capabilities list in `server.ready` notification.

- [ ] **Step 6: Build to check compile**

```bash
cd core-go && go build ./...
```

Expected: success (no compile errors).

- [ ] **Step 7: Run all Go tests**

```bash
cd core-go && go test ./...
```

Expected: all PASS.

- [ ] **Step 8: Generate contracts**

```bash
cd apps/desktop && pnpm generate:contracts
```

Expected: generates `shared/generated/methods/provider-update-paths.ts` and `shared/generated/methods/provider-reset-paths.ts`.

- [ ] **Step 9: Run contract drift check**

```bash
cd apps/desktop && pnpm check:contracts-drift
```

Expected: no drift.

- [ ] **Step 10: Commit**

```bash
git add core-go/internal/app/wire.go \
        core-go/cmd/skillbox-core/main.go \
        shared/api-contracts/methods/provider.updatePaths.json \
        shared/api-contracts/methods/provider.resetPaths.json \
        shared/api-contracts/index.json \
        shared/generated/
git commit -m "Wire updatePaths/resetPaths handlers + contracts + generated types (PR-2A)"
```

---

## Task 7: Renderer — methods + hooks

**Files:**
- Modify: `apps/desktop/renderer/src/lib/core-client/methods.ts`
- Create: `apps/desktop/renderer/src/features/providers/use-update-provider-paths.ts`
- Create: `apps/desktop/renderer/src/features/providers/use-reset-provider-paths.ts`
- Create: `apps/desktop/renderer/src/features/providers/__tests__/use-update-provider-paths.test.tsx`
- Create: `apps/desktop/renderer/src/features/providers/__tests__/use-reset-provider-paths.test.tsx`

- [ ] **Step 1: Add methods to methods.ts**

Import new types at top of `methods.ts`:

```typescript
import type {
  // ... existing imports ...
  ProviderUpdatePathsRequest,
  ProviderUpdatePathsResponse,
  ProviderResetPathsRequest,
  ProviderResetPathsResponse,
} from "@contracts/index.js";
```

Add to `methods` object:

```typescript
  updateProviderPaths: (req: ProviderUpdatePathsRequest) =>
    invoke<ProviderUpdatePathsResponse>("provider.updatePaths", req),

  resetProviderPaths: (req: ProviderResetPathsRequest) =>
    invoke<ProviderResetPathsResponse>("provider.resetPaths", req),
```

- [ ] **Step 2: Write failing hook tests**

```typescript
// apps/desktop/renderer/src/features/providers/__tests__/use-update-provider-paths.test.tsx
// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { updateProviderPaths: vi.fn() },
}));

import { useUpdateProviderPaths } from "../use-update-provider-paths.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockUpdateProviderPaths = methods.updateProviderPaths as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return {
    client,
    Wrapper: ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    ),
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useUpdateProviderPaths", () => {
  it("calls methods.updateProviderPaths with params", async () => {
    mockUpdateProviderPaths.mockResolvedValue({ updated: true });
    const { Wrapper, client } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useUpdateProviderPaths(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "claude", scope: "project", purpose: "detect", paths: [".custom"] });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockUpdateProviderPaths).toHaveBeenCalledWith({
      providerKey: "claude", scope: "project", purpose: "detect", paths: [".custom"],
    });
  });

  it("invalidates providers.list on success", async () => {
    mockUpdateProviderPaths.mockResolvedValue({ updated: true });
    const { Wrapper, client } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useUpdateProviderPaths(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "claude", scope: "project", purpose: "detect", paths: [".c"] });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["providers", "list"] });
  });

  it("exposes error on failure", async () => {
    mockUpdateProviderPaths.mockRejectedValue(new Error("validation_error"));
    const { Wrapper } = makeWrapper();

    const { result } = renderHook(() => useUpdateProviderPaths(), { wrapper: Wrapper });

    await act(async () => { result.current.mutate({ providerKey: "x", scope: "project", purpose: "detect", paths: [".c"] }); });
    await waitFor(() => expect(result.current.isError).toBe(true));
  });
});
```

```typescript
// apps/desktop/renderer/src/features/providers/__tests__/use-reset-provider-paths.test.tsx
// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";

vi.mock("../../../lib/core-client/methods.js", () => ({
  methods: { resetProviderPaths: vi.fn() },
}));

import { useResetProviderPaths } from "../use-reset-provider-paths.js";
import { methods } from "../../../lib/core-client/methods.js";

const mockResetProviderPaths = methods.resetProviderPaths as ReturnType<typeof vi.fn>;

function makeWrapper() {
  const client = new QueryClient({ defaultOptions: { mutations: { retry: false } } });
  return {
    client,
    Wrapper: ({ children }: { children: React.ReactNode }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    ),
  };
}

beforeEach(() => vi.clearAllMocks());

describe("useResetProviderPaths", () => {
  it("calls methods.resetProviderPaths and invalidates on success", async () => {
    mockResetProviderPaths.mockResolvedValue({ reset: true });
    const { Wrapper, client } = makeWrapper();
    vi.spyOn(client, "invalidateQueries").mockResolvedValue(undefined);

    const { result } = renderHook(() => useResetProviderPaths(), { wrapper: Wrapper });

    await act(async () => {
      result.current.mutate({ providerKey: "claude", scope: "project", purpose: "detect" });
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));

    expect(mockResetProviderPaths).toHaveBeenCalledWith({
      providerKey: "claude", scope: "project", purpose: "detect",
    });
    expect(client.invalidateQueries).toHaveBeenCalledWith({ queryKey: ["providers", "list"] });
  });
});
```

- [ ] **Step 3: Run tests to confirm they fail**

```bash
cd apps/desktop && pnpm test -- --testPathPattern="use-update-provider-paths|use-reset-provider-paths" 2>&1 | head -20
```

Expected: import error — module not found.

- [ ] **Step 4: Implement hooks**

```typescript
// apps/desktop/renderer/src/features/providers/use-update-provider-paths.ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { ProviderUpdatePathsRequest } from "@contracts/index.js";

export function useUpdateProviderPaths() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ProviderUpdatePathsRequest) => methods.updateProviderPaths(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.providers.list() });
    },
  });
}
```

```typescript
// apps/desktop/renderer/src/features/providers/use-reset-provider-paths.ts
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { methods } from "../../lib/core-client/methods.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { ProviderResetPathsRequest } from "@contracts/index.js";

export function useResetProviderPaths() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ProviderResetPathsRequest) => methods.resetProviderPaths(req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.providers.list() });
    },
  });
}
```

- [ ] **Step 5: Run hook tests**

```bash
cd apps/desktop && pnpm test -- --testPathPattern="use-update-provider-paths|use-reset-provider-paths"
```

Expected: all PASS.

- [ ] **Step 6: Run typecheck**

```bash
cd apps/desktop && pnpm typecheck
```

Expected: success.

- [ ] **Step 7: Commit**

```bash
git add apps/desktop/renderer/src/lib/core-client/methods.ts \
        apps/desktop/renderer/src/features/providers/use-update-provider-paths.ts \
        apps/desktop/renderer/src/features/providers/use-reset-provider-paths.ts \
        apps/desktop/renderer/src/features/providers/__tests__/use-update-provider-paths.test.tsx \
        apps/desktop/renderer/src/features/providers/__tests__/use-reset-provider-paths.test.tsx
git commit -m "Add useUpdateProviderPaths + useResetProviderPaths hooks (PR-2A)"
```

---

## Task 8: Provider paths editor dialog

**Files:**
- Create: `apps/desktop/renderer/src/features/providers/provider-paths-editor.tsx`
- Create: `apps/desktop/renderer/src/features/providers/__tests__/provider-paths-editor.test.tsx`

- [ ] **Step 1: Write failing component tests**

```typescript
// apps/desktop/renderer/src/features/providers/__tests__/provider-paths-editor.test.tsx
// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, cleanup, fireEvent } from "@testing-library/react";
import React from "react";

vi.mock("../use-update-provider-paths.js", () => ({
  useUpdateProviderPaths: vi.fn(),
}));

import { ProviderPathsEditor } from "../provider-paths-editor.js";
import { useUpdateProviderPaths } from "../use-update-provider-paths.js";

const mockUseUpdateProviderPaths = useUpdateProviderPaths as ReturnType<typeof vi.fn>;

const defaultProps = {
  providerKey: "claude",
  scope: "project" as const,
  purpose: "detect" as const,
  currentPaths: [".claude"],
  onClose: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
  mockUseUpdateProviderPaths.mockReturnValue({
    mutate: vi.fn(),
    isPending: false,
    error: null,
    isError: false,
  });
});

afterEach(() => cleanup());

describe("ProviderPathsEditor", () => {
  it("renders the dialog with current paths pre-filled", () => {
    render(<ProviderPathsEditor {...defaultProps} />);
    const input = screen.getByRole("textbox") as HTMLInputElement;
    expect(input.value).toContain(".claude");
  });

  it("shows scope and purpose labels", () => {
    render(<ProviderPathsEditor {...defaultProps} />);
    expect(screen.getByText(/project/i)).not.toBeNull();
    expect(screen.getByText(/detect/i)).not.toBeNull();
  });

  it("calls mutate on save", () => {
    const mutate = vi.fn();
    mockUseUpdateProviderPaths.mockReturnValue({ mutate, isPending: false, error: null, isError: false });

    render(<ProviderPathsEditor {...defaultProps} />);
    fireEvent.click(screen.getByRole("button", { name: /save/i }));

    expect(mutate).toHaveBeenCalledWith({
      providerKey: "claude",
      scope: "project",
      purpose: "detect",
      paths: [".claude"],
    });
  });

  it("calls onClose on cancel", () => {
    const onClose = vi.fn();
    render(<ProviderPathsEditor {...defaultProps} onClose={onClose} />);
    fireEvent.click(screen.getByRole("button", { name: /cancel/i }));
    expect(onClose).toHaveBeenCalled();
  });

  it("shows a metadata note that overrides are config only", () => {
    render(<ProviderPathsEditor {...defaultProps} />);
    expect(screen.getByText(/configuration metadata|behavior integration/i)).not.toBeNull();
  });
});
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd apps/desktop && pnpm test -- --testPathPattern="provider-paths-editor" 2>&1 | head -20
```

Expected: import error — module not found.

- [ ] **Step 3: Implement ProviderPathsEditor**

```tsx
// apps/desktop/renderer/src/features/providers/provider-paths-editor.tsx
import React, { useState } from "react";
import { X } from "lucide-react";
import { useUpdateProviderPaths } from "./use-update-provider-paths.js";

interface Props {
  providerKey: string;
  scope: "project" | "global";
  purpose: string;
  currentPaths: string[];
  onClose: () => void;
}

export function ProviderPathsEditor({ providerKey, scope, purpose, currentPaths, onClose }: Props): React.JSX.Element {
  const [rawPaths, setRawPaths] = useState(currentPaths.join("\n"));
  const mutation = useUpdateProviderPaths();

  function handleSave() {
    const paths = rawPaths.split("\n").map((p) => p.trim()).filter(Boolean);
    mutation.mutate(
      { providerKey, scope, purpose, paths: paths.length > 0 ? paths : currentPaths },
      { onSuccess: onClose },
    );
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30">
      <div className="w-full max-w-sm rounded border border-zinc-200 bg-white shadow-lg">
        <div className="flex items-center justify-between border-b border-zinc-100 px-4 py-3">
          <div>
            <div className="text-sm font-semibold text-zinc-800">Edit paths</div>
            <div className="mt-0.5 text-xs text-zinc-500">
              <span className="font-mono">{providerKey}</span>
              {" · "}
              <span>{scope}</span>
              {" · "}
              <span>{purpose}</span>
            </div>
          </div>
          <button onClick={onClose} className="rounded p-1 text-zinc-400 hover:bg-zinc-100">
            <X size={14} />
          </button>
        </div>
        <div className="px-4 py-3">
          <label className="mb-1 block text-xs font-medium text-zinc-600">
            Paths (one per line)
          </label>
          <textarea
            className="w-full rounded border border-zinc-200 px-2 py-1.5 font-mono text-xs text-zinc-800 focus:outline-none focus:ring-1 focus:ring-zinc-400"
            rows={4}
            value={rawPaths}
            onChange={(e) => setRawPaths(e.target.value)}
          />
          <p className="mt-2 text-xs text-zinc-400">
            These are configuration metadata only. Behavior integration (scan, install) is a later slice.
          </p>
          {mutation.isError && mutation.error != null && (
            <p className="mt-1 text-xs text-red-500">{String(mutation.error)}</p>
          )}
        </div>
        <div className="flex justify-end gap-2 border-t border-zinc-100 px-4 py-3">
          <button
            onClick={onClose}
            className="rounded border border-zinc-200 px-3 py-1.5 text-xs text-zinc-600 hover:bg-zinc-50"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            disabled={mutation.isPending}
            className="rounded bg-zinc-800 px-3 py-1.5 text-xs text-white hover:bg-zinc-700 disabled:opacity-50"
          >
            {mutation.isPending ? "Saving…" : "Save"}
          </button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run tests**

```bash
cd apps/desktop && pnpm test -- --testPathPattern="provider-paths-editor"
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/renderer/src/features/providers/provider-paths-editor.tsx \
        apps/desktop/renderer/src/features/providers/__tests__/provider-paths-editor.test.tsx
git commit -m "Add ProviderPathsEditor dialog component (PR-2A)"
```

---

## Task 9: Update Settings screen with override UI

**Files:**
- Modify: `apps/desktop/renderer/src/screens/settings-screen.tsx`
- Modify: `apps/desktop/renderer/src/screens/__tests__/settings-screen.test.tsx`

- [ ] **Step 1: Write failing tests for override/edit/reset**

Add these test cases to the existing `settings-screen.test.tsx` describe block:

```typescript
// Add these vi.mock entries at the top (after existing mocks):
vi.mock("../../features/providers/provider-paths-editor.js", () => ({
  ProviderPathsEditor: ({ onClose }: { onClose: () => void }) => (
    <div data-testid="paths-editor">
      <button onClick={onClose}>CloseEditor</button>
    </div>
  ),
}));
vi.mock("../../features/providers/use-reset-provider-paths.js", () => ({
  useResetProviderPaths: vi.fn(),
}));

// Add to imports:
// import { useResetProviderPaths } from "../../features/providers/use-reset-provider-paths.js";
// const mockUseResetProviderPaths = useResetProviderPaths as ReturnType<typeof vi.fn>;

// Add to beforeEach:
// mockUseResetProviderPaths.mockReturnValue({ mutate: vi.fn(), isPending: false });

// Add test cases:
it("shows Override badge when any candidate has source=override", () => {
  mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
  mockUseProviderList.mockReturnValue({
    data: {
      providers: [makeProvider({
        candidates: [
          { relativePath: ".custom", scope: "project" as const, purpose: "detect" as const, priority: 10, source: "override" as const, verificationStatus: "assumed" as const },
        ],
      })],
    },
  });
  render(<SettingsScreen />);
  expect(screen.getByText(/override/i)).not.toBeNull();
});

it("shows Edit button for each provider row", () => {
  mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
  mockUseProviderList.mockReturnValue({ data: { providers: [makeProvider()] } });
  render(<SettingsScreen />);
  expect(screen.getAllByRole("button", { name: /edit/i }).length).toBeGreaterThan(0);
});

it("opens editor dialog when Edit clicked", () => {
  mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
  mockUseProviderList.mockReturnValue({ data: { providers: [makeProvider()] } });
  render(<SettingsScreen />);
  fireEvent.click(screen.getAllByRole("button", { name: /edit/i })[0]);
  expect(screen.getByTestId("paths-editor")).not.toBeNull();
});

it("shows Reset button for provider with override candidates", () => {
  mockUseAppSettings.mockReturnValue({ isPending: false, isError: false, data: baseSettings });
  mockUseProviderList.mockReturnValue({
    data: {
      providers: [makeProvider({
        candidates: [
          { relativePath: ".custom", scope: "project" as const, purpose: "detect" as const, priority: 10, source: "override" as const, verificationStatus: "assumed" as const },
        ],
      })],
    },
  });
  render(<SettingsScreen />);
  expect(screen.getByRole("button", { name: /reset/i })).not.toBeNull();
});
```

Also add `fireEvent` to imports: `import { render, screen, cleanup, fireEvent } from "@testing-library/react";`

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd apps/desktop && pnpm test -- --testPathPattern="settings-screen" 2>&1 | head -30
```

Expected: failures for "Override badge", "Edit button", "opens editor", "Reset button" tests.

- [ ] **Step 3: Update settings-screen.tsx**

Replace the `<div>` section inside `<div>` with providers table:

```tsx
// Add imports at top:
import React, { useState } from "react";
import { Pencil, RotateCcw } from "lucide-react";
import { ProviderPathsEditor } from "../features/providers/provider-paths-editor.js";
import { useResetProviderPaths } from "../features/providers/use-reset-provider-paths.js";
import type { ProviderListProvider, ProviderListPathCandidate } from "@contracts/index.js";

// Add after existing helper functions:
function hasOverride(provider: ProviderListProvider): boolean {
  return provider.candidates.some((c) => c.source === "override");
}

function candidatePathsWithSource(
  provider: ProviderListProvider,
  scope: "project" | "global",
  purpose: "detect" | "skills",
): { paths: string[]; source: string } {
  const cands = provider.candidates.filter((c) => c.scope === scope && c.purpose === purpose);
  const sorted = [...cands].sort((a, b) => b.priority - a.priority);
  const source = sorted.some((c) => c.source === "override") ? "override" : "builtin";
  return { paths: sorted.map((c) => c.relativePath), source };
}
```

Replace the provider table row `<tr>` content to include edit button, override badge, and reset button. Here is the updated `SettingsScreen` component — only the providers section changes:

```tsx
// Inside SettingsScreen, replace the providers section:
<div>
  <div className="flex items-center justify-between">
    <h3 className="text-sm font-semibold text-zinc-800">Providers</h3>
  </div>
  <p className="mt-0.5 text-xs text-zinc-500">
    Override paths are configuration metadata only — scan and install behavior integration is a later slice.
  </p>

  <div className="mt-3 overflow-x-auto rounded border border-zinc-200">
    <table className="min-w-full text-xs">
      <thead>
        <tr className="border-b border-zinc-100 bg-zinc-50 text-left text-zinc-500">
          <th className="px-3 py-2 font-medium">Provider</th>
          <th className="px-3 py-2 font-medium">Key</th>
          <th className="px-3 py-2 font-medium">Status</th>
          <th className="px-3 py-2 font-medium">Project detect</th>
          <th className="px-3 py-2 font-medium">Project skills</th>
          <th className="px-3 py-2 font-medium">Global skills</th>
          <th className="px-3 py-2 font-medium">Actions</th>
        </tr>
      </thead>
      <tbody className="divide-y divide-zinc-100">
        {(providerData?.providers ?? []).map((provider) => (
          <ProviderRow key={provider.key} provider={provider} />
        ))}
        {(providerData?.providers ?? []).length === 0 && (
          <tr>
            <td colSpan={7} className="px-3 py-4 text-center text-zinc-400">
              Loading providers…
            </td>
          </tr>
        )}
      </tbody>
    </table>
  </div>
</div>
```

Add a `ProviderRow` component below the helper functions:

```tsx
function ProviderRow({ provider }: { provider: ProviderListProvider }): React.JSX.Element {
  const [editSlot, setEditSlot] = useState<{ scope: "project" | "global"; purpose: string; paths: string[] } | null>(null);
  const resetMutation = useResetProviderPaths();

  const projectDetect = candidatePathsWithSource(provider, "project", "detect");
  const projectSkills = candidatePathsWithSource(provider, "project", "skills");
  const globalSkills = provider.hasGlobalLevel ? candidatePathsWithSource(provider, "global", "skills") : null;
  const overridden = hasOverride(provider);

  function handleReset() {
    const overrideCand = provider.candidates.find((c) => c.source === "override");
    if (!overrideCand) return;
    resetMutation.mutate({ providerKey: provider.key, scope: overrideCand.scope as "project" | "global", purpose: overrideCand.purpose });
  }

  return (
    <>
      <tr className={!provider.isAvailable ? "opacity-50" : ""}>
        <td className="px-3 py-2">
          <div className="flex items-center gap-2">
            <ProviderIcon providerKey={provider.key} iconKey={provider.iconKey} />
            <span className="font-medium text-zinc-800">{provider.displayName}</span>
            {overridden && (
              <span className="inline-flex items-center rounded bg-blue-50 px-1.5 py-0.5 text-[10px] font-medium text-blue-600">
                Override
              </span>
            )}
          </div>
        </td>
        <td className="px-3 py-2 font-mono text-zinc-500">{provider.key}</td>
        <td className="px-3 py-2">
          <ProviderStatusBadge status={provider.status} />
        </td>
        <td className="px-3 py-2">
          <PathListWithSource data={projectDetect} />
        </td>
        <td className="px-3 py-2">
          <PathListWithSource data={projectSkills} />
        </td>
        <td className="px-3 py-2">
          {globalSkills ? <PathListWithSource data={globalSkills} /> : <span className="text-zinc-300">—</span>}
        </td>
        <td className="px-3 py-2">
          <div className="flex items-center gap-1">
            <button
              onClick={() => setEditSlot({ scope: "project", purpose: "detect", paths: projectDetect.paths })}
              className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700"
              title="Edit project detect paths"
            >
              <Pencil size={12} aria-label="Edit" />
            </button>
            {overridden && (
              <button
                onClick={handleReset}
                disabled={resetMutation.isPending}
                className="rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-red-500 disabled:opacity-50"
                title="Reset to defaults"
              >
                <RotateCcw size={12} aria-label="Reset" />
              </button>
            )}
          </div>
        </td>
      </tr>
      {editSlot != null && (
        <ProviderPathsEditor
          providerKey={provider.key}
          scope={editSlot.scope}
          purpose={editSlot.purpose}
          currentPaths={editSlot.paths}
          onClose={() => setEditSlot(null)}
        />
      )}
    </>
  );
}

function PathListWithSource({ data }: { data: { paths: string[]; source: string } }): React.JSX.Element {
  if (data.paths.length === 0) return <span className="text-zinc-400">—</span>;
  return (
    <span className={`font-mono text-xs ${data.source === "override" ? "text-blue-600" : "text-zinc-600"}`}>
      {data.paths.join(", ")}
    </span>
  );
}
```

Also remove the old `candidatePaths` and `PathList` helpers (replaced by `candidatePathsWithSource` and `PathListWithSource`).

Remove the old `<p>` line "Built-in provider registry — read only. Path overrides and enable/disable controls are coming in a future update." and replace with the new text shown above.

- [ ] **Step 4: Run settings screen tests**

```bash
cd apps/desktop && pnpm test -- --testPathPattern="settings-screen"
```

Expected: all PASS.

- [ ] **Step 5: Typecheck**

```bash
cd apps/desktop && pnpm typecheck
```

Expected: success.

- [ ] **Step 6: Commit**

```bash
git add apps/desktop/renderer/src/screens/settings-screen.tsx \
        apps/desktop/renderer/src/screens/__tests__/settings-screen.test.tsx
git commit -m "Update Settings Providers table with override indicator + edit/reset (PR-2A)"
```

---

## Task 10: Final verification

- [ ] **Step 1: Run all Go tests**

```bash
cd core-go && go test ./...
```

Expected: all PASS.

- [ ] **Step 2: Run all renderer tests**

```bash
cd apps/desktop && pnpm test
```

Expected: all PASS.

- [ ] **Step 3: Typecheck**

```bash
cd apps/desktop && pnpm typecheck
```

Expected: success.

- [ ] **Step 4: Contract drift check**

```bash
cd apps/desktop && pnpm check:contracts-drift
```

Expected: no drift.

- [ ] **Step 5: git diff --check**

```bash
git diff --check
```

Expected: no whitespace errors.

- [ ] **Step 6: Squash or final commit if needed**

If any fixups are needed, commit them. Then report the HEAD commit hash.

---

## Self-Review Checklist

**Spec coverage:**
- [x] Migration with down + FK safety + tests → Task 1
- [x] Override storage (separate table, no mutation of defaults) → Tasks 1, 3
- [x] provider.list returns source/override info → Tasks 2, 4, 5
- [x] provider.updatePaths with all validations → Tasks 4, 5
- [x] provider.resetPaths → Tasks 4, 5
- [x] No provider.setEnabled → not added (correct)
- [x] No custom provider creation → not added (correct)
- [x] Settings override indicator → Task 9
- [x] Edit dialog → Tasks 8, 9
- [x] Reset action → Tasks 7, 9
- [x] Dense operational UI, no marketing copy → Task 9
- [x] Config-metadata note in UI → Tasks 8, 9
- [x] Contract generation clean → Task 6
- [x] All 5 verification commands → Task 10

**Placeholder scan:** None found. All code blocks contain actual implementation.

**Type consistency:** `ProviderPathsEditor` props use `scope: "project" | "global"` consistent with contract types. `useUpdateProviderPaths`/`useResetProviderPaths` use types from `@contracts/index.js`. `ProviderPathOverride` struct used consistently between domain, repo, and service.
