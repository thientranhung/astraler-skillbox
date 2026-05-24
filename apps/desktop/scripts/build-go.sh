#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
go build -o "$REPO_ROOT/apps/desktop/dist/skillbox-core" "$REPO_ROOT/core-go/cmd/skillbox-core"
