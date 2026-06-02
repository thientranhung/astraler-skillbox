#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <output-db-path>" >&2
  exit 64
fi

out_db="$1"
out_dir="$(dirname "$out_db")"
mkdir -p "$out_dir"
rm -f "$out_db" "$out_db-wal" "$out_db-shm"

sqlite3 "$out_db" <<'SQL'
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;
CREATE TABLE schema_migrations (
  version INTEGER NOT NULL PRIMARY KEY,
  dirty INTEGER NOT NULL
);
INSERT INTO schema_migrations(version, dirty) VALUES (23, 1);
CREATE TABLE qa_partial_migration_marker (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  scenario TEXT NOT NULL,
  created_at TEXT NOT NULL
);
INSERT INTO qa_partial_migration_marker(id, scenario, created_at)
VALUES (1, 'dirty migration marker for TC-MIGRATE-005', strftime('%Y-%m-%dT%H:%M:%SZ','now'));
PRAGMA integrity_check;
SQL

sqlite3 "$out_db" "SELECT 'schema_migrations', version, dirty FROM schema_migrations;"
