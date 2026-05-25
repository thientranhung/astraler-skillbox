# Slice 2B: Project Utilities — Design

Date: 2026-05-25
Status: approved by PM
Scope: small project management utilities only: open a project folder in Finder and remove a project from Skillbox without touching project files.

## 1. Purpose

Slice 2A lets users add, list, inspect, and scan projects. Slice 2B closes the basic management loop:

- User can open a tracked project folder in Finder from the list or detail view.
- User can remove a project from Skillbox when it is no longer relevant.

The core safety rule is unchanged: **Skillbox must not delete or modify any file inside the project folder.**

## 2. In Scope

- Add `project.remove { projectId } -> { removed: true }`.
- Implement remove as a soft-remove by setting `projects.status = removed`.
- Hide removed projects from `project.list`.
- Treat removed projects as not found for `project.get` and `project.scan`.
- Re-adding the same project path revives the existing row to `active` instead of creating a duplicate.
- Add an Electron-native open-folder action for an existing project path.
- Add Open Folder and Remove actions in Projects list and Project detail.
- Confirm remove in the UI with copy that makes clear files on disk are not deleted.

## 3. Out Of Scope

- Hard delete of project rows or related scan history.
- Any filesystem write into project folders.
- Provider-specific install/uninstall actions.
- Provider setup, relink, sync, or update flows.
- Bulk project remove.
- Custom modal design system work. A native `window.confirm` is acceptable for this slice.

## 4. UX

Projects list row actions:

- `Scan`
- `Open Folder`
- `Remove`

Project detail header actions:

- `Scan`
- `Open Folder`
- `Remove`

Remove confirmation text:

```text
Remove this project from Skillbox? Files on disk will not be deleted.
```

After successful remove:

- From list: invalidate and refresh the projects list.
- From detail: navigate back to `/projects`.

Open Folder errors should surface through the existing error/toast path used by other core-client mutations.

## 5. Data And API Rules

The existing `projects.status` enum already includes `removed`, so no migration is required.

Repository behavior:

- `List` filters out `status = 'removed'`.
- `GetByID` returns nil for removed rows.
- `UpsertByPath` revives a removed row by setting `status = 'active'`, updating `name`, and refreshing `updated_at`.
- A new remove method updates active/missing/unreadable projects to `removed`.

Service behavior:

- `RemoveProject` validates the project exists and is not removed.
- Unknown or already removed project IDs return `validation_error`.
- Remove returns `{ removed: true }`.

Electron behavior:

- Open folder is handled in Electron main via `shell.openPath(path)`.
- Renderer may only call it through the allowlisted core bridge method.
- If Electron returns an error string, expose it as an operation failure.

## 6. Acceptance Criteria

- Removing a project never deletes or changes files in the project folder.
- Removed projects disappear from Projects list.
- Removed project detail and scan requests return validation errors.
- Re-adding the same path brings the project back as active.
- Open Folder works from list and detail on macOS.
- Existing Slice 2A scan/list/detail behavior remains green.
