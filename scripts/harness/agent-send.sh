#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/harness/agent-send.sh <tmux-target> <prompt...>
  scripts/harness/agent-send.sh <tmux-target> -f <prompt-file>
  scripts/harness/agent-send.sh <tmux-target> -

Sends a prompt to an interactive agent pane through a tmux buffer, submits it,
and prints a short capture so the orchestrator can confirm it entered the TUI.
USAGE
}

if [[ $# -lt 2 ]]; then
  usage >&2
  exit 2
fi

target=$1
shift

if ! tmux display-message -p -t "$target" '#{pane_id}' >/dev/null 2>&1; then
  echo "agent-send: tmux target not found: $target" >&2
  exit 1
fi

if [[ ${1:-} == "-f" ]]; then
  if [[ $# -ne 2 ]]; then
    usage >&2
    exit 2
  fi
  prompt=$(cat "$2")
elif [[ ${1:-} == "-" ]]; then
  prompt=$(cat)
else
  prompt=$*
fi

if [[ -z ${prompt//[[:space:]]/} ]]; then
  echo "agent-send: prompt is empty" >&2
  exit 2
fi

capture=$(tmux capture-pane -t "$target" -p | tail -40)
if printf '%s\n' "$capture" | grep -qE 'Enter to select|Do you want to proceed|❯ 1\.|Yes, |No, go back'; then
  echo "agent-send: target appears to be at a selection/permission prompt; inspect manually" >&2
  printf '%s\n' "$capture" >&2
  exit 1
fi
if printf '%s\n' "$capture" | tail -8 | grep -qE '^› .+'; then
  echo "agent-send: target already has queued input; submit/clear it manually before sending" >&2
  printf '%s\n' "$capture" >&2
  exit 1
fi

buffer="agent-send-$$"
cleanup() {
  tmux delete-buffer -b "$buffer" >/dev/null 2>&1 || true
}
trap cleanup EXIT

tmux send-keys -t "$target" C-u
printf '%s' "$prompt" | tmux load-buffer -b "$buffer" -
tmux paste-buffer -t "$target" -b "$buffer"
tmux send-keys -t "$target" Enter
sleep 1

first_line=${prompt%%$'\n'*}
first_line=${first_line:0:48}
after_capture=$(tmux capture-pane -t "$target" -p | tail -60)
if [[ -n $first_line ]] \
  && printf '%s\n' "$after_capture" | grep -Fq "› $first_line" \
  && ! printf '%s\n' "$after_capture" | grep -qE 'Working \([0-9]+s|esc to interrupt|… *\([0-9]+[smh]'; then
  tmux send-keys -t "$target" Enter
  sleep 1
fi

echo "agent-send: submitted to $target"
tmux capture-pane -t "$target" -p | tail -50
