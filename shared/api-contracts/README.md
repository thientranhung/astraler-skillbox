# Skillbox API Contracts

JSON Schema contracts for the Astraler Skillbox JSON-RPC API.

## Structure

```
shared/api-contracts/
  index.json              Schema manifest (input for generator)
  methods/                Go-handled RPC methods
  notifications/          Server-push notifications (no id)
  shared/                 Shared entity types
  electron/               Electron-handled methods (NOT forwarded to Go)
```

## Naming Conventions

- Schema files: `<method-or-type>.json` using dots for namespace (`host.choose.json`)
- Generated TS files: kebab-case (`host-choose.ts`)
- Request type: `<MethodName>Request` (e.g. `HostChooseRequest`)
- Response type: `<MethodName>Response` (e.g. `HostChooseResponse`)
- Notification type: `<EventName>Notification` (e.g. `OperationProgressNotification`)

## ID Convention

All entity IDs are **integer** (SQLite auto-increment), not UUID.

## Error Codes

App-defined JSON-RPC error codes (outside reserved range -32768..-32000):

| Category | Code |
|---|---|
| `validation_error` | 1001 |
| `filesystem_error` | 1002 |
| `provider_error` | 1003 |
| `database_error` | 1004 |
| `conflict_error` | 1005 |
| `user_cancelled` | 1006 |
| `operation_cancelled` | 1007 |
| `unknown_error` | 1099 |

## Versioning

No `$id` versioning. On breaking changes, rename the file with `.v2` suffix (e.g. `host.choose.v2.json`).

## Generating TypeScript

```bash
(cd apps/desktop && pnpm generate:contracts)
# Check no drift:
(cd apps/desktop && pnpm check:contracts-drift)
```

## Electron-Handled Methods

Schemas under `electron/` describe methods intercepted by Electron main and **never forwarded to the Go core**. They exist for documentation and type generation only.

Current: `dialog.openHostFolder` — opens the native OS folder picker.
