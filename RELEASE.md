# Astraler Skillbox — macOS Release Runbook

Operational guide for producing a signed, notarized macOS DMG.
All commands run from `apps/desktop/` unless stated otherwise.

---

## 1. Prerequisites

| Requirement | Detail |
|-------------|--------|
| macOS | 13 Ventura or later (codesign, notarytool, spctl are required) |
| Node.js | 20+ |
| pnpm | 9+ (`pnpm -v`) |
| JS deps installed | `(cd apps/desktop && pnpm install)` |
| Apple Developer Program | Active paid membership |

---

## 2. Credential Setup

### 2a. Code Signing

**Option A — Keychain (recommended for local builds)**

1. Download your **Developer ID Application** certificate from the Apple Developer portal.
2. Double-click the `.p12` to import it into your login keychain.
3. Confirm it appears:
   ```sh
   security find-identity -v -p codesigning | grep "Developer ID Application"
   ```
4. electron-builder picks it up automatically from the keychain; no env vars needed.

**Option B — Env vars (CI / headless)**

```sh
export CSC_LINK=/path/to/certificate.p12   # or base64-encoded contents
export CSC_KEY_PASSWORD=<p12 passphrase>
```

> **Never commit `.p12` files or print `CSC_KEY_PASSWORD` to logs.**

---

### 2b. Notarization

**Option A — App Store Connect API key (recommended)**

```sh
export APPLE_API_KEY_ID=<10-char key ID>
export APPLE_API_ISSUER=<UUID issuer>
export APPLE_API_KEY=/path/to/AuthKey_XXXXXXXXXX.p8
```

**Option B — Apple ID + app-specific password**

```sh
export APPLE_ID=you@example.com
export APPLE_APP_SPECIFIC_PASSWORD=xxxx-xxxx-xxxx-xxxx   # generated at appleid.apple.com
export APPLE_TEAM_ID=<10-char team ID>
```

> **Never commit `.p8` files or print any of these values to logs.**
> App-specific passwords are not your Apple ID password.

---

## 3. Preflight Check

Run before every release attempt:

```sh
cd apps/desktop
pnpm release:mac:check
```

| Result | Meaning |
|--------|---------|
| All checks pass | Machine is ready to build a signed, notarized release |
| `FAIL no signing credential` | No Developer ID Application cert in keychain and no usable `CSC_LINK`/`CSC_KEY_PASSWORD`; customer release cannot proceed |
| `FAIL no complete credential group` | Neither notarization credential group is complete; customer release cannot proceed |
| Other `FAIL ...` | Blocking issue; read the remediation and resolve before proceeding |

Failures are intentional release gates. `release:mac:full` stops at preflight when any required release check fails.

---

## 4. Canonical Release Command

```sh
cd apps/desktop
pnpm release:mac:full
```

This runs the full pipeline in order:

1. `release:mac:check` — preflight gate
2. `package:mac` — electron-builder DMG build (signs if credentials present)
3. `release:mac:verify <dmg>` — artifact integrity check
4. `release:mac:manifest <dmg>` — write manifest JSON + SHA256SUMS

The command exits non-zero on the first failure; fix the error and re-run.

---

## 5. Expected Artifacts

After a successful run, `apps/desktop/dist/` contains:

| File | Description |
|------|-------------|
| `Astraler Skillbox-<version>-arm64.dmg` | Installable DMG |
| `Astraler Skillbox-<version>-arm64.dmg.manifest.json` | Build metadata: version, arch, file size, checksum |
| `SHA256SUMS` | Canonical checksums for all release artifacts |

---

## 6. Post-Release Verification

### Checksum

```sh
# macOS
cd dist && shasum -a 256 -c SHA256SUMS

# Linux (cross-check)
cd dist && sha256sum -c SHA256SUMS
```

### Artifact integrity

```sh
cd apps/desktop
pnpm release:mac:verify dist/Astraler\ Skillbox-<version>-arm64.dmg
```

### Gatekeeper basics (requires signing + notarization)

```sh
# Signature
codesign --verify --verbose=2 /Volumes/Astraler\ Skillbox/Astraler\ Skillbox.app

# Notarization ticket
spctl --assess --type exec --verbose /Volumes/Astraler\ Skillbox/Astraler\ Skillbox.app
```

Expected output for a fully signed and notarized build:
- `codesign`: `valid on disk`, `satisfies its Designated Requirement`
- `spctl`: `accepted` (source: `Notarized Developer ID`)

---

## 7. Current State on This Machine (No Credentials)

This machine currently has:
- No Developer ID Application certificate installed
- No notarization credential group configured

Running `pnpm release:mac:full` will:
- Stop at `release:mac:check`
- Exit non-zero
- Not invoke `package:mac`
- Not produce or verify a customer release artifact

This is the expected customer-release behavior without credentials. Use `pnpm package:mac:unsigned` only for local packaging smoke, not for distribution.

---

## 8. Troubleshooting

### `FAIL no signing credential`
- Keychain path: import the Developer ID Application `.p12` and re-run `pnpm release:mac:check`.
- CI path: set `CSC_LINK` + `CSC_KEY_PASSWORD` env vars.

### `FAIL no complete credential group`
- Set either the API key group (`APPLE_API_KEY_ID`, `APPLE_API_ISSUER`, `APPLE_API_KEY`) or the Apple ID group (`APPLE_ID`, `APPLE_APP_SPECIFIC_PASSWORD`, `APPLE_TEAM_ID`).
- Do not mix both groups — pick one.

### Verify fails: multiple DMGs in dist/
- `release:mac:verify` without an explicit path expects exactly one DMG in `dist/`.
- Pass the path explicitly: `pnpm release:mac:verify dist/<exact-name>.dmg`.
- Or clean `dist/` before rebuilding.

### Verify fails: checksum mismatch
- DMG was modified after manifest generation. Re-run the full pipeline from scratch.

### Manifest/SHA256SUMS missing or stale
- `release:mac:manifest` failed or was skipped. After credentials are installed, run `pnpm release:mac:full` again.
- For an existing local DMG, regenerate integrity files explicitly: `pnpm release:mac:manifest "dist/<exact-name>.dmg"`.

### Notarization timeout / `stapler` error
- Apple's notarization service can be slow. Wait a few minutes and re-run.
- Check status at https://developer.apple.com/system-status/.

### Gatekeeper blocks app on another Mac
- Confirms the build was ad-hoc only. Signing + notarization are required for distribution.

---

## 9. Distribution

The current tooling **does not push or upload** artifacts anywhere.
Distribution (GitHub Releases, S3, CDN) is out of scope for this runbook.
After verifying the artifacts locally, upload them manually to your distribution channel.
