# Test Design Expansion Inventory

A repo-grounded, pre-release inventory that applies
[`methodology.md`](methodology.md) across every major product surface. Goal: a
single place to see, per surface, the state model, the negative-topology /
failure-matrix dimensions, what existing cases already cover, the gaps, and the
candidate cases needed for release-grade QA.

This is a **proposal**, not the cases themselves. Candidate IDs are reserved here
but no YAML is written until a surface is scheduled. Tiers follow
[`README.md`](README.md): T0 = data-integrity/destructive (blocks release),
T1 = core journey, T2 = secondary. Fixtures reference
[`fixtures-taxonomy.md`](fixtures-taxonomy.md).

Baseline at time of writing: **74 existing cases** (including PR #27 and PR #28
additions). This inventory proposes **41 new candidate cases**; see the
[Summary](#summary).

How to read each surface: **State model** = entity lifecycle edges; **Matrix
dims** = the negative topology / FS×DB×UI / async cells to cover; **Covered** =
existing case IDs; **Gaps → candidates** = proposed new cases.

---

## 1. Setup / Skill Host Folder

- **State model.** Host folder: `unset → chosen → {valid | missing | unreadable | changed}`. Changing the host folder is a re-point edge, not a delete edge.
- **Matrix dims.** Host path: missing / unreadable / not-a-dir / outside-approved / changed-while-installs-exist. UI↔DB: stale host facts after re-point.
- **Covered.** `TC-SETUP-001` (first launch choose), `TC-SETUP-002` (missing host warnings), `TC-SETTINGS-001` (change host without silent relink), `TC-FS-004` (unreadable host reported).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-SETUP-003 | Re-point host folder while project installs exist keeps install records consistent | T0 | host-with-installs |
  | TC-SETUP-004 | Host folder pointed at a file or non-existent path is rejected, prior host retained | T1 | host-is-file |
  | TC-SETTINGS-003 | Re-point host to an empty folder shows zero host skills without dropping install metadata | T1 | empty-host |

## 2. Skills Library

- **State model.** Host skill entry: `absent → scanned → {valid | invalid | not-a-skill}`; list ↔ detail must agree.
- **Matrix dims.** Skill folder shapes: valid / missing SKILL.md / malformed frontmatter / nested / duplicate name / non-skill file. List↔Detail↔FS consistency.
- **Covered.** `TC-SKILL-001` (scan shows host skills), `TC-FS-005` (ignores non-skill files).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-SKILL-008 | Skill with missing or malformed SKILL.md is listed as invalid, not usable | T1 | host/skill-malformed-md |
  | TC-SKILL-009 | Duplicate skill names across host are disambiguated without collision in install | T1 | host/duplicate-names |
  | TC-SKILL-010 | Skill Library detail matches list row and on-disk path after rescan | T2 | host-valid |

## 3. Projects

- **State model.** Project: `added → scanned → {ok | warning | missing-path}`; removal never deletes the folder.
- **Matrix dims.** Project path: valid / nonexistent / file / no-read / moved-after-add. Picker cancel. Projects↔Dashboard count consistency.
- **Covered.** `TC-PROJ-001` (add+scan), `TC-PROJ-002` (auto-scan no dup), `TC-ERROR-001` (picker cancel), `TC-ERROR-002` (nonexistent path rejected), `TC-FS-006` (removal keeps folder), `TC-DASH-001` (dashboard agrees).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-PROJ-004 | Project folder moved/renamed after add reports missing without losing the record | T1 | project-valid |
  | TC-PROJ-005 | Adding the same project path twice does not create a duplicate record | T1 | project-valid |
  | TC-PROJ-006 | Add project pointed at a file (not a directory) is rejected | T2 | project-is-file |

## 4. Project Detail

- **State model.** Detail view derives from project + provider facts; must reflect rescan and missing-provider edges live.
- **Matrix dims.** Provider present/missing/incomplete; broken project symlink; stale detail route after delete.
- **Covered.** `TC-PROJ-003` (rescan marks removed missing), `TC-SKILL-006` (broken symlink warning), `TC-ERROR-004` (stale detail route recovers), `TC-PROVIDER-006/007` (partial structure).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-PROJ-007 | Project Detail with multiple detected providers shows each provider's facts independently | T1 | project-multi-provider |
  | TC-PROJ-008 | Project Detail reflects an external mutation only after explicit rescan (no phantom live state) | T1 | project-valid |

## 5. Project providers

- **State model.** Provider facts: `unknown → detected → {installed | error | missing | incomplete}`; covered well after PR #27/#28.
- **Matrix dims.** Marker present without skills; skills as file/broken-symlink; multiple providers; provider disabled mid-project.
- **Covered.** `TC-PROVIDER-001..008`, `TC-PROVIDER-004` (disable keeps folders).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-PROVIDER-009 | Two providers detected in one project keep independent path/skill facts | T1 | project-multi-provider |
  | TC-PROVIDER-010 | Provider disabled after install hides current state without deleting install records | T1 | project-with-install |

## 6. Add / Remove / Switch install

- **State model.** Install op: `idle → running → {success | failure}`; install record `none → active → removed`; switch = atomic re-target. The mode×topology matrix is the largest untested combination.
- **Matrix dims.** Mode {symlink, copy} × target {missing-folder, exists, read-only, conflict, scaffoldable-provider} × outcome {success, failure}. Remove/switch must preserve host source.
- **Covered.** `TC-SKILL-002` (symlink install), `TC-SKILL-003` (remove keeps host), `TC-SKILL-004` (switch re-targets), `TC-SKILL-005` (conflict no partial), `TC-SKILL-007` (scaffold missing folder), `TC-FS-001` (escape rejected), `TC-FS-002` (remove refuses external symlink), `TC-FS-003` (read-only target), `TC-DB-002` (failed install no metadata), `TC-PROVIDER-008` (delete-before-install).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-SKILL-011 | Copy-mode install writes the skill into the project target and records mode=copy | T0 | project-with-provider |
  | TC-SKILL-012 | Switch from symlink to copy (and back) replaces only the target, host intact | T0 | project-with-install |
  | TC-SKILL-013 | Remove when the project target was already deleted on disk reconciles to removed | T1 | project-with-install |
  | TC-SKILL-014 | Re-install a previously removed skill produces a clean active record (no resurrection) | T1 | project-with-provider |
  | TC-SKILL-015 | Install of the same skill into two providers in one project is independent | T1 | project-multi-provider |

## 7. Global Skills

- **State model.** Global skill state derived from approved global paths only; missing/external paths must not present stale current entries.
- **Matrix dims.** Global path missing / external symlink / disabled provider / detail-vs-list drift.
- **Covered.** `TC-GLOBAL-001..005` (scan, missing path, external symlink, disabled provider, detail match).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-GLOBAL-006 | Global skills path that is a file or broken symlink reports without stale current entries | T1 | global/skills-malformed |
  | TC-GLOBAL-007 | Re-enabling a previously disabled global provider re-scans without duplicating entries | T2 | global-disabled |

## 8. Global Plugins

- **State model.** Plugin: version/enabled state per layer; global ↔ project-override display must agree unless explicitly overridden.
- **Matrix dims.** Settings file missing / malformed / too-large; toggle isolation; project override vs global; manual-update-only.
- **Covered.** `TC-PLUGIN-001..005`, `TC-PROVIDER-003` (global config override in QA home), `TC-ERROR-003` (too-large file).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-PLUGIN-006 | Project plugin override disagreeing with global is labeled as override, not drift | T1 | project-plugin-override |
  | TC-PLUGIN-007 | Missing provider settings file shows no-plugins state without inventing versions | T1 | global/plugin-settings-missing |

## 9. Settings / provider enablement / path overrides

- **State model.** Override: `none → active → reset`; enablement toggles only the selected slot; reset restores built-in path without deleting files.
- **Matrix dims.** Override scope {project, global} × value {valid, `..`, absolute, nonexistent} × reset. Enable/disable isolation. Unknown provider key.
- **Covered.** `TC-PROVIDER-001` (override one slot), `TC-PROVIDER-002` (invalid rejected), `TC-PROVIDER-003` (global config in QA home), `TC-PROVIDER-004` (disable keeps folders), `TC-PROVIDER-005` (unknown key rejected), `TC-SETTINGS-002` (reset all data).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-PROVIDER-011 | Reset of an active override restores built-in path with no file deletion or target rewrite | T0 | project-with-override |
  | TC-PROVIDER-012 | Enable/disable one provider does not change other providers' enabled state or paths | T1 | multi-provider-settings |
  | TC-SETTINGS-004 | Override pointing at a nonexistent path is rejected or flagged, scan behavior unchanged | T1 | project-valid |

## 10. Operations / progress / restart recovery

- **State model.** Operation: `idle → queued → running → {success | failure | cancelled}`; `running --app-restart--> failed`. No duplicate op per target; one terminal state.
- **Matrix dims.** Duplicate prevention × {scan, install, remove, plugin-write, update-check}; cancel; restart-during-running; failure-no-false-success.
- **Covered.** `TC-OPS-001` (host scan no dup), `TC-OPS-002` (project scan no dup), `TC-OPS-003` (cancel safe), `TC-OPS-004` (stale running → failed after restart), `TC-OPS-005` (failure no false toast), `TC-MIGRATE-003` (crash during op).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-OPS-006 | Install/remove operations cannot duplicate for the same skill+provider target | T0 | project-with-provider |
  | TC-OPS-007 | App restart during a running install resolves to a terminal state with consistent DB/FS | T0 | project-with-provider |
  | TC-OPS-008 | Plugin-write operation cannot duplicate and reaches a terminal state | T1 | global-plugin |

## 11. DB / migration / recovery

- **State model.** DB: `current ↔ older(migrate) ↔ corrupt(report)`; metadata FKs and counts coherent; reset preserves schema version.
- **Matrix dims.** Older schema / corrupt DB / migration failure / orphan rows / restart-preserves-metadata.
- **Covered.** `TC-DB-001` (no orphans), `TC-DB-002` (failed install no metadata), `TC-DB-003` (reset preserves schema), `TC-DB-004` (corrupt DB reported), `TC-MIGRATE-001..004`.
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-DB-005 | Foreign-key/count consistency holds after a full add→install→remove→reset cycle | T0 | release-cycle |
  | TC-MIGRATE-005 | Partial/interrupted migration leaves DB recoverable and never touches real app data | T0 | db/partial-migration |

## 12. Offline / local-first

- **State model.** Network: only manual update check is an approved outbound path; no telemetry, no background polling.
- **Matrix dims.** Launch / idle / manual-check / offline core workflows / app-vs-plugin update distinction.
- **Covered.** `TC-PRIVACY-001..004` (launch no network, manual plugin check, app/plugin distinct, offline workflows).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-PRIVACY-005 | No background polling or telemetry occurs during an extended idle session | T1 | network-monitor |

## 13. Packaged app / release

- **State model.** Packaged build: launches with bundled sidecar, isolated app-data, no dev CDP, no orphan sidecar after quit, restart recovery in packaged mode.
- **Matrix dims.** Sidecar path / app-data isolation / CDP exposure / signing+checksums / orphan process / quit+relaunch.
- **Covered.** `TC-PACKAGE-001` (bundled sidecar + isolated DB), `TC-PACKAGE-002` (no Keychain on first launch), `TC-RELEASE-001..005` (preflight, DMG launch, no dev CDP, manifest/checksum, signed verification).
- **Gaps → candidates.**
  | ID | Title | Tier | Fixture |
  |---|---|---|---|
  | TC-PACKAGE-003 | Quit leaves no orphaned Go sidecar process | T0 | packaged-artifact |
  | TC-PACKAGE-004 | Packaged app restart recovers in-progress operations like the dev build | T1 | packaged-artifact |
  | TC-RELEASE-006 | Packaged app uses packaged app-data DB path, never a repo/dev DB | T0 | packaged-artifact |

---

## Summary

### How many new cases

**41 candidate cases** across 13 surfaces:

| Surface | New | of which T0 |
|---|---|---|
| Setup / Skill Host | 3 | 1 |
| Skills Library | 3 | 0 |
| Projects | 3 | 0 |
| Project Detail | 2 | 0 |
| Project providers | 2 | 0 |
| Add/Remove/Switch install | 5 | 2 |
| Global Skills | 2 | 0 |
| Global Plugins | 2 | 0 |
| Settings/enablement/overrides | 3 | 1 |
| Operations/restart recovery | 3 | 2 |
| DB/migration/recovery | 2 | 2 |
| Offline/local-first | 1 | 0 |
| Packaged/release | 3 | 2 |
| **Total** | **41** | **13** |

Tier split: **13 T0, ~22 T1, ~6 T2.**

### Which are release-full

All **13 T0** plus the **T1** cases on core journeys and release paths are
`release-full`. Excluded from `release-full`: the **T2** polish/secondary cases
(`TC-SKILL-010`, `TC-GLOBAL-007`, `TC-PROJ-006`) — record and triage, do not block
release. Net: **~35 of 41** carry the `release-full` tag.

### What to write first (P0)

Order by release risk — class-level gaps first (data integrity, async terminal,
restart recovery), since those are how the 2026-06-02 escape happened:

1. **Install matrix T0** — `TC-SKILL-011` (copy mode), `TC-SKILL-012` (switch mode). Largest untested combination on a destructive path.
2. **Restart recovery T0** — `TC-OPS-007` (restart during install), `TC-PACKAGE-004`, `TC-RELEASE-006`, `TC-PACKAGE-003` (orphan sidecar). Restart + packaged paths are thin today.
3. **DB cycle integrity T0** — `TC-DB-005` (full-cycle FK/count), `TC-MIGRATE-005` (interrupted migration).
4. **Override reset / enablement T0/T1** — `TC-PROVIDER-011`, `TC-PROVIDER-012`, `TC-SETTINGS-004`.
5. **Op duplication T0** — `TC-OPS-006` (install/remove dedup), `TC-OPS-008`.
6. **Host re-point T0** — `TC-SETUP-003`.

P0 set = the **13 T0** candidates. Remaining T1 cases follow once P0 lands; T2
last.

### Fixture roadmap

Most candidates depend on fixture templates that do not exist yet. Build these
(per [`fixtures-taxonomy.md`](fixtures-taxonomy.md)) before writing the cases:

- `project-with-provider`, `project-with-install`, `project-with-override`,
  `project-multi-provider`, `multi-provider-settings`
- `host-with-installs`, `host-is-file`, `empty-host`, `host/skill-malformed-md`,
  `host/duplicate-names`
- `global/skills-malformed`, `global/plugin-settings-missing`,
  `project-plugin-override`, `global-plugin`, `global-disabled`
- `db/partial-migration`, `release-cycle`
- `packaged-artifact` (depends on a built release artifact), `network-monitor`

### Sequencing

1. Land this inventory (proposal). 2. Build the fixture templates above.
3. Write P0 (13 T0) as YAML. 4. Write release-full T1. 5. Add T2 last.
No YAML cases are added until a surface is explicitly scheduled.
