# User Flows

This document describes the main user flows in Skillbox. GUI is the primary
interface.

## 1. First-Time Setup

Goal: configure Skillbox for the first time to establish which folder is the
source of truth for skills.

Flow:

```text
User opens Skillbox for the first time
  -> Skillbox prompts the user to select a Skill Host Folder
  -> User selects a folder on the machine
  -> Skillbox checks for or creates the .agents/skills structure
  -> Skillbox scans existing skills in that folder
  -> Skillbox saves the configuration to the database
  -> User arrives at Dashboard
```

Result:

- App has a configured Skill Host Folder.
- That folder becomes the source of truth for skills.
- Dashboard has initial data from the first scan.

## 2. Add Project

Goal: bring a project into Skillbox to manage its skills.

Flow:

```text
User selects Add Project
  -> User selects a project folder
  -> Skillbox scans provider conventions within the project
  -> Skillbox scans skills currently present in the provider folders
  -> Skillbox saves the project to the database
  -> User is taken to Project Detail
```

Result:

- Project is tracked by Skillbox.
- App knows which providers the project has.
- App knows which skills are in the project and their install mode if detected.

## 3. Scan Project

Goal: update the true state of a project from the filesystem.

Flow:

```text
User opens Project Detail
  -> Skillbox checks lastScannedAt: if null or > 10 minutes -> silent auto-scan (no toast)
  -> User selects Scan or Rescan (manual, always with toast)
  -> Skillbox reads the project folder
  -> Skillbox detects provider folders
  -> Skillbox reads skill entries in each provider
  -> Skillbox classifies install mode
  -> Skillbox updates the database
  -> UI displays the new state
```

Install mode:

- `symlink`: skill in project is a symlink to the Skill Host Folder.
- `direct`: skill exists in project but is not managed by Skillbox.

Result:

- Project Detail reflects the current filesystem state.
- Changes made outside the app are brought into the database.

## 4. Scan Global Skills

Goal: find out which skills/config exist at the provider global level on the
machine.

Flow:

```text
User opens Global Skills
  -> Skillbox checks locations[].lastScannedAt: if any null or oldest > 10 minutes -> silent auto-scan
  -> User selects Scan Global (manual, with toast)
  -> Skillbox reads known/configured global provider locations
  -> Skillbox detects global skills/entries
  -> Skillbox classifies mode/status
  -> Skillbox updates the database
  -> UI displays global skill state by provider
```

Result:

- User knows which skills exist at the provider global level.
- App distinguishes global skills from project-level skills.
- Warnings are created if a global entry is missing, broken, external, or
  unmanaged.

Phase 1:

- Global Skills is a scan/visibility/remediation surface.
- No `Install Skill To Global Location` flow yet.
- Add Skill flow only targets project providers.

## 5. Install Skill To Project

Goal: install a skill from the Skill Host Folder into a project.

Flow (happy path — at least one installable provider):

```text
User opens Project Detail
  -> User selects Add Skill
  -> Add Skill Wizard opens, displaying a tab strip
     (each tab = one installable provider: tab header includes ProviderIcon + display name
      + short skills path badge + "experimental" badge if the provider is experimental)
  -> User selects the tab of the provider they want to install into
  -> Wizard displays the skill list from the Skill Host Folder for that tab
     (already installed at this provider: checkbox disabled + opacity-50 + "Installed" badge)
  -> User checks one or more skills that are not yet installed
  -> Footer displays the path hint for the active provider, plus Cancel and Install buttons
  -> User clicks Install
  -> Skillbox installs the skill into the provider folder of the active tab
  -> Skillbox writes install metadata to the database
  -> Wizard closes; UI updates the installed skills list
```

Flow (edge case — no installable provider):

```text
User opens Project Detail
  -> User selects Add Skill
  -> Add Skill Wizard opens with no tabs
  -> Wizard shows empty state: "No provider is ready for install."
  -> CTA "Scan project" → calls useScanProject; wizard closes after triggering
```

Notes when using the wizard:

- Switching tabs resets the selection (selectedSkillIds is cleared) and removes
  any install error.
- Clicking Install only installs into the provider of the active tab (1 submit =
  1 provider).

Result:

- Skill appears in the project provider folder.
- Database records the project, provider, skill, install mode, and source path.

## 6. Deferred: Fetch Skill Updates

Goal: check upstream to find out which skills have new versions.

Status: deferred from the current shipped UI. There is no standalone Updates
route and no Fetch All button in the current renderer. Keep this flow as product
intent for future source-integration work.

Flow:

```text
User opens a future Host Skills / Skill Detail update surface
  -> User starts a future upstream fetch action
  -> Skillbox reads the source metadata of the skill
  -> Skillbox checks GitHub or Vercel skills
  -> Skillbox saves fetch results to the database
  -> UI displays skills with updates
```

Result:

- User knows which skills have new versions.
- UI can display affected projects for each skill.

## 7. Deferred: Update Skill Host Folder

Goal: update the skill copy in the Skill Host Folder from upstream.

Status: deferred from the current shipped UI.

Flow:

```text
User opens a future Host Skills / Skill Detail update surface
  -> User selects Update skill
  -> Skillbox displays affected projects
  -> User confirms the update
  -> Skillbox updates the skill in the Skill Host Folder
  -> Skillbox updates the version/source metadata
  -> UI refreshes the update status
```

Result:

- Skill in the Skill Host Folder is updated.
- Projects using `symlink` receive the change immediately.

## 10. Remove Skill From Project

Goal: uninstall a skill from a project/provider.

Flow:

```text
User opens Project Detail
  -> User selects an installed skill
  -> User selects Remove
  -> Skillbox confirms the action
  -> Skillbox removes the symlink or folder copy from the provider folder
  -> Skillbox updates the database
  -> UI removes the skill from the installed skills list
```

Result:

- Skill is no longer installed in that project/provider.
- The original skill in the Skill Host Folder is unaffected.

## 11. Deferred: Add Skill To Skill Host Folder

Goal: bring a new skill into the source of truth.

Status: deferred from the current shipped UI. Host Skills can scan and open the
Skill Host Folder, but it does not currently include an add/import workflow.

Flow:

```text
User opens a future Host Skills import surface
  -> User selects Add / Import Skill
  -> User selects source: GitHub, Vercel skills, local/manual
  -> Skillbox imports the skill into the Skill Host Folder
  -> Skillbox saves the source metadata to the database
  -> Skillbox rescans Host Skills
```

Result:

- New skill appears in Host Skills.
- Skill can be installed to projects via symlink.

## 12. Change Skill Host Folder

Goal: switch the source of truth to a different folder.

Flow:

```text
User opens Settings
  -> User selects Change Skill Host Folder
  -> User selects a new folder
  -> Skillbox scans the new folder
  -> Skillbox warns if current installs point to the old folder
  -> User confirms
  -> Skillbox updates config/database
```

Result:

- Skillbox uses the new Skill Host Folder as the source of truth.
- Projects symlinking to the old folder may need to be relinked if the user
  wants.

## 13. App Startup

Goal: when the app opens, Skillbox reflects a sufficiently reliable system state.

Flow:

```text
User opens Skillbox
  -> Skillbox loads the database
  -> Skillbox checks whether the Skill Host Folder still exists
  -> Skillbox checks global provider locations if configured
  -> Skillbox checks whether project paths still exist
  -> Skillbox displays Dashboard
  -> If a path is missing, UI displays a warning
```

Result:

- User sees a summary overview when the app opens.
- Missing Skill Host Folder, global provider location, or project path is clearly
  reported in the UI.

## 14. Check for App Updates

Goal: user finds out there is a new version of Skillbox to download.

Flow:

```text
User opens About screen (sidebar → About)
  -> Sees the current app version
  -> User may click "Check for Updates"
     -> Compares the latest tag with the current version
     -> If a new version exists: displays a download link to the GitHub Release
     -> If up-to-date: displays "You're up to date"
     -> If network error: displays an error message without blocking the UI
```

Result:

- User knows the running version and can check for new versions when needed.
- App-update check is manual-triggered from About. Plugin update-check is also
  manual-trigger-only (user must click "Check Updates" in Global Plugins). App
  remains 100% usable offline.

## 15. Reset All Data

Goal: user wants to delete all data and start from scratch.

Flow:

```text
User goes to Settings → Danger Zone
  -> Clicks "Reset All Data" (red button)
  -> Confirm dialog step 1: "Delete all data? This cannot be undone."
  -> User types "RESET" into the input to unlock step 2
  -> Clicks Confirm
  -> Go core runs TRUNCATE on all user data tables in a transaction
  -> Resets app_settings + network_settings to defaults
  -> UI reloads to the state as if the app was first installed
```

Result:

- All projects, skills, scan history, and plugin data are deleted.
- DB schema is preserved (migrations do not re-run); app does not restart.
- Settings (install mode, network preferences) are reset to defaults.
- Two-step confirm prevents fat-finger errors.
