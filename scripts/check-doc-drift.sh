#!/usr/bin/env sh
# check-doc-drift.sh — detect concepts in code with no mention in canonical docs
#
# Usage:
#   ./scripts/check-doc-drift.sh [--quiet] [--verbose] [--git-range <ref>..<ref>]
#
# Exit 0: no gaps. Exit 1: gaps found or ADR required.
# Bypass: git push --no-verify (bypass is visible in git reflog)

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOCS_DIR="$REPO_ROOT/docs"

QUIET=0
VERBOSE=0
GIT_RANGE=""

while [ $# -gt 0 ]; do
  case "$1" in
    --quiet)     QUIET=1 ;;
    --verbose)   VERBOSE=1 ;;
    --git-range) GIT_RANGE="${2:-}"; shift ;;
  esac
  shift
done

# Returns 0 if <term> appears in canonical docs (excludes archive/ and superpowers/).
# Uses -q (quiet) so grep exits on first match — fast.
# Options must come before the pattern; -- cannot be used before --include/--exclude-dir.
in_docs() {
  grep -rql --include="*.md" \
    --exclude-dir=archive --exclude-dir=superpowers \
    "$1" "$DOCS_DIR" 2>/dev/null
}

# Case-insensitive variant.
in_docs_i() {
  grep -rqli --include="*.md" \
    --exclude-dir=archive --exclude-dir=superpowers \
    "$1" "$DOCS_DIR" 2>/dev/null
}

CONCEPT_GAPS=0
CONCEPT_REPORT=""
ADR_FAIL=0
ADR_MSG=""

flag_gap() {
  CONCEPT_GAPS=$((CONCEPT_GAPS + 1))
  CONCEPT_REPORT="${CONCEPT_REPORT}  - Concept: $1\n    Code:     $2\n    Docs:     $3\n"
}

log_ok() {
  [ "$VERBOSE" -eq 1 ] && printf "  OK [%-10s] %s\n" "$1" "$2"
  return 0
}

# ── Tables (migrations) ──────────────────────────────────────────────────────
for table in $(grep -rh "CREATE TABLE" "$REPO_ROOT/core-go/migrations/"*.up.sql 2>/dev/null \
    | grep -oE "CREATE TABLE( IF NOT EXISTS)? [a-zA-Z_]+" \
    | awk '{print $NF}' | sort -u); do
  if in_docs "$table"; then
    log_ok "table" "$table"
  else
    flag_gap "$table" \
      "core-go/migrations/*.up.sql" \
      "docs/06-data-model.md, docs/07-schema-dictionary.md"
  fi
done

# ── RPC namespaces ───────────────────────────────────────────────────────────
# Check by namespace prefix (e.g. "providerPlugin" covers providerPlugin.list, .setEnabled, …)
for ns in $(find "$REPO_ROOT/shared/api-contracts/methods/" -name '*.json' 2>/dev/null \
    | xargs -n1 basename | sed 's/\.json$//' | cut -d'.' -f1 | sort -u); do
  if in_docs "$ns"; then
    log_ok "rpc-ns" "$ns"
  else
    flag_gap "${ns}.*" \
      "shared/api-contracts/methods/${ns}.*.json" \
      "docs/10-technical-architecture.md"
  fi
done

# ── UI screens ───────────────────────────────────────────────────────────────
for screen in $(find "$REPO_ROOT/apps/desktop/renderer/src/screens/" -name '*.tsx' \
    ! -name '*.test.tsx' 2>/dev/null \
    | xargs -n1 basename | sed 's/\.tsx$//' | sort -u); do
  # Strip "-screen" suffix; try both hyphenated and space-separated (docs use title case)
  term=$(printf '%s' "$screen" | sed 's/-screen$//')
  term_space=$(printf '%s' "$term" | tr '-' ' ')
  if in_docs_i "$term" || in_docs_i "$term_space"; then
    log_ok "screen" "$screen"
  else
    flag_gap "$screen" \
      "apps/desktop/renderer/src/screens/${screen}.tsx" \
      "docs/03-information-architecture.md, docs/09-ui-wireframes.md"
  fi
done

# ── UI features ──────────────────────────────────────────────────────────────
for feature in $(find "$REPO_ROOT/apps/desktop/renderer/src/features/" \
    -maxdepth 1 -mindepth 1 -type d 2>/dev/null \
    | xargs -n1 basename | sort -u); do
  # Try exact kebab name, then space-separated case-insensitive (docs use title case)
  term_space=$(printf '%s' "$feature" | tr '-' ' ')
  if in_docs "$feature" || in_docs_i "$term_space"; then
    log_ok "feature" "$feature"
  else
    flag_gap "$feature" \
      "apps/desktop/renderer/src/features/${feature}/" \
      "docs/04-user-flows.md"
  fi
done

# ── Domain objects ───────────────────────────────────────────────────────────
for obj in $(find "$REPO_ROOT/core-go/internal/domain/" -name '*.go' \
    ! -name '*_test.go' 2>/dev/null \
    | xargs -n1 basename | sed 's/\.go$//' | sort -u); do
  # Skip infrastructure/utility files that are not domain concepts
  case "$obj" in
    errors|warning|operation|install) continue ;;
  esac
  if in_docs "$obj"; then
    log_ok "domain" "$obj"
  else
    flag_gap "$obj" \
      "core-go/internal/domain/${obj}.go" \
      "docs/02-product-notes.md, docs/06-data-model.md"
  fi
done

# ── Provider adapters ────────────────────────────────────────────────────────
for provider in $(find "$REPO_ROOT/core-go/internal/providers/" -name '*.go' \
    ! -name '*_test.go' 2>/dev/null \
    | xargs -n1 basename | sed 's/\.go$//' | sort -u); do
  # Skip infrastructure wrappers; only check named provider adapters
  case "$provider" in
    adapter|registry|install_targets|global_adapter|conventional_provider) continue ;;
    *_plugin_remover|*_plugin_writer) continue ;;
  esac
  # Use first underscore-segment as provider identity: "claude_settings" → "claude"
  term=$(printf '%s' "$provider" | cut -d'_' -f1)
  if in_docs "$term"; then
    log_ok "provider" "$provider"
  else
    flag_gap "$provider" \
      "core-go/internal/providers/${provider}.go" \
      "docs/08-provider-model.md"
  fi
done

# ── ADR check (architecture-level additions in the push) ─────────────────────
# Only runs when called with --git-range from the pre-push hook.
# Heuristic: if any new migration, domain object, or RPC method was added
# but no new ADR was created, warn and block.
if [ -n "$GIT_RANGE" ]; then
  new_files=$(git -C "$REPO_ROOT" log --diff-filter=A --name-only --format="" \
    "$GIT_RANGE" 2>/dev/null | grep -v '^$' || true)

  arch_signals=""
  for f in $new_files; do
    case "$f" in
      core-go/migrations/*.up.sql)
        arch_signals="${arch_signals}  + [migration] ${f}\n" ;;
      core-go/internal/domain/*.go)
        arch_signals="${arch_signals}  + [domain]    ${f}\n" ;;
      shared/api-contracts/methods/*.json)
        arch_signals="${arch_signals}  + [rpc]       ${f}\n" ;;
    esac
  done

  if [ -n "$arch_signals" ]; then
    new_adr=$(printf '%s\n' $new_files | grep 'docs/decisions/[0-9]' || true)
    if [ -z "$new_adr" ]; then
      ADR_FAIL=1
      ADR_MSG="${arch_signals}"
    fi
  fi
fi

# ── Output ────────────────────────────────────────────────────────────────────
if [ "$ADR_FAIL" -eq 1 ] && [ "$QUIET" -eq 0 ]; then
  printf '\n%s\n' "══ ADR REQUIRED ══════════════════════════════════════════════════════════"
  printf 'Architecture-level additions detected in this push:\n'
  printf '%b\n' "$ADR_MSG"
  printf 'No new ADR found in docs/decisions/.\n'
  printf 'Create one: cp docs/decisions/template.md docs/decisions/NNNN-title.md\n'
  printf 'See: docs/decisions/README.md for criteria and workflow.\n'
  printf 'Bypass: git push --no-verify (bypass is visible in git reflog)\n'
fi

if [ "$CONCEPT_GAPS" -gt 0 ] && [ "$QUIET" -eq 0 ]; then
  printf '\n%s\n\n' "══ DOC DRIFT DETECTED ════════════════════════════════════════════════════"
  printf 'The following concepts exist in code but are absent from canonical docs:\n\n'
  printf '%b\n' "$CONCEPT_REPORT"
  printf 'Update the indicated docs, then push again.\n'
  printf 'Bypass: git push --no-verify (bypass is visible in git reflog)\n\n'
fi

if [ "$ADR_FAIL" -eq 1 ] || [ "$CONCEPT_GAPS" -gt 0 ]; then
  exit 1
fi

[ "$QUIET" -eq 0 ] && printf 'Doc drift check: OK\n'
exit 0
