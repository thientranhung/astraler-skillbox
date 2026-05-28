#!/usr/bin/env node
// install-hooks.mjs — configure git to use scripts/hooks/ as the hooks directory.
// Called by apps/desktop postinstall so any `pnpm install` sets it up automatically.

import { execSync } from 'child_process';
import { unlinkSync, lstatSync } from 'fs';
import { join } from 'path';

function run(cmd, opts = {}) {
  return execSync(cmd, { encoding: 'utf8', ...opts }).trim();
}

const repoRoot = run('git rev-parse --show-toplevel');

// Set core.hooksPath — idempotent, safe to run multiple times.
run('git config core.hooksPath scripts/hooks', { cwd: repoRoot });

// Remove any leftover .git/hooks/pre-push symlink from the previous symlink-based install.
const legacyHook = join(repoRoot, '.git', 'hooks', 'pre-push');
try {
  const stat = lstatSync(legacyHook);
  if (stat.isSymbolicLink()) {
    unlinkSync(legacyHook);
    console.log('install-hooks: removed legacy .git/hooks/pre-push symlink');
  }
} catch {
  // File doesn't exist — nothing to clean up.
}

console.log('install-hooks: git hook path set to scripts/hooks/ (via core.hooksPath)');
