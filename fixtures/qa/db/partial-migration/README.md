# Partial Migration DB Fixture

Deterministic setup for `TC-MIGRATE-005`.

The tracked fixture is a script, not a binary database. A QA run creates the DB
inside its run folder:

```sh
fixtures/qa/db/partial-migration/create-dirty-db.sh \
  docs/qa/runs/<run-id>/fixtures/db/partial-migration/qa-partial-migration.db
```

The generated DB contains `schema_migrations(version = 23, dirty = 1)`, which is
the SQLite dirty-migration state used by the Go migration driver after an
interrupted migration. The app must not fall back to real app data when launched
with `SKILLBOX_DB_PATH` pointing at this generated DB.
