#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/harness/agent-status.sh <tmux-target>

Prints a coarse status for an interactive agent pane and a recent capture.
Statuses are hints for the orchestrator, not final verdicts.
USAGE
}

if [[ $# -ne 1 ]]; then
  usage >&2
  exit 2
fi

target=$1

if ! tmux display-message -p -t "$target" '#{pane_id}' >/dev/null 2>&1; then
  echo "STATUS=missing_target"
  echo "target=$target"
  exit 1
fi

capture=$(tmux capture-pane -t "$target" -p | tail -80)

status=idle_or_waiting
if printf '%s\n' "$capture" | grep -qE 'Enter to select|Do you want to proceed|❯ 1\.|Yes, |No, go back'; then
  status=selection_prompt
elif printf '%s\n' "$capture" | grep -qE 'command not found|Error:|panic:|Traceback|Permission denied'; then
  status=needs_attention
elif printf '%s\n' "$capture" | grep -qE '[$#] $'; then
  status=shell_or_leak
elif printf '%s\n' "$capture" | grep -qE '… *\([0-9]+[smh]|esc to interrupt|◎ /goal active|↓ [0-9]|↑ [0-9]|· [0-9.]+k? tokens|Working \([0-9]+s'; then
  status=busy
elif printf '%s\n' "$capture" | tail -8 | grep -qE '^› .+'; then
  status=queued_input
fi

echo "STATUS=$status"
echo "target=$target"
printf '%s\n' "--- capture ---"
printf '%s\n' "$capture" | grep -n '[^[:space:]]' | tail -50
