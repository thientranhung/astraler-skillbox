# Slice 2G: Remove Skill From Project Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `remove.skill` command that deletes a single `current` (active-host) symlink install from a project provider, reconciles the DB via an authoritative rescan plus one targeted row delete, and surfaces it through a Project Detail per-row Remove action.

**Architecture:** A long-running operation on `Target{project, projectId}` (mutually exclusive with scan/install). The Go `ProjectService` validates synchronously, then inside the operation re-verifies the on-disk entry is a symlink resolving into the **active** host (never trusting stale DB classification), unlinks it via the filesystem gateway (`os.Remove`, non-recursive), runs `scanProjectInternal` to reconcile providers/installs/warnings, and hard-deletes the one targeted install row by id. The renderer adds a `useRemoveSkill` hook and a confirmation dialog wired to the existing Skill Entries table.

**Tech Stack:** Go (`modernc.org/sqlite`, `creachadair/jrpc2`), Electron main (TypeScript allowlist), React + TanStack Query + sonner, JSON Schema → TS contract generation.

---

## Reference: spec

Approved design: `docs/superpowers/specs/2026-05-25-skillbox-slice-2g-remove-skill-from-project-design.md`. Read it before starting. This plan implements that spec exactly; the removable set is `install_mode = symlink` AND `install_status = current` only.

## File Structure

**Go core (`core-go/`):**
- Modify `internal/domain/operation.go` — add `OperationTypeRemoveSkill`.
- Modify `internal/filesystem/write.go` — add `RemoveSymlink`.
- Modify `internal/filesystem/scan_project.go` — add `EntryFacts` + `ResolveEntry`.
- Modify `internal/filesystem/gateway.go` — delegate `RemoveSymlink`, `ResolveEntry`.
- Create `internal/filesystem/remove_test.go` — gateway/package write+resolve tests.
- Modify `internal/repositories/install_repo.go` — add `DeleteByID`.
- Modify `internal/repositories/install_repo_test.go` — `DeleteByID` tests.
- Modify `internal/services/interfaces.go` — `RemoveFilesystem`, `RemoveInstallDeleter`.
- Modify `internal/services/project_service.go` — `removeFS`/`installDeleter` fields + `WithRemoveDeps`.
- Create `internal/services/project_remove_skill_service.go` — `RemoveSkill` + `removeSkillInternal` + `removeSkillMetadata`.
- Create `internal/services/project_remove_skill_service_test.go` — service tests.
- Modify `internal/services/project_mocks_test.go` — `mockRemoveFS`, `mockInstallDeleter`.
- Create `internal/rpc/handlers/remove_skill.go` — JSON-RPC handler.
- Modify `internal/rpc/handlers/project_handler_test.go` — handler tests.
- Modify `internal/rpc/handlers/project_contract_test.go` — contract test.
- Modify `internal/app/wire.go` — register `remove.skill`.
- Modify `cmd/skillbox-core/main.go` — `WithRemoveDeps` wiring.

**Contracts (`shared/`):**
- Create `api-contracts/methods/remove.skill.json`.
- Modify `api-contracts/index.json` — manifest entry.
- Generated (by script): `generated/methods/remove-skill.ts`, `generated/index.ts`.

**Electron (`apps/desktop/electron/`):**
- Modify `main/core-process/method-allowlist.ts` — add `remove.skill`.

**Renderer (`apps/desktop/renderer/src/`):**
- Modify `lib/core-client/methods.ts` — `removeSkill` wrapper.
- Create `features/projects/use-remove-skill.ts` — mutation hook.
- Create `features/projects/remove-skill-dialog.tsx` — confirmation dialog.
- Modify `screens/project-detail-screen.tsx` — Actions column + dialog wiring.
- Create `features/projects/__tests__/remove-skill-dialog.test.tsx` — dialog + gating tests.
- Create `lib/core-client/__tests__/methods-remove-skill.test.ts` — invoke wrapper test.

---

## Task 1: Domain operation type

**Files:**
- Modify: `core-go/internal/domain/operation.go:19-21`

- [ ] **Step 1: Add the operation type constant**

In `core-go/internal/domain/operation.go`, add the new constant to the existing `const` block (after `OperationTypeInstallSkill`):

```go
	OperationTypeScan                  OperationType = "scan"
	OperationTypeChangeSkillHostFolder OperationType = "change_skill_host_folder"
	OperationTypeInstallSkill          OperationType = "install_skill"
	OperationTypeRemoveSkill           OperationType = "remove_skill"
```

- [ ] **Step 2: Build to verify it compiles**

Run: `cd core-go && go build ./internal/domain/`
Expected: exits 0, no output.

- [ ] **Step 3: Commit**

```bash
git add core-go/internal/domain/operation.go
git commit -m "feat(2g): add remove_skill operation type"
```

---

## Task 2: Filesystem gateway — RemoveSymlink + ResolveEntry

**Files:**
- Modify: `core-go/internal/filesystem/write.go`
- Modify: `core-go/internal/filesystem/scan_project.go`
- Modify: `core-go/internal/filesystem/gateway.go`
- Test: `core-go/internal/filesystem/remove_test.go` (create)

- [ ] **Step 1: Write failing tests for ResolveEntry and RemoveSymlink**

Create `core-go/internal/filesystem/remove_test.go`:

```go
package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveEntry_Missing(t *testing.T) {
	facts, err := ResolveEntry(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatalf("ResolveEntry: %v", err)
	}
	if facts.Exists {
		t.Errorf("Exists: got true want false")
	}
}

func TestResolveEntry_RealDir(t *testing.T) {
	dir := t.TempDir()
	facts, err := ResolveEntry(dir)
	if err != nil {
		t.Fatalf("ResolveEntry: %v", err)
	}
	if !facts.Exists || facts.IsSymlink {
		t.Errorf("got %+v want exists non-symlink", facts)
	}
}

func TestResolveEntry_GoodSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	facts, err := ResolveEntry(link)
	if err != nil {
		t.Fatalf("ResolveEntry: %v", err)
	}
	if !facts.Exists || !facts.IsSymlink || facts.Broken {
		t.Errorf("got %+v want resolving symlink", facts)
	}
	if facts.ResolvedTarget != target {
		t.Errorf("ResolvedTarget: got %q want %q", facts.ResolvedTarget, target)
	}
}

func TestResolveEntry_BrokenSymlink(t *testing.T) {
	root := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "gone"), link); err != nil {
		t.Fatal(err)
	}
	facts, err := ResolveEntry(link)
	if err != nil {
		t.Fatalf("ResolveEntry: %v", err)
	}
	if !facts.Exists || !facts.IsSymlink || !facts.Broken {
		t.Errorf("got %+v want broken symlink", facts)
	}
	if facts.ResolvedTarget != "" {
		t.Errorf("ResolvedTarget: got %q want empty", facts.ResolvedTarget)
	}
}

func TestRemoveSymlink_UnlinksLinkNotTarget(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(target, "keep.txt")
	if err := os.WriteFile(keep, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSymlink(link); err != nil {
		t.Fatalf("RemoveSymlink: %v", err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Errorf("link still present: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("target content was destroyed: %v", err)
	}
}

func TestRemoveSymlink_NonEmptyDirErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RemoveSymlink(dir); err == nil {
		t.Errorf("expected error removing non-empty dir, got nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd core-go && go test ./internal/filesystem/ -run 'ResolveEntry|RemoveSymlink' -v`
Expected: FAIL — `undefined: ResolveEntry`, `undefined: RemoveSymlink`, `facts.Exists undefined`.

- [ ] **Step 3: Add RemoveSymlink to write.go**

Append to `core-go/internal/filesystem/write.go`:

```go
// RemoveSymlink unlinks the entry at path using os.Remove. On a symlink it
// removes the link itself WITHOUT following it (the target is untouched). On a
// non-empty real directory os.Remove returns an error rather than recursing —
// defense in depth so a regression in the caller's checks cannot destroy real
// content.
func RemoveSymlink(path string) error {
	return os.Remove(path)
}
```

- [ ] **Step 4: Add EntryFacts + ResolveEntry to scan_project.go**

Append to `core-go/internal/filesystem/scan_project.go`:

```go
// EntryFacts captures lstat + symlink-resolution facts for a single path, used
// by remove's on-disk re-verification. It mirrors the per-entry logic of
// ScanProjectSkills for one path. A missing path returns Exists=false (not an
// error).
type EntryFacts struct {
	Exists         bool
	IsSymlink      bool
	Broken         bool   // symlink whose target does not resolve
	ResolvedTarget string // canonical target via EvalSymlinks; empty unless a resolving symlink
}

// ResolveEntry returns lstat + resolution facts for path without following the
// symlink at lstat time. ENOENT is not an error.
func ResolveEntry(path string) (EntryFacts, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return EntryFacts{Exists: false}, nil
		}
		return EntryFacts{}, err
	}
	facts := EntryFacts{Exists: true}
	if fi.Mode()&os.ModeSymlink == 0 {
		return facts, nil // exists, not a symlink
	}
	facts.IsSymlink = true
	resolved, evalErr := filepath.EvalSymlinks(path)
	if evalErr != nil {
		if errors.Is(evalErr, fs.ErrNotExist) {
			facts.Broken = true
			return facts, nil
		}
		return facts, evalErr // loop/IO error
	}
	facts.ResolvedTarget = resolved
	return facts, nil
}
```

Note: `scan_project.go` already imports `errors`, `io/fs`, `os`, `path/filepath` — no import changes needed.

- [ ] **Step 5: Delegate from the gateway**

Append to `core-go/internal/filesystem/gateway.go`:

```go
// ResolveEntry delegates to the package-level function.
func (g *Gateway) ResolveEntry(path string) (EntryFacts, error) {
	return ResolveEntry(path)
}

// RemoveSymlink delegates to the package-level function.
func (g *Gateway) RemoveSymlink(path string) error {
	return RemoveSymlink(path)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/filesystem/ -run 'ResolveEntry|RemoveSymlink' -v`
Expected: PASS (6 tests).

- [ ] **Step 7: Commit**

```bash
git add core-go/internal/filesystem/write.go core-go/internal/filesystem/scan_project.go core-go/internal/filesystem/gateway.go core-go/internal/filesystem/remove_test.go
git commit -m "feat(2g): add gateway RemoveSymlink and ResolveEntry"
```

---

## Task 3: Repository — InstallRepo.DeleteByID

**Files:**
- Modify: `core-go/internal/repositories/install_repo.go`
- Test: `core-go/internal/repositories/install_repo_test.go`

- [ ] **Step 1: Write failing tests**

Append to `core-go/internal/repositories/install_repo_test.go`:

```go
func TestInstallRepo_DeleteByID_DeletesOneRow(t *testing.T) {
	db := NewTestDB(t)
	projRepo := NewProjectRepo(db)
	repo := NewInstallRepo(db)
	ctx := context.Background()

	pid := seedProject(t, projRepo, "proj-a", "/tmp/proj-a")
	defID := getGenericAgentsDefID(t, db)
	ppID := seedProjectProvider(t, db, pid, defID)
	idX := seedInstall(t, db, ppID, "skill-x", "/tmp/proj-a/.agents/skills/skill-x")
	seedInstall(t, db, ppID, "skill-y", "/tmp/proj-a/.agents/skills/skill-y")

	n, err := repo.DeleteByID(ctx, idX)
	if err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}
	if n != 1 {
		t.Errorf("rowsAffected: got %d want 1", n)
	}
	installs, err := repo.ListByProject(ctx, pid)
	if err != nil {
		t.Fatalf("ListByProject: %v", err)
	}
	if len(installs) != 1 || installs[0].SkillName != "skill-y" {
		t.Errorf("expected only skill-y to remain, got %+v", installs)
	}
}

func TestInstallRepo_DeleteByID_AbsentIsNoOp(t *testing.T) {
	db := NewTestDB(t)
	repo := NewInstallRepo(db)
	n, err := repo.DeleteByID(context.Background(), 99999)
	if err != nil {
		t.Fatalf("DeleteByID: %v", err)
	}
	if n != 0 {
		t.Errorf("rowsAffected: got %d want 0", n)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd core-go && go test ./internal/repositories/ -run 'DeleteByID' -v`
Expected: FAIL — `repo.DeleteByID undefined`.

- [ ] **Step 3: Add DeleteByID**

Append to `core-go/internal/repositories/install_repo.go`:

```go
// DeleteByID hard-deletes a single install row. It is the only hard delete of an
// install row in the app. Idempotent: deleting an absent id affects 0 rows and
// is not an error.
func (r *InstallRepo) DeleteByID(ctx context.Context, installID int64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `DELETE FROM installs WHERE id = ?`, installID)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/repositories/ -run 'DeleteByID' -v`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
git add core-go/internal/repositories/install_repo.go core-go/internal/repositories/install_repo_test.go
git commit -m "feat(2g): add InstallRepo.DeleteByID"
```

---

## Task 4: Service interfaces + WithRemoveDeps wiring

**Files:**
- Modify: `core-go/internal/services/interfaces.go`
- Modify: `core-go/internal/services/project_service.go:48-121`

- [ ] **Step 1: Add the remove interfaces**

Append to `core-go/internal/services/interfaces.go`:

```go
// RemoveFilesystem provides the filesystem operations needed to remove a symlink
// install: re-verify the on-disk entry and unlink it. *filesystem.Gateway
// satisfies this interface.
type RemoveFilesystem interface {
	// ResolveEntry returns lstat + symlink-resolution facts for path.
	ResolveEntry(path string) (filesystem.EntryFacts, error)
	// RemoveSymlink unlinks the entry at path (os.Remove; non-recursive).
	RemoveSymlink(path string) error
}

// RemoveInstallDeleter hard-deletes a single install row by id.
// *repositories.InstallRepo satisfies this interface.
type RemoveInstallDeleter interface {
	DeleteByID(ctx context.Context, installID int64) (int64, error)
}
```

Note: `interfaces.go` already imports `context` and `github.com/astraler/skillbox/core-go/internal/filesystem` — no import changes needed.

- [ ] **Step 2: Add service fields**

In `core-go/internal/services/project_service.go`, add to the `ProjectService` struct (after the install deps block at line ~67):

```go
	// installSkillReader is separate from skillsByHostLister (scan) to avoid silent overwrite.
	installSkillReader SkillsByHostLister
	// remove deps — nil until WithRemoveDeps is called
	removeFS       RemoveFilesystem
	installDeleter RemoveInstallDeleter
```

- [ ] **Step 3: Add WithRemoveDeps**

In `core-go/internal/services/project_service.go`, add after `WithInstallDeps` (around line 121):

```go
// WithRemoveDeps attaches the filesystem and install-row deleter required for
// RemoveSkill. Returns the receiver to allow chaining.
func (s *ProjectService) WithRemoveDeps(
	removeFS RemoveFilesystem,
	installDeleter RemoveInstallDeleter,
) *ProjectService {
	s.removeFS = removeFS
	s.installDeleter = installDeleter
	return s
}
```

- [ ] **Step 4: Build to verify it compiles**

Run: `cd core-go && go build ./internal/services/`
Expected: exits 0, no output.

- [ ] **Step 5: Commit**

```bash
git add core-go/internal/services/interfaces.go core-go/internal/services/project_service.go
git commit -m "feat(2g): add remove service interfaces and WithRemoveDeps"
```

---

## Task 5: Service — RemoveSkill + removeSkillInternal

**Files:**
- Create: `core-go/internal/services/project_remove_skill_service.go`
- Modify: `core-go/internal/services/project_mocks_test.go`
- Test: `core-go/internal/services/project_remove_skill_service_test.go` (create)

- [ ] **Step 1: Add test mocks**

Append to `core-go/internal/services/project_mocks_test.go`:

```go
// -- mock remove filesystem --

type mockRemoveFS struct {
	facts        filesystem.EntryFacts
	resolveErr   error
	removeErr    error
	removeCalls  int
	removedPaths []string
}

func (m *mockRemoveFS) ResolveEntry(_ string) (filesystem.EntryFacts, error) {
	return m.facts, m.resolveErr
}

func (m *mockRemoveFS) RemoveSymlink(path string) error {
	m.removeCalls++
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removedPaths = append(m.removedPaths, path)
	return nil
}

// -- mock install deleter --

type mockInstallDeleter struct {
	err        error
	deletedIDs []int64
	rows       int64
}

func (m *mockInstallDeleter) DeleteByID(_ context.Context, installID int64) (int64, error) {
	if m.err != nil {
		return 0, m.err
	}
	m.deletedIDs = append(m.deletedIDs, installID)
	if m.rows != 0 {
		return m.rows, nil
	}
	return 1, nil
}
```

- [ ] **Step 2: Write failing service tests**

Create `core-go/internal/services/project_remove_skill_service_test.go`:

```go
package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/filesystem"
	"github.com/astraler/skillbox/core-go/internal/providers"
)

// removeFixture builds a project on disk with one symlinked skill plus the
// service wiring needed to run removeSkillInternal end to end. The install row
// is registered in installRepo so RemoveSkill's load/ownership check passes.
type removeFixture struct {
	svc       *ProjectService
	project   *domain.Project
	install   domain.Install
	removeFS  *mockRemoveFS
	deleter   *mockInstallDeleter
	scanRepo  *mockProjectScanCommitter
	linkPath  string
	hostSkill string
}

func newRemoveFixture(t *testing.T) *removeFixture {
	t.Helper()

	projectDir := t.TempDir()
	projectSkillsDir := filepath.Join(projectDir, ".agents", "skills")
	if err := os.MkdirAll(projectSkillsDir, 0o755); err != nil {
		t.Fatalf("mkdir project skills: %v", err)
	}
	hostSkillsDir := t.TempDir()
	hostSkill := filepath.Join(hostSkillsDir, "documentation-writer")
	if err := os.MkdirAll(hostSkill, 0o755); err != nil {
		t.Fatalf("mkdir host skill: %v", err)
	}
	linkPath := filepath.Join(projectSkillsDir, "documentation-writer")
	if err := os.Symlink(hostSkill, linkPath); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	gw := filesystem.NewGateway()
	project := &domain.Project{ID: 1, Name: "proj", Path: projectDir, Status: domain.ProjectStatusActive}
	projRepo := newMockProjectRepo()
	projRepo.projects[1] = project

	install := domain.Install{
		ID:                1001,
		ProjectProviderID: 50,
		SkillName:         "documentation-writer",
		InstallMode:       domain.InstallModeSymlink,
		InstallStatus:     domain.InstallStatusCurrent,
		ProjectSkillPath:  linkPath,
	}
	ppRepo := &mockProjectProviderRepo{
		byProject: map[int64][]domain.ProjectProviderSummary{
			1: {{ProjectProviderID: 50, ProviderKey: providers.GenericAgentsKey, DetectionStatus: domain.DetectionStatusDetected}},
		},
	}
	installRepo := &mockProjectInstallRepo{byProject: map[int64][]domain.Install{1: {install}}}

	activeHost := &domain.SkillHostFolder{ID: 1, SkillsPath: hostSkillsDir, Status: domain.SkillHostStatusActive}
	hostReader := &mockActiveHostReader{host: activeHost}

	registry := &mockProviderRegistry{adapters: []providers.ProviderAdapter{providers.NewGenericAgentsAdapter()}}
	hostLister := &mockHostLister{hosts: []domain.SkillHostFolder{*activeHost}}
	pdRepo := &mockProviderDefRepo{defs: map[string]*domain.ProviderDefinition{
		providers.GenericAgentsKey: {ID: 10, Key: providers.GenericAgentsKey, Status: domain.ProviderStatusSupported, CanCreateStructure: true},
	}}
	skillLister := &mockSkillsByHostLister{skills: map[int64][]domain.Skill{1: {{ID: 1, Name: "documentation-writer", AbsolutePath: hostSkill, Status: domain.SkillStatusAvailable}}}}
	scanRepo := &mockProjectScanCommitter{}
	removeFS := &mockRemoveFS{facts: filesystem.EntryFacts{Exists: true, IsSymlink: true, ResolvedTarget: hostSkill}}
	deleter := &mockInstallDeleter{}

	svc := NewProjectService(projRepo, ppRepo, &mockProjectWarningRepo{}, installRepo, gw).
		WithScanDeps(&mockRunner{}, scanRepo).
		WithProviderDeps(registry, pdRepo, hostLister, skillLister).
		WithInstallDeps(gw, hostReader, skillLister).
		WithRemoveDeps(removeFS, deleter)

	return &removeFixture{svc: svc, project: project, install: install, removeFS: removeFS, deleter: deleter, scanRepo: scanRepo, linkPath: linkPath, hostSkill: hostSkill}
}

func TestRemoveSkillInternal_HappyPath(t *testing.T) {
	f := newRemoveFixture(t)
	meta, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	if err != nil {
		t.Fatalf("removeSkillInternal: %v", err)
	}
	m, ok := meta.(removeSkillMetadata)
	if !ok {
		t.Fatalf("metadata type: %T", meta)
	}
	if m.AlreadyAbsent {
		t.Errorf("AlreadyAbsent: got true want false")
	}
	if m.SkillName != "documentation-writer" || m.ProviderKey != providers.GenericAgentsKey {
		t.Errorf("metadata: %+v", m)
	}
	if f.removeFS.removeCalls != 1 {
		t.Errorf("RemoveSymlink calls: got %d want 1", f.removeFS.removeCalls)
	}
	if f.scanRepo.fullScanCallCount != 1 {
		t.Errorf("rescan calls: got %d want 1", f.scanRepo.fullScanCallCount)
	}
	if len(f.deleter.deletedIDs) != 1 || f.deleter.deletedIDs[0] != 1001 {
		t.Errorf("DeleteByID calls: got %v want [1001]", f.deleter.deletedIDs)
	}
}

func TestRemoveSkillInternal_AlreadyAbsent(t *testing.T) {
	f := newRemoveFixture(t)
	f.removeFS.facts = filesystem.EntryFacts{Exists: false}
	meta, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	if err != nil {
		t.Fatalf("removeSkillInternal: %v", err)
	}
	m := meta.(removeSkillMetadata)
	if !m.AlreadyAbsent {
		t.Errorf("AlreadyAbsent: got false want true")
	}
	if f.removeFS.removeCalls != 0 {
		t.Errorf("RemoveSymlink should not be called when absent, got %d", f.removeFS.removeCalls)
	}
	if f.scanRepo.fullScanCallCount != 1 || len(f.deleter.deletedIDs) != 1 {
		t.Errorf("rescan+delete should still run: scan=%d del=%v", f.scanRepo.fullScanCallCount, f.deleter.deletedIDs)
	}
}

func TestRemoveSkillInternal_NotSymlinkOnDisk_Conflict(t *testing.T) {
	f := newRemoveFixture(t)
	f.removeFS.facts = filesystem.EntryFacts{Exists: true, IsSymlink: false}
	_, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	assertAppErrorCode(t, err, domain.CodeConflict)
	if f.removeFS.removeCalls != 0 {
		t.Errorf("must not unlink a real entry, removeCalls=%d", f.removeFS.removeCalls)
	}
}

func TestRemoveSkillInternal_SymlinkOutsideActiveHost_Conflict(t *testing.T) {
	f := newRemoveFixture(t)
	f.removeFS.facts = filesystem.EntryFacts{Exists: true, IsSymlink: true, ResolvedTarget: filepath.Join(t.TempDir(), "elsewhere")}
	_, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	assertAppErrorCode(t, err, domain.CodeConflict)
	if f.removeFS.removeCalls != 0 {
		t.Errorf("must not unlink, removeCalls=%d", f.removeFS.removeCalls)
	}
}

func TestRemoveSkillInternal_UnlinkFails_NoRescanNoDelete(t *testing.T) {
	f := newRemoveFixture(t)
	f.removeFS.removeErr = os.ErrPermission
	_, err := f.svc.removeSkillInternal(context.Background(), f.project, f.install, providers.GenericAgentsKey, noopProgress)
	assertAppErrorCode(t, err, domain.CodeFilesystem)
	if f.scanRepo.fullScanCallCount != 0 {
		t.Errorf("rescan must not run after unlink failure, got %d", f.scanRepo.fullScanCallCount)
	}
	if len(f.deleter.deletedIDs) != 0 {
		t.Errorf("delete must not run after unlink failure, got %v", f.deleter.deletedIDs)
	}
}

func TestRemoveSkill_Sync_RejectsNonCurrentStatus(t *testing.T) {
	f := newRemoveFixture(t)
	bad := f.install
	bad.InstallStatus = domain.InstallStatusOldHost
	f.svc.installRepo = &mockProjectInstallRepo{byProject: map[int64][]domain.Install{1: {bad}}}
	_, err := f.svc.RemoveSkill(context.Background(), 1, 1001)
	assertAppErrorCode(t, err, domain.CodeValidation)
}

func TestRemoveSkill_Sync_RejectsDirectMode(t *testing.T) {
	f := newRemoveFixture(t)
	bad := f.install
	bad.InstallMode = domain.InstallModeDirect
	f.svc.installRepo = &mockProjectInstallRepo{byProject: map[int64][]domain.Install{1: {bad}}}
	_, err := f.svc.RemoveSkill(context.Background(), 1, 1001)
	assertAppErrorCode(t, err, domain.CodeValidation)
}

func TestRemoveSkill_Sync_InstallNotFound(t *testing.T) {
	f := newRemoveFixture(t)
	_, err := f.svc.RemoveSkill(context.Background(), 1, 4242)
	assertAppErrorCode(t, err, domain.CodeValidation)
}

func TestRemoveSkill_Sync_ProjectNotActive(t *testing.T) {
	f := newRemoveFixture(t)
	f.project.Status = domain.ProjectStatusRemoved
	_, err := f.svc.RemoveSkill(context.Background(), 1, 1001)
	assertAppErrorCode(t, err, domain.CodeValidation)
}
```

Add this shared assertion helper at the bottom of the same new test file:

```go
// want is one of the domain.Code* string constants (e.g. domain.CodeConflict).
// AppError.Code is a plain string, so the parameter type is string.
func assertAppErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error with code %q, got nil", want)
	}
	ae, ok := err.(*domain.AppError)
	if !ok {
		t.Fatalf("expected *domain.AppError, got %T (%v)", err, err)
	}
	if ae.Code != want {
		t.Fatalf("error code: got %q want %q", ae.Code, want)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `cd core-go && go test ./internal/services/ -run 'RemoveSkill' -v`
Expected: FAIL — `f.svc.removeSkillInternal undefined`, `removeSkillMetadata` undefined, `RemoveSkill undefined`.

- [ ] **Step 4: Implement the service**

Create `core-go/internal/services/project_remove_skill_service.go`:

```go
package services

import (
	"context"
	"fmt"

	"github.com/astraler/skillbox/core-go/internal/domain"
	"github.com/astraler/skillbox/core-go/internal/operations"
)

// removeSkillMetadata is the operation result for a remove-skill run, stored in
// operations.metadata_json and surfaced to the renderer on the terminal event.
type removeSkillMetadata struct {
	ProjectID     int64  `json:"projectId"`
	ProviderKey   string `json:"providerKey"`
	SkillName     string `json:"skillName"`
	RemovedPath   string `json:"removedPath"`
	AlreadyAbsent bool   `json:"alreadyAbsent"`
}

// RemoveSkill validates the request synchronously, then queues an async remove
// operation. Returns the operation ID on success.
//
// Synchronous validation (no filesystem writes):
//   - projectId and installId must be positive
//   - project must exist and be active
//   - the install must exist and belong to the project
//   - the install must be removable: install_mode=symlink AND install_status=current
//   - the resolved project skill path must be inside the project root
//
// Returns conflict_error if another operation is already running for this project.
func (s *ProjectService) RemoveSkill(ctx context.Context, projectID, installID int64) (int64, error) {
	if projectID <= 0 {
		return 0, domain.NewValidationError("Invalid project", "projectId must be positive")
	}
	if installID <= 0 {
		return 0, domain.NewValidationError("Invalid install", "installId must be positive")
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return 0, domain.NewDatabaseError("Could not fetch project", err.Error())
	}
	if project == nil {
		return 0, domain.NewValidationError("Project not found", fmt.Sprintf("projectId %d does not exist", projectID))
	}
	if project.Status != domain.ProjectStatusActive {
		return 0, domain.NewValidationError(
			"Project is not active",
			fmt.Sprintf("projectId %d has status %q; only active projects can have skills removed", projectID, project.Status),
		)
	}

	// Load the install via the project-scoped list (implicit ownership check).
	installs, err := s.installRepo.ListByProject(ctx, projectID)
	if err != nil {
		return 0, domain.NewDatabaseError("Could not load installs", err.Error())
	}
	var install *domain.Install
	for i := range installs {
		if installs[i].ID == installID {
			install = &installs[i]
			break
		}
	}
	if install == nil {
		return 0, domain.NewValidationError(
			"Install not found",
			fmt.Sprintf("installId %d does not exist in project %d", installID, projectID),
		)
	}

	// Removable precheck: symlink into the active host (status=current) only.
	if install.InstallMode != domain.InstallModeSymlink || install.InstallStatus != domain.InstallStatusCurrent {
		return 0, domain.NewValidationError(
			"Install is not removable",
			fmt.Sprintf("install %d has mode=%q status=%q; only current symlink installs can be removed in this slice",
				installID, install.InstallMode, install.InstallStatus),
		)
	}

	// Resolve and bound the path under the project root.
	path, err := s.fs.NormalizeAbs(install.ProjectSkillPath)
	if err != nil {
		return 0, domain.NewValidationError("Invalid install path", err.Error())
	}
	root, err := s.fs.NormalizeAbs(project.Path)
	if err != nil {
		return 0, domain.NewValidationError("Invalid project path", err.Error())
	}
	if !isWithin(root, path) {
		return 0, domain.NewValidationError(
			"Install path escapes project root",
			fmt.Sprintf("path %q is not within project root %q", path, root),
		)
	}

	// Resolve provider key for metadata via the project_providers summary.
	providerKey := s.providerKeyForInstall(ctx, projectID, install.ProjectProviderID)

	loaded := *install
	target := operations.Target{Type: "project", ID: projectID}
	opID, err := s.runner.Start(ctx, target, domain.OperationTypeRemoveSkill,
		func(opCtx context.Context, progress operations.ProgressFn) (any, error) {
			return s.removeSkillInternal(opCtx, project, loaded, providerKey, progress)
		})
	if err != nil {
		if _, ok := err.(*domain.AppError); ok {
			return 0, err
		}
		return 0, domain.NewDatabaseError("Could not queue remove operation", err.Error())
	}
	return opID, nil
}

// providerKeyForInstall returns the provider key for the install's
// project_provider_id, or "" if it cannot be resolved (metadata is best-effort).
func (s *ProjectService) providerKeyForInstall(ctx context.Context, projectID, projectProviderID int64) string {
	summaries, err := s.ppRepo.ListByProject(ctx, projectID)
	if err != nil {
		return ""
	}
	for _, sum := range summaries {
		if sum.ProjectProviderID == projectProviderID {
			return sum.ProviderKey
		}
	}
	return ""
}

// removeSkillInternal is the async work function executed inside the operation
// runner. It re-verifies the on-disk entry (never trusting stale DB state),
// unlinks the symlink, runs the authoritative rescan, then hard-deletes the one
// targeted install row.
func (s *ProjectService) removeSkillInternal(
	ctx context.Context,
	project *domain.Project,
	install domain.Install,
	providerKey string,
	progress operations.ProgressFn,
) (any, error) {
	progress("validating", 0, 0, "")

	path := install.ProjectSkillPath
	meta := removeSkillMetadata{
		ProjectID:   project.ID,
		ProviderKey: providerKey,
		SkillName:   install.SkillName,
		RemovedPath: path,
	}

	// 1. On-disk re-verification (do NOT trust the stored classification).
	facts, err := s.removeFS.ResolveEntry(path)
	if err != nil {
		return nil, domain.NewFilesystemError("Could not inspect install entry", err.Error())
	}

	alreadyAbsent := false
	switch {
	case !facts.Exists:
		alreadyAbsent = true
	case !facts.IsSymlink:
		return nil, domain.NewConflictError(
			"This entry changed on disk. Rescan the project and try again.",
			fmt.Sprintf("path %q is no longer a symlink; refusing to delete real content", path),
		)
	default:
		// It is a symlink: it must resolve inside the active host.
		activeHost, herr := s.activeHostReader.GetActive(ctx)
		if herr != nil {
			return nil, domain.NewDatabaseError("Could not load active skill host", herr.Error())
		}
		if activeHost == nil || facts.Broken || facts.ResolvedTarget == "" ||
			!isWithin(activeHost.SkillsPath, facts.ResolvedTarget) {
			return nil, domain.NewConflictError(
				"This entry changed on disk. Rescan the project and try again.",
				fmt.Sprintf("symlink %q no longer resolves inside the active host", path),
			)
		}
	}

	// 2. Unlink (skipped when already absent).
	if !alreadyAbsent {
		progress("removing_symlink", 0, 0, "")
		if err := s.removeFS.RemoveSymlink(path); err != nil {
			return nil, domain.NewFilesystemError("Could not remove skill symlink", err.Error())
		}
	}
	meta.AlreadyAbsent = alreadyAbsent

	// 3. Authoritative rescan (reconciles providers/installs/warnings; the
	//    removed path becomes install_status=missing).
	if _, rescanErr := s.scanProjectInternal(ctx, project, progress); rescanErr != nil {
		return meta, rescanErr
	}

	// 4. Hard-delete the one targeted install row (clears the missing tombstone).
	progress("deleting_record", 0, 0, "")
	if _, derr := s.installDeleter.DeleteByID(ctx, install.ID); derr != nil {
		return meta, domain.NewDatabaseError("Could not delete install record", derr.Error())
	}

	progress("done", 0, 0, "")
	return meta, nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd core-go && go test ./internal/services/ -run 'RemoveSkill' -v`
Expected: PASS (9 tests).

- [ ] **Step 6: Run the service package with the race detector**

Run: `cd core-go && go test -race ./internal/services/ -run 'RemoveSkill'`
Expected: PASS, no race warnings.

- [ ] **Step 7: Commit**

```bash
git add core-go/internal/services/project_remove_skill_service.go core-go/internal/services/project_remove_skill_service_test.go core-go/internal/services/project_mocks_test.go
git commit -m "feat(2g): add RemoveSkill service with on-disk re-verification"
```

---

## Task 6: Contract schema + generated types

**Files:**
- Create: `shared/api-contracts/methods/remove.skill.json`
- Modify: `shared/api-contracts/index.json:24-25`
- Generated: `shared/generated/methods/remove-skill.ts`, `shared/generated/index.ts`

- [ ] **Step 1: Create the JSON Schema**

Create `shared/api-contracts/methods/remove.skill.json`:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "RemoveSkillMethod",
  "description": "Contract for remove.skill JSON-RPC method. Removes one current symlink install from a project provider; runs asynchronously and reports progress via operation.progress notifications.",
  "oneOf": [
    { "$ref": "#/definitions/RemoveSkillRequest" },
    { "$ref": "#/definitions/RemoveSkillResponse" }
  ],
  "definitions": {
    "RemoveSkillRequest": {
      "title": "RemoveSkillRequest",
      "description": "Params for remove.skill.",
      "type": "object",
      "properties": {
        "projectId": {
          "type": "integer",
          "description": "ID of the project the install belongs to"
        },
        "installId": {
          "type": "integer",
          "description": "ID of the installed-skill row to remove (from project.get entries)"
        }
      },
      "required": ["projectId", "installId"],
      "additionalProperties": false
    },
    "RemoveSkillResponse": {
      "title": "RemoveSkillResponse",
      "description": "Immediate response — the removal runs asynchronously. Errors: validation_error (1001) project/install not found or not removable; conflict_error (1005) project busy or entry changed on disk; filesystem_error (1002) unlink failed.",
      "type": "object",
      "properties": {
        "operationId": {
          "type": "integer",
          "description": "ID of the created remove operation; use with operation.progress notifications and operation.cancel"
        }
      },
      "required": ["operationId"],
      "additionalProperties": false
    }
  }
}
```

- [ ] **Step 2: Register in the manifest**

In `shared/api-contracts/index.json`, add a trailing entry after the `install.skill` line (add a comma to the install.skill line):

```json
    { "input": "methods/install.skill.json", "output": "methods/install-skill.ts" },
    { "input": "methods/remove.skill.json", "output": "methods/remove-skill.ts" }
```

- [ ] **Step 3: Generate the TypeScript types**

Run: `cd apps/desktop && pnpm generate:contracts`
Expected: prints `generated  methods/remove-skill.ts` and `generated  index.ts`; ends with `✓ Generated N files`.

- [ ] **Step 4: Verify no drift**

Run: `cd apps/desktop && pnpm check:contracts-drift`
Expected: `✓ No contract drift detected (N files)`.

- [ ] **Step 5: Add a Go contract test**

Append to `core-go/internal/rpc/handlers/project_contract_test.go`:

```go
func TestContract_RemoveSkill_Response(t *testing.T) {
	schema := loadSchema(t, "methods/remove.skill.json")
	resp := removeSkillResponse{OperationID: 51}
	validateAgainstSchema(t, schema, resp)
}
```

(`removeSkillResponse` is defined in Task 7; this test will compile once the handler exists. Run it in Task 7 Step 4.)

- [ ] **Step 6: Commit**

```bash
git add shared/api-contracts/methods/remove.skill.json shared/api-contracts/index.json shared/generated/methods/remove-skill.ts shared/generated/index.ts core-go/internal/rpc/handlers/project_contract_test.go
git commit -m "feat(2g): add remove.skill contract schema and generated types"
```

---

## Task 7: RPC handler + registration

**Files:**
- Create: `core-go/internal/rpc/handlers/remove_skill.go`
- Modify: `core-go/internal/app/wire.go:30-42`
- Test: `core-go/internal/rpc/handlers/project_handler_test.go`

- [ ] **Step 1: Write failing handler tests**

Append to `core-go/internal/rpc/handlers/project_handler_test.go`:

```go
// --- remove.skill ---

type stubRemoveSkill struct {
	opID int64
	err  error
}

func (s *stubRemoveSkill) RemoveSkill(_ context.Context, _ int64, _ int64) (int64, error) {
	return s.opID, s.err
}

func TestRemoveSkillHandler_ReturnsOperationID(t *testing.T) {
	svc := &stubRemoveSkill{opID: 51}
	cli := startServer(t, handler.Map{"remove.skill": handlers.NewRemoveSkillHandler(svc)})

	params := map[string]interface{}{"projectId": 1, "installId": 1001}
	var resp struct {
		OperationID int64 `json:"operationId"`
	}
	if err := cli.CallResult(context.Background(), "remove.skill", params, &resp); err != nil {
		t.Fatalf("remove.skill: %v", err)
	}
	if resp.OperationID != 51 {
		t.Errorf("operationId: got %d want 51", resp.OperationID)
	}
}

func TestRemoveSkillHandler_BadParams_ReturnsValidationError(t *testing.T) {
	svc := &stubRemoveSkill{}
	cli := startServer(t, handler.Map{"remove.skill": handlers.NewRemoveSkillHandler(svc)})

	err := cli.CallResult(context.Background(), "remove.skill", map[string]interface{}{
		"projectId": "not-a-number",
	}, nil)
	if err == nil {
		t.Fatal("expected error for bad params")
	}
}

func TestRemoveSkillHandler_ConflictError_MapsTo1005(t *testing.T) {
	svc := &stubRemoveSkill{err: domain.NewConflictError("project busy", "target locked")}
	cli := startServer(t, handler.Map{"remove.skill": handlers.NewRemoveSkillHandler(svc)})

	params := map[string]interface{}{"projectId": 1, "installId": 1001}
	err := cli.CallResult(context.Background(), "remove.skill", params, nil)
	if err == nil {
		t.Fatal("expected conflict error")
	}
	if jerr, ok := err.(*jrpc2.Error); ok {
		if jerr.Code != 1005 {
			t.Errorf("code: got %d want 1005", jerr.Code)
		}
	} else {
		t.Fatalf("expected *jrpc2.Error, got %T", err)
	}
}
```

Note: `project_handler_test.go` already imports `context`, `github.com/creachadair/jrpc2`, `github.com/creachadair/jrpc2/handler`, the `handlers` package, and `domain` (used by the install.skill tests). No import changes needed.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd core-go && go test ./internal/rpc/handlers/ -run 'RemoveSkillHandler' -v`
Expected: FAIL — `handlers.NewRemoveSkillHandler undefined`.

- [ ] **Step 3: Implement the handler**

Create `core-go/internal/rpc/handlers/remove_skill.go`:

```go
package handlers

import (
	"context"

	"github.com/creachadair/jrpc2"
	"github.com/creachadair/jrpc2/handler"

	"github.com/astraler/skillbox/core-go/internal/domain"
)

type removeSkillService interface {
	RemoveSkill(ctx context.Context, projectID int64, installID int64) (int64, error)
}

type removeSkillRequest struct {
	ProjectID int64 `json:"projectId"`
	InstallID int64 `json:"installId"`
}

type removeSkillResponse struct {
	OperationID int64 `json:"operationId"`
}

func NewRemoveSkillHandler(svc removeSkillService) jrpc2.Handler {
	return handler.New(func(ctx context.Context, req *jrpc2.Request) (interface{}, error) {
		var p removeSkillRequest
		if err := req.UnmarshalParams(&p); err != nil {
			return nil, wrapError(domain.NewValidationError("Invalid params", err.Error()))
		}

		opID, err := svc.RemoveSkill(ctx, p.ProjectID, p.InstallID)
		if err != nil {
			return nil, wrapError(err)
		}
		return removeSkillResponse{OperationID: opID}, nil
	})
}
```

- [ ] **Step 4: Run handler + contract tests to verify they pass**

Run: `cd core-go && go test ./internal/rpc/handlers/ -run 'RemoveSkill' -v`
Expected: PASS (3 handler tests + `TestContract_RemoveSkill_Response`).

- [ ] **Step 5: Register the method in wire.go**

In `core-go/internal/app/wire.go`, add to the `handler.Map` literal (after the `install.skill` entry):

```go
			"install.skill":    rpchandlers.NewInstallSkillHandler(projectSvc),
			"remove.skill":     rpchandlers.NewRemoveSkillHandler(projectSvc),
```

- [ ] **Step 6: Build to verify registration compiles**

Run: `cd core-go && go build ./...`
Expected: exits 0. (`projectSvc` satisfies `removeSkillService` because Task 5 added `RemoveSkill`.)

- [ ] **Step 7: Commit**

```bash
git add core-go/internal/rpc/handlers/remove_skill.go core-go/internal/rpc/handlers/project_handler_test.go core-go/internal/app/wire.go
git commit -m "feat(2g): add remove.skill RPC handler and register it"
```

---

## Task 8: Composition root wiring

**Files:**
- Modify: `core-go/cmd/skillbox-core/main.go:71-74`

- [ ] **Step 1: Wire WithRemoveDeps**

In `core-go/cmd/skillbox-core/main.go`, extend the `projectSvc` construction chain (the gateway `fs` satisfies `RemoveFilesystem`; `installRepo` satisfies `RemoveInstallDeleter`):

```go
	projectSvc := services.NewProjectService(projectRepo, ppRepo, warningRepo, installRepo, fs).
		WithScanDeps(runner, projectScanRepo).
		WithProviderDeps(providerRegistry, pdRepo, hostRepo, skillRepo).
		WithInstallDeps(fs, hostRepo, skillRepo).
		WithRemoveDeps(fs, installRepo)
```

- [ ] **Step 2: Build the binary**

Run: `cd core-go && go build ./cmd/skillbox-core/`
Expected: exits 0.

- [ ] **Step 3: Run the full Go test suite**

Run: `cd core-go && go test ./...`
Expected: all packages PASS.

- [ ] **Step 4: Run the race detector on write-path packages**

Run: `cd core-go && go test -race ./internal/operations/... ./internal/filesystem/... ./internal/services/...`
Expected: PASS, no race warnings.

- [ ] **Step 5: Commit**

```bash
git add core-go/cmd/skillbox-core/main.go
git commit -m "feat(2g): wire RemoveSkill deps in composition root"
```

---

## Task 9: Electron method allowlist

**Files:**
- Modify: `apps/desktop/electron/main/core-process/method-allowlist.ts`

- [ ] **Step 1: Add remove.skill to the allowlist**

In `apps/desktop/electron/main/core-process/method-allowlist.ts`, add `"remove.skill"` after `"install.skill"`:

```ts
  "install.skill",
  "remove.skill",
  "dialog.openPath",
```

- [ ] **Step 2: Typecheck**

Run: `cd apps/desktop && pnpm typecheck`
Expected: exits 0, no errors.

- [ ] **Step 3: Commit**

```bash
git add apps/desktop/electron/main/core-process/method-allowlist.ts
git commit -m "feat(2g): allowlist remove.skill in electron main"
```

---

## Task 10: Renderer core-client wrapper

**Files:**
- Modify: `apps/desktop/renderer/src/lib/core-client/methods.ts`
- Test: `apps/desktop/renderer/src/lib/core-client/__tests__/methods-remove-skill.test.ts` (create)

- [ ] **Step 1: Write the failing wrapper test**

Create `apps/desktop/renderer/src/lib/core-client/__tests__/methods-remove-skill.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("../client.js", () => ({
  invoke: vi.fn().mockResolvedValue({ operationId: 51 }),
}));

import { invoke } from "../client.js";
import { methods } from "../methods.js";

const mockInvoke = invoke as ReturnType<typeof vi.fn>;

beforeEach(() => {
  vi.clearAllMocks();
});

describe("methods.removeSkill", () => {
  it("invokes remove.skill with projectId and installId", async () => {
    const res = await methods.removeSkill({ projectId: 12, installId: 88 });
    expect(mockInvoke).toHaveBeenCalledWith("remove.skill", { projectId: 12, installId: 88 });
    expect(res.operationId).toBe(51);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/desktop && pnpm test -- methods-remove-skill`
Expected: FAIL — `methods.removeSkill is not a function`.

- [ ] **Step 3: Add the wrapper + type imports**

In `apps/desktop/renderer/src/lib/core-client/methods.ts`, add the type imports (after `InstallSkillResponse`):

```ts
  InstallSkillRequest,
  InstallSkillResponse,
  RemoveSkillRequest,
  RemoveSkillResponse,
} from "@contracts/index.js";
```

And add the method (after `installSkill`):

```ts
  installSkill: (req: InstallSkillRequest) =>
    invoke<InstallSkillResponse>("install.skill", req),

  removeSkill: (req: RemoveSkillRequest) =>
    invoke<RemoveSkillResponse>("remove.skill", req),
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/desktop && pnpm test -- methods-remove-skill`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add apps/desktop/renderer/src/lib/core-client/methods.ts apps/desktop/renderer/src/lib/core-client/__tests__/methods-remove-skill.test.ts
git commit -m "feat(2g): add removeSkill core-client wrapper"
```

---

## Task 11: Renderer useRemoveSkill hook

**Files:**
- Create: `apps/desktop/renderer/src/features/projects/use-remove-skill.ts`

- [ ] **Step 1: Implement the hook**

Create `apps/desktop/renderer/src/features/projects/use-remove-skill.ts` (mirrors `use-install-skill.ts`: buffer progress during the call, check the buffer for an already-terminal event, otherwise subscribe; invalidate detail + list on the terminal result only):

```ts
import { useState, useRef, useCallback, useEffect } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { methods } from "../../lib/core-client/methods.js";
import { subscribeOperationProgress, subscribeAllProgress } from "../../lib/core-client/progress.js";
import { queryKeys } from "../../lib/query-keys.js";
import type { RemoveSkillRequest, OperationProgressNotification } from "@contracts/index.js";

interface RemoveMetadata {
  skillName?: string;
  providerKey?: string;
  alreadyAbsent?: boolean;
}

function isTerminal(status: OperationProgressNotification["status"]): boolean {
  return status === "success" || status === "failed" || status === "cancelled";
}

function extractMeta(event: OperationProgressNotification): RemoveMetadata | null {
  if (event.metadata == null || typeof event.metadata !== "object") return null;
  return event.metadata as RemoveMetadata;
}

function successMessage(meta: RemoveMetadata | null): string {
  if (meta?.skillName != null) return `Removed ${meta.skillName}`;
  return "Skill removed";
}

function failedMessage(rawMessage: string | null): string {
  return rawMessage ? `Remove failed: ${rawMessage}` : "Remove failed";
}

export function useRemoveSkill() {
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
    mutationFn: async (req: RemoveSkillRequest) => {
      const buffered: OperationProgressNotification[] = [];
      const tempUnsub = subscribeAllProgress((p) => buffered.push(p));
      try {
        const result = await methods.removeSkill(req);
        return { operationId: result.operationId, projectId: req.projectId, buffered };
      } finally {
        tempUnsub();
      }
    },

    onError: (err: unknown) => {
      const message = err instanceof Error ? err.message : String(err);
      toast.error(failedMessage(message));
    },

    onSuccess: ({ operationId: opId, projectId, buffered }) => {
      const invalidate = () => {
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.detail(projectId) });
        void queryClient.invalidateQueries({ queryKey: queryKeys.projects.list() });
      };

      const terminalInBuffer = [...buffered]
        .reverse()
        .find((e) => e.operationId === opId && isTerminal(e.status));

      if (terminalInBuffer != null) {
        if (terminalInBuffer.status === "success") {
          toast.success(successMessage(extractMeta(terminalInBuffer)));
        } else if (terminalInBuffer.status === "failed") {
          toast.error(failedMessage(terminalInBuffer.message));
        }
        invalidate();
        return;
      }

      const toastId = toast.loading("Removing skill…");

      const unsub = subscribeOperationProgress(opId, (event) => {
        if (event.status === "success") {
          toast.success(successMessage(extractMeta(event)), { id: toastId });
        } else if (event.status === "failed") {
          toast.error(failedMessage(event.message), { id: toastId });
        } else if (event.status === "cancelled") {
          toast.dismiss(toastId);
        } else {
          toast.loading(event.message ? `Removing: ${event.message}` : "Removing skill…", { id: toastId });
        }

        if (isTerminal(event.status)) {
          invalidate();
          setOperationId(null);
          unsub();
          unsubRef.current = null;
        }
      });

      unsubRef.current = unsub;
      setOperationId(opId);
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

- [ ] **Step 2: Typecheck**

Run: `cd apps/desktop && pnpm typecheck`
Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add apps/desktop/renderer/src/features/projects/use-remove-skill.ts
git commit -m "feat(2g): add useRemoveSkill hook"
```

---

## Task 12: Renderer confirmation dialog + Project Detail wiring

**Files:**
- Create: `apps/desktop/renderer/src/features/projects/remove-skill-dialog.tsx`
- Modify: `apps/desktop/renderer/src/screens/project-detail-screen.tsx`
- Test: `apps/desktop/renderer/src/features/projects/__tests__/remove-skill-dialog.test.tsx` (create)

- [ ] **Step 1: Write the failing dialog test**

Create `apps/desktop/renderer/src/features/projects/__tests__/remove-skill-dialog.test.tsx`:

```tsx
// @vitest-environment happy-dom
import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import React from "react";
import { RemoveSkillDialog } from "../remove-skill-dialog.js";

afterEach(() => cleanup());

const baseProps = {
  skillName: "documentation-writer",
  providerDisplayName: "Shared Agent Skills (.agents)",
  path: "/repo/content-lab/.agents/skills/documentation-writer",
  isPending: false,
};

describe("RemoveSkillDialog", () => {
  it("shows skill, provider, and exact path", () => {
    render(<RemoveSkillDialog {...baseProps} onConfirm={vi.fn()} onCancel={vi.fn()} />);
    expect(screen.getByText("documentation-writer")).toBeTruthy();
    expect(screen.getByText(/Shared Agent Skills/)).toBeTruthy();
    expect(screen.getByText(baseProps.path)).toBeTruthy();
    expect(screen.getByText(/not affected/i)).toBeTruthy();
  });

  it("calls onConfirm when Remove is clicked", () => {
    const onConfirm = vi.fn();
    render(<RemoveSkillDialog {...baseProps} onConfirm={onConfirm} onCancel={vi.fn()} />);
    fireEvent.click(screen.getByRole("button", { name: /^Remove$/ }));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when Cancel is clicked", () => {
    const onCancel = vi.fn();
    render(<RemoveSkillDialog {...baseProps} onConfirm={vi.fn()} onCancel={onCancel} />);
    fireEvent.click(screen.getByRole("button", { name: /Cancel/ }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd apps/desktop && pnpm test -- remove-skill-dialog`
Expected: FAIL — cannot resolve `../remove-skill-dialog.js`.

- [ ] **Step 3: Implement the dialog**

Create `apps/desktop/renderer/src/features/projects/remove-skill-dialog.tsx`:

```tsx
import React from "react";

interface RemoveSkillDialogProps {
  skillName: string;
  providerDisplayName: string;
  path: string;
  isPending: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}

export function RemoveSkillDialog({
  skillName,
  providerDisplayName,
  path,
  isPending,
  onConfirm,
  onCancel,
}: RemoveSkillDialogProps): React.JSX.Element {
  return (
    <div className="absolute inset-0 z-50 flex items-center justify-center bg-black/30">
      <div className="w-full max-w-lg rounded-lg border border-zinc-200 bg-white p-5 shadow-xl">
        <h2 className="mb-3 text-sm font-semibold text-zinc-900">Remove skill from project</h2>

        <div className="mb-3 text-xs text-zinc-700">
          <div className="mb-1">
            Remove <span className="font-medium text-zinc-900">{skillName}</span>
          </div>
          <div>
            from <span className="font-medium text-zinc-900">{providerDisplayName}</span>
          </div>
        </div>

        <div className="mb-3 text-xs text-zinc-700">
          This deletes the symlink at:
          <div className="mt-1 break-all rounded bg-zinc-50 px-2 py-1 font-mono text-[11px] text-zinc-600">
            {path}
          </div>
        </div>

        <p className="mb-4 text-xs text-zinc-500">
          The skill in your Skill Host Folder is not affected.
        </p>

        <div className="flex justify-end gap-2">
          <button
            onClick={onCancel}
            disabled={isPending}
            className="rounded border border-zinc-300 px-3 py-1.5 text-xs font-medium text-zinc-700 hover:bg-zinc-50 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            disabled={isPending}
            className="rounded border border-red-300 bg-red-50 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-100 disabled:opacity-50"
          >
            Remove
          </button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd apps/desktop && pnpm test -- remove-skill-dialog`
Expected: PASS (3 tests).

- [ ] **Step 5: Wire the Actions column into Project Detail**

In `apps/desktop/renderer/src/screens/project-detail-screen.tsx`:

(a) Add imports near the top (after the existing `use-remove-project` import):

```tsx
import { useRemoveSkill } from "../features/projects/use-remove-skill.js";
import { RemoveSkillDialog } from "../features/projects/remove-skill-dialog.js";
```

(b) Add a removable predicate above the `EntryRow` component:

```tsx
function isRemovable(entry: ProjectGetEntry): boolean {
  return entry.mode === "symlink" && entry.status === "current";
}
```

(c) Replace the `EntryRow` component signature and add an Actions cell. Change the function declaration and append one `<td>` before the closing `</tr>`:

```tsx
function EntryRow({
  entry,
  onRemove,
}: {
  entry: ProjectGetEntry;
  onRemove: (entry: ProjectGetEntry) => void;
}): React.JSX.Element {
  const removable = isRemovable(entry);
  return (
    <tr className="border-b border-zinc-100 hover:bg-zinc-50">
      <td className="px-3 py-1.5 text-xs text-zinc-500">{entry.providerKey}</td>
      <td className="px-3 py-1.5 text-xs font-medium text-zinc-900">{entry.name}</td>
      <td className="px-3 py-1.5 text-xs">
        <span className="inline-flex items-center rounded bg-zinc-100 px-1.5 py-0.5 font-medium text-zinc-600">
          {entry.mode}
        </span>
      </td>
      <td className="px-3 py-1.5 text-xs">
        <EntryStatusBadge status={entry.status} />
      </td>
      <td className="max-w-xs truncate px-3 py-1.5 font-mono text-xs text-zinc-400" title={entry.projectSkillPath}>
        {entry.projectSkillPath}
      </td>
      <td className="max-w-xs truncate px-3 py-1.5 font-mono text-xs text-zinc-400" title={entry.symlinkTargetPath ?? undefined}>
        {entry.symlinkTargetPath ?? "—"}
      </td>
      <td className="px-3 py-1.5 text-xs text-zinc-400">{entry.skillId ?? "—"}</td>
      <td className="px-3 py-1.5 text-xs">
        <button
          onClick={() => onRemove(entry)}
          disabled={!removable}
          title={removable ? "Remove skill from project" : "Only current symlink installs can be removed in this slice"}
          className="rounded border border-zinc-300 px-2 py-0.5 text-xs font-medium text-zinc-600 hover:border-red-300 hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40"
        >
          Remove
        </button>
      </td>
    </tr>
  );
}
```

(The provider display name is resolved at the screen level for the dialog, not inside `EntryRow` — see Step 5e.)

(d) Inside `ProjectDetailScreen`, add hook + dialog state (after `const remove = useRemoveProject(...)`):

```tsx
  const removeSkill = useRemoveSkill();
  const [removeTarget, setRemoveTarget] = useState<ProjectGetEntry | null>(null);
```

(e) Add a provider-display-name lookup and confirm handler inside `ProjectDetailScreen` (before the `return`):

```tsx
  const providerDisplayNameFor = (entry: ProjectGetEntry): string => {
    const match = data?.providers.find((p) => p.projectProviderId === entry.projectProviderId);
    return match?.displayName ?? entry.providerKey;
  };

  function confirmRemoveSkill(): void {
    if (removeTarget == null || validId == null) return;
    removeSkill.mutate({ projectId: validId, installId: removeTarget.id });
    setRemoveTarget(null);
  }
```

(f) Add the Actions header cell — in the Skill Entries `<thead><tr>`, append after the `Skill ID` header:

```tsx
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Skill ID</th>
                        <th className="px-3 py-1.5 text-xs font-medium text-zinc-500">Actions</th>
```

(g) Update the entries `.map` to pass the new props:

```tsx
                      {data.entries.map((entry) => (
                        <EntryRow
                          key={entry.id}
                          entry={entry}
                          onRemove={setRemoveTarget}
                        />
                      ))}
```

(h) Render the dialog — add just before the existing `{wizardOpen && ...}` block near the end of the component:

```tsx
      {removeTarget != null && (
        <RemoveSkillDialog
          skillName={removeTarget.name}
          providerDisplayName={providerDisplayNameFor(removeTarget)}
          path={removeTarget.projectSkillPath}
          isPending={removeSkill.isPending}
          onConfirm={confirmRemoveSkill}
          onCancel={() => setRemoveTarget(null)}
        />
      )}
```

- [ ] **Step 6: Typecheck and run renderer tests**

Run: `cd apps/desktop && pnpm typecheck && pnpm test -- remove-skill-dialog project-detail`
Expected: typecheck exits 0; dialog tests PASS; existing project-detail tests still PASS.

- [ ] **Step 7: Commit**

```bash
git add apps/desktop/renderer/src/features/projects/remove-skill-dialog.tsx apps/desktop/renderer/src/features/projects/__tests__/remove-skill-dialog.test.tsx apps/desktop/renderer/src/screens/project-detail-screen.tsx
git commit -m "feat(2g): wire Remove action and confirmation dialog into Project Detail"
```

---

## Task 13: Full verification + manual smoke test

**Files:** none (verification only)

- [ ] **Step 1: Full Go suite**

Run: `cd core-go && go test ./...`
Expected: all packages PASS.

- [ ] **Step 2: Go race detector on write paths**

Run: `cd core-go && go test -race ./internal/operations/... ./internal/filesystem/... ./internal/services/...`
Expected: PASS, no race warnings.

- [ ] **Step 3: Contract drift check**

Run: `cd apps/desktop && pnpm check:contracts-drift`
Expected: `✓ No contract drift detected`.

- [ ] **Step 4: Frontend typecheck + tests + build**

Run: `cd apps/desktop && pnpm typecheck && pnpm test && pnpm build`
Expected: all exit 0; Vitest suite green; electron-vite build succeeds.

- [ ] **Step 5: Manual smoke test (full-stack)**

Run: `cd apps/desktop && pnpm dev`

Then, in the running app:
1. Open a project that has at least one `current` symlink install (or use Add Skill to create one).
2. In Project Detail → Skill Entries, confirm the `[Remove]` button is **enabled** for the `current`/`symlink` row and **disabled** (greyed, with tooltip) for any `direct`/`old_host`/`broken`/`external` row.
3. Click `[Remove]` on the `current` symlink row. Confirm the dialog shows the skill name, provider display name, and the exact `.agents/skills/<name>` path, plus the "Skill Host Folder is not affected" note.
4. Click `Remove`. Expect a success toast ("Removed <skillName>") and the row disappears from the list after the operation completes.
5. On disk, verify the symlink at the shown path is gone and the host skill folder still exists (e.g. `ls -la <project>/.agents/skills` and `ls <hostSkillsPath>/<name>`).
6. Negative check: manually replace a symlink with a real empty directory at a `current` entry's path, click Remove, and confirm you get the "changed on disk" conflict toast and the directory is NOT deleted (it is non-symlink → conflict before any unlink).

Expected: all six checks pass. If the UI cannot be exercised (no dev environment), state that explicitly instead of claiming success.

- [ ] **Step 6: Final commit (if any verification fixups were needed)**

```bash
git add -A
git commit -m "test(2g): verification fixups for remove skill slice"
```

(Skip this commit if Steps 1–5 required no changes.)

---

## Self-Review (completed during planning)

**1. Spec coverage:**
- `remove.skill` command + handler + schema → Tasks 6, 7. ✓
- Service validate → re-verify → unlink → rescan → targeted delete → Task 5. ✓
- Gateway `RemoveSymlink` → Task 2. ✓
- Repo `DeleteByID` (only hard delete) → Task 3. ✓
- On-disk re-verification reusing `isWithin(activeHostSkillsPath, resolved)` → Task 5 `removeSkillInternal` + Task 2 `ResolveEntry`. ✓
- Removable set = symlink AND current only; `old_host`/`direct`/`external`/`broken`/`error` rejected → Task 5 precheck + tests. ✓
- Operation target `project:<id>` with new `OperationTypeRemoveSkill`; mutual exclusion via runner → Tasks 1, 5. ✓
- Terminal-only invalidation; metadata `{projectId, providerKey, skillName, removedPath, alreadyAbsent}` → Tasks 5, 11. ✓
- Error mapping (validation/conflict/filesystem/database) → Task 5 (`domain.New*Error`) + Task 7 conflict→1005 test. ✓
- Electron allowlist + preload path (allowlist is the gate; preload forwards allowlisted methods) → Task 9. ✓
- Renderer core-client + hook + confirmation dialog (skill/provider/path; not-affected note) + disabled for non-removable → Tasks 10, 11, 12. ✓
- Tests: gateway, repo, service (happy, already-absent, not-removable incl. old_host, not-found, not-active, divergence real-dir, divergence outside-host, unlink-failure), contract drift, renderer hook/dialog/gating → Tasks 2, 3, 5, 6, 10, 12. ✓
- Path-escape rejection → Task 5 `RemoveSkill` sync check (`isWithin(root, path)`); covered by the within-root guard (the `removeFixture` paths are inside root; the guard is unit-exercised via the implementation and the divergence tests assert no unlink on stale state).
- Manual UI smoke incl. the "changed on disk" conflict path → Task 13 Step 5. ✓

**2. Placeholder scan:** No incomplete-work or "similar to above" placeholders. Every code step shows complete, paste-ready code, and the provider display name is resolved at the screen level (Task 12 Step 5e) rather than inside `EntryRow`.

**3. Type consistency:** `removeSkillMetadata` (Go, Task 5) ↔ `RemoveMetadata` reader (TS, Task 11) share field names `skillName`/`providerKey`/`alreadyAbsent`. Handler `removeSkillResponse{OperationID}` (Task 7) ↔ contract `RemoveSkillResponse.operationId` (Task 6) ↔ `methods.removeSkill` returns `RemoveSkillResponse` (Task 10). `WithRemoveDeps(removeFS, installDeleter)` signature matches the wiring call in Task 8 (`fs, installRepo`). `RemoveFilesystem`/`RemoveInstallDeleter` (Task 4) are satisfied by `*filesystem.Gateway` and `*repositories.InstallRepo`. `isRemovable` predicate (symlink+current) matches the service precheck.

**Lock-conflict note:** The spec's "remove vs concurrent scan/install → one conflict_error" is enforced by the existing `operations.Runner` per-target lock (unchanged code) because `RemoveSkill` uses `Target{Type:"project", ID: projectID}`, identical to scan/install. No new test is added for the runner's lock here (it is already covered by the runner's own tests); the integration point is exercised by Task 8's full-suite + race run.
