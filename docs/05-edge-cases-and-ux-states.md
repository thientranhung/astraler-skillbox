# Edge Cases And UX States

This document describes non-ideal situations that Skillbox must handle for a
tighter UI/UX. This is not implementation detail; the goal is to define states,
risks, and appropriate UI responses.

## 1. Skill Host Folder States

### Skill Host Folder Not Configured

Situation:

- User opens the app for the first time.
- Database does not yet have a `skill_host_folder`.

UI should:

- Display onboarding so the user can select a Skill Host Folder.
- Not allow skill installs until configuration is complete.

### Skill Host Folder Does Not Exist

Situation:

- Folder has been moved, renamed, unmounted, or deleted.

UI should:

- Display a clear warning on Dashboard and Settings.
- Allow the user to select a new folder.
- Allow the user to view the list of projects/installs that may be affected.

### Skill Host Folder Missing `.agents/skills`

Situation:

- User selects a new folder or a folder that has not been normalized.

UI should:

- Explain that Skillbox needs a `.agents/skills` structure.
- Allow the user to create this structure via a clear action.

### Skill Host Folder Is Empty

Situation:

- `.agents/skills` exists but contains no skills.

UI should:

- Display an empty state in Host Skills.
- Offer `Open Skill Host Folder` in the current UI. Add/import remains
  deferred.

### Skill Host Folder Has No Read/Write Permission

Situation:

- App cannot scan or import/update skills.

UI should:

- Distinguish read errors from write errors.
- Allow the user to select a different folder or open the folder in the file
  manager to fix permissions.

### Skill Host Folder Changed To New Path

Situation:

- User changes the source of truth in Settings.

UI should:

- Scan the new folder before applying.
- Display projects/installs that point to the old folder.
- Offer to relink symlinks if the user wants.

### Skill Host Folder on External or Synced Drive

Situation:

- Folder is on an external drive, iCloud, Dropbox, Google Drive, NAS, etc.

UI should:

- Not block.
- Warn if the path is temporarily unavailable.
- Allow rescan when the folder becomes available again.

## 2. Project States

### Project Path Does Not Exist

Situation:

- Project has been moved, renamed, unmounted, or deleted.

UI should:

- Display a project warning in Projects and Dashboard.
- Allow the user to update the path or remove the project from the database.

### Project Has No Provider Folders

Situation:

- User adds a new project folder but no convention has been detected.

UI should:

- Display a `No provider detected` state.
- Allow the user to select a provider/convention to set up if the app supports
  creating the structure.

### Project Has Multiple Provider Folders

Situation:

- Project has Claude and `.agents`, or multiple conventions coexisting.

UI should:

- Display provider badges clearly.
- When installing a skill, require the user to select a provider target.

### Project Has Manual Skills

Situation:

- A skill exists in the provider folder but has no Skillbox install metadata.

UI should:

- Classify it as `direct`.
- Not claim it as a managed install.
- Allow the user to adopt/import it into Skillbox if this feature is supported
  later.

### Project Has Skill With Duplicate Name Across Providers

Situation:

- `skill-a` exists in both the Claude folder and `.agents/skills`.

UI should:

- Display by provider scope, not merge blindly.
- In Project Detail, skill rows should clearly show the provider.

## 3. Global Skill States

### Global Provider Location Not Configured

Situation:

- Provider has a global level but Skillbox does not yet know the global path for
  that provider.

UI should:

- Display the provider global state as `not configured`.
- Allow the user to configure the path if the provider adapter supports global
  location.

### Global Provider Location Does Not Exist

Situation:

- Global path previously existed but has been moved, unmounted, or deleted.

UI should:

- Display a warning in Dashboard and Global Skills.
- Allow the user to update the path, rescan, or disable the location.

### Global Provider Location Exists But Is Empty

Situation:

- Provider global folder exists but contains no skills/global entries.

UI should:

- Display an empty state by provider.
- Not treat this as an error.

### Global Skill Unmanaged/Direct

Situation:

- A global entry exists but is not managed by Skillbox.

UI should:

- Classify it as `direct`.
- Clearly display that this is a global-level entry.
- Not automatically remove or relink it.

### Global Skill Overlaps With Project-Level Skill

Situation:

- The same skill name exists at both global level and project level.

UI should:

- Display a warning/informational state so the user knows there may be overlap.
- Not automatically decide precedence since provider behavior may differ.

### Global Symlink Is Broken or External

Situation:

- Global entry is a broken symlink or points outside the Skill Host Folder.

UI should:

- Classify similarly to project install: `broken_symlink` or
  `external_symlink`.
- Allow the user to relink, remove, or leave as-is.

## 4. Install States

### Valid Symlink

Situation:

- Project skill is a symlink pointing to the Skill Host Folder.

UI should:

- Display mode `symlink`.
- Display the source path.
- Allow opening the source folder and the project folder.

### Broken Symlink

Situation:

- Symlink target no longer exists.

UI should:

- Display a warning in Project Detail.
- Allow the user to relink or remove.

### Symlink Points to Old Skill Host Folder

Situation:

- User has changed the Skill Host Folder, but the project still has a symlink
  to the old host.

UI should:

- Classify as symlink but mark as `old host`.
- Allow the user to relink to the current Skill Host Folder.

### Symlink Points Outside Skill Host Folder

Situation:

- Skill in the project is a symlink but the target is not in the current Skill
  Host Folder.

UI should:

- Classify as `external symlink`.
- Not auto-fix.
- Allow the user to remove, relink, or leave as-is.

### Rsync/Copy States

> **Deferred.** Rsync/copy mode is not implemented. The states `rsync/copy
> current` and `rsync/copy outdated` are not in the current release.

### Direct Install

Situation:

- Skill is a regular folder with no Skillbox metadata.

UI should:

- Display mode `direct`.
- Not display update/sync actions as for a managed install.

### Target Folder Already Exists When Installing

Situation:

- User installs a skill but the provider folder already has an entry with the
  same name.

UI should:

- Block overwriting by default.
- Allow the user to choose replace, skip, or cancel.
- If replacing, require clear confirmation since this is a destructive action.

### Conflict When Switching Mode

> **Deferred.** Switching install mode is not implemented. This case does not
> apply in the current release.

## 5. Fetch And Update States

### Skill Has No Source Metadata

Situation:

- A local/manual skill does not know where its upstream is.

UI should:

- Display source as `local/manual`.
- Disable Fetch for that skill or allow the user to configure a source.

### GitHub Repo Unreachable

Situation:

- Repo has been deleted, is private, has wrong URL, or is missing auth.

UI should:

- Display fetch errors per skill.
- Not corrupt the fetch state of other skills.
- Allow the user to fix the source metadata.

### Vercel Skills Fetch Fails

Situation:

- Vercel skills source is temporarily unreachable or returns an invalid response.

UI should:

- Display a recoverable error.
- Allow retry.

### Network Offline

Situation:

- A future upstream skill fetch/update action cannot reach the network.

UI should:

- Display a scoped fetch warning in the relevant future update surface.
- Keep the last known update state.
- Allow retry when the network comes back.

### Upstream Has Update

Situation:

- Fetch detects a new version/commit.

UI should:

- Display the skill in the relevant future update surface.
- Display affected projects and install modes.

### Upstream Has No Update

Situation:

- Skill is at the latest version.

UI should:

- Display `up to date` state.
- Not add it to the action list.

### Local Skill Has Been Modified From Upstream

Situation:

- Skill in the Skill Host Folder has local modifications.

UI should:

- Not auto-overwrite.
- Display a state that needs review.
- Allow the user to choose to keep local, overwrite, or create a snapshot
  depending on the final design.

### Update Affects Many Symlinked Projects

Situation:

- A skill is symlinked and shared by many projects.

UI should:

- Before updating, display affected projects.
- After updating, symlinked projects are considered to have received the change
  immediately.

### Rsync/Copy Projects Need Sync After Update

> **Deferred.** Rsync/copy mode is not implemented. This case does not apply in
> the current release.

## 6. Provider States

### Provider Clearly Detected

Situation:

- Project has folder/path matching the provider adapter's convention.

UI should:

- Display a provider badge/icon.
- Allow installing skills into that provider.

### Provider Convention Not Yet Supported

Situation:

- Project shows signs of using a provider but Skillbox has no adapter for it.

UI should:

- Display as `unsupported provider`.
- Not write to paths it does not understand.

### Provider Folder Exists But Has Unexpected Format

Situation:

- Folder name matches the convention but the internal structure is not as
  expected.

UI should:

- Display a warning.
- Allow the user to view the path and rescan.

### Claude and `.agents` Coexist

Situation:

- Project uses both the Claude-specific convention and the shared `.agents`
  convention.

UI should:

- Separate provider scopes clearly.
- Add Skill flow must require selecting a provider target.

## 7. Add Skill Wizard States

### 0 Installable Providers (Empty State)

Situation:

- Project has no valid provider for install (no provider with
  `detection_status = detected/configured` and
  `provider_definitions.status = supported/experimental` and a resolvable
  `skills_path`).

UI should:

- Display empty state card in wizard: "No provider is ready for install."
- Offer CTA "Scan project" as primary action.
- When user clicks "Scan project", call `useScanProject` and close the wizard.
- Not display the tab strip, skill list, or Install button.

### Skill Already Installed at Active Tab's Provider

Situation:

- A skill in the list already has an install record at the active tab's provider.

UI should:

- Display that skill's checkbox as disabled + opacity-50.
- Show an "Installed" badge next to the skill name.
- Not allow the user to check that skill at the current tab.
- The skill may still be selected at another provider's tab if not yet installed
  there (installed-state is per-provider, not globally disabled).

### Switching Tab Resets Selection

Situation:

- User has checked some skills in tab A, then switches to tab B.

UI should:

- Clear all `selectedSkillIds` when the tab changes.
- Clear the install error (if any) when the tab changes.
- Tab B starts with an empty selection, not inheriting tab A's choices.

### Provider Is Experimental

Situation:

- The wizard tab corresponds to a provider with
  `provider_definitions.status = experimental`.

UI should:

- Display an "experimental" badge in the tab header next to the display name.
- Still allow normal install (experimental does not block install).
- Not require an extra confirmation modal just for experimental.

### Install Error (e.g. conflict_error 1005)

Situation:

- Skillbox returns an error when the user clicks Install (e.g. target folder
  already exists, permission denied, conflict_error code 1005, …).

UI should:

- Keep the wizard open; do not close after an error.
- Display an error row in the footer (text-red-600) just above Cancel/Install.
- Allow the user to fix the selection or click Cancel to exit.
- Clear the error row if the user switches tabs or changes selection.
- Not partial-update the database if the install operation fails.

## 8. Database And App State

### Database Does Not Exist

Situation:

- User opens the app for the first time, or the database has been deleted.

UI should:

- Create a new database.
- Run First-Time Setup.

### Database Corrupt

Situation:

- SQLite file is unreadable or has schema errors.

UI should:

- Not crash silently.
- Display a blocking error.
- Allow the user to back up/export the corrupt file if possible.

### Database Diverged From Filesystem

Situation:

- Database records an install but the filesystem has been modified outside the
  app.

UI should:

- Rescan to reconcile.
- Treat the filesystem as the true state.

### Filesystem Has Skill That Database Does Not Know About

Situation:

- User manually copies a skill into a project or Skill Host Folder.

UI should:

- Detect it on scan and display it.
- For project installs, classify as `direct` if no metadata.
- For Skill Host Folder, add skill to library after scan.

### Schema Migration

Situation:

- A new app version requires a SQLite schema change.

UI should:

- Run migration before opening the main app.
- If migration fails, display a clear error and not continue writing new data.

## 8. UI/UX States

### Empty State

Applies to:

- No Skill Host Folder configured.
- Skill Host Folder empty.
- No projects.
- No global skills.
- Project has no skills.

UI should:

- Clearly state the current condition.
- Offer a single primary next action.

### Auto-Scan on Mount

Project Detail, Global Skills, and Plugins screens automatically trigger a scan
on mount if data is stale:

- **Trigger condition**: `lastScannedAt == null` (never scanned) OR
  `Date.now() - lastScannedAt > 10 minutes`.
- **Anti double-trigger**: 3 guards — hook-level `isPending/operationId` check;
  session-level `sessionAutoScanRegistry` (Set keyed by
  `"auto-scan:<target>:<id>"`); component-level `useRef` flag.
- **Toast policy**: auto-scan uses `silent: true` → no loading/success toast.
  Error toast still displays. Manual scan uses `silent: false` (default) →
  full toast.
- **Manual Scan button**: always present, not blocked by auto-scan.
- **Projects list**: does NOT auto-scan (many projects; avoids an operation
  storm).

### Loading/Scanning State

Applies to:

- Scanning Skill Host Folder.
- Scanning project.
- Scanning global locations.
- Fetching update.

UI should:

- Display progress or busy state.
- Not allow a duplicate dangerous operation on the same target.

### Confirm Destructive Action

Applies to:

- Removing a skill from a project.
- Replacing an existing folder.
- Changing Skill Host Folder when there are affected symlinks.
- Deleting a project from the database.

UI should:

- Display the affected objects.
- Require clear confirmation.

### Recoverable Warning

Applies to:

- Missing path.
- Broken symlink.
- Fetch failure.
- Unsupported provider.

UI should:

- Not block the whole app.
- Offer a specific action such as rescan, retry, relink, choose folder, remove.

### Blocking Error

Applies to:

- Corrupt database.
- Cannot read Skill Host Folder.
- Cannot write when user is installing/updating.

UI should:

- Block the related action.
- Explain the error and the next step.

### Impact Preview

Applies to:

- Updating a skill in the Skill Host Folder.
- Changing Skill Host Folder.
- Switching install mode.

UI should:

- Display affected projects/providers/skills before the user confirms.

### Quick Actions

Error states should have quick actions:

- Open folder.
- Rescan.
- Retry.
- Relink.
- Sync.
- Remove from database.
- Configure source.
- Open folder.
