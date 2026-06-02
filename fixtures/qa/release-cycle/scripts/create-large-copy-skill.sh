#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: $0 <run-local-release-cycle-dir> [size-mb]" >&2
  exit 64
fi

root="$1"
size_mb="${2:-512}"

case "$size_mb" in
  ''|*[!0-9]*)
    echo "size-mb must be a positive integer" >&2
    exit 64
    ;;
esac

if [ "$size_mb" -lt 1 ]; then
  echo "size-mb must be >= 1" >&2
  exit 64
fi

skill_dir="$root/hosts/host-a/.agents/skills/zz-large-copy-skill"
payload_dir="$skill_dir/payload"
rm -rf "$skill_dir"
mkdir -p "$payload_dir"

cat > "$skill_dir/SKILL.md" <<'EOF'
---
name: zz-large-copy-skill
description: Large run-local QA fixture skill for restart-during-copy-install.
---

# Large Copy Skill

Generated inside a QA run folder. Do not commit generated payload files.
EOF

i=0
while [ "$i" -lt "$size_mb" ]; do
  file="$payload_dir/payload-$(printf '%04d' "$i").bin"
  dd if=/dev/zero of="$file" bs=1048576 count=1 status=none
  i=$((i + 1))
done

echo "created $skill_dir with ${size_mb} MiB payload"
