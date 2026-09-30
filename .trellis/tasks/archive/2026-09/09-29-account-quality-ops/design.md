# Technical design: account quality operations

## Boundaries

The feature extends the existing scheduled-test domain rather than creating a
second scheduler. Connectivity plans keep their current runner and
`auto_recover` semantics. Quality plans add a distinct operation kind and
state machine behind the same cron/lease dispatch boundary.

Expected touch points are:

- migrations and generated Ent/schema or repository models for plans, runs,
  leases, snapshots, and history;
- `scheduled_test_repo.go` and a focused quality repository/service for
  transactional persistence;
- `scheduled_test_runner_service.go` or its existing runner seam for due-plan
  dispatch, plus a quality judge service for probe/verdict logic;
- admin scheduled-test handler/routes and typed request/response payloads;
- the existing scheduled-test API module and account administration panel;
- localized strings and focused backend/frontend tests.

Do not modify the local Codex fingerprint/identity or WebSocket forwarding
implementation unless a failing integration test identifies a concrete seam.

## Domain flow

1. The existing scheduler finds a due, enabled quality plan and acquires a
   database lease keyed by plan/account/run window.
2. The runner probes the target through the normal account forwarding path.
3. A judge candidate query filters for schedulable, model-compatible accounts
   and excludes the target account. Candidate selection is deterministic for a
   run, with bounded retries.
4. The judge response is decoded from a strict JSON schema. Invalid, missing,
   or conflicting fields become `unknown`.
5. `correct` records success. `unknown` records diagnostics and leaves live
   account state untouched. `incorrect` enters the mutation transaction.
6. The mutation transaction snapshots only fields it will change, applies the
   configured isolation action, and appends an operation-history row. The
   snapshot contains an operation identifier and the values observed before
   mutation.
7. A later complete successful round can restore a snapshot only when each
   field still equals the value written by that operation. If an administrator
   changed a field meanwhile, restoration skips that field and records the
   conflict.

## Persistence and transaction boundaries

Use forward-only migrations compatible with current scheduled-test tables.
Keep plan configuration, run/lease state, mutation snapshots, and history
separable so deleting history cannot delete the live plan. Lease acquisition
must be an atomic insert/update with expiry; release occurs in a `defer` path.

The isolation transaction locks the target account and relevant group rows,
creates the snapshot, applies membership/scheduling changes, and writes the
history event before commit. No network request runs inside this transaction.
Restore uses the same compare-before-write rule and writes a separate history
event. History reads use an opaque cursor over `(created_at, id)`.

## Contracts

Stable API enums:

- verdict: `correct | incorrect | unknown`;
- isolation action: `remove_groups | disable_scheduling`;
- operation outcome: `probed | judged | isolated | restored | skipped | error`.

Nullable diagnostics include judge account, retry count, error code, and
mutation details. Credentials, raw tokens, and full prompts are never returned
or logged. The UI reuses existing admin authorization and localization
patterns.

## Failure and rollback semantics

Network, timeout, authentication, judge availability, malformed JSON, and
lease contention are non-mutating outcomes. A failed migration must leave
existing scheduled-test data readable. If quality execution is disabled or a
plan is deleted, prior snapshots remain auditable but are not automatically
restored. Automatic restore is opt-in and only follows a complete successful
round.

Rollback is a forward code/config rollback: disable quality plans, leave
history and snapshots intact, and run a compatible down-migration only if the
project's migration policy permits it. Do not roll back by resetting existing
account state or by reverting unrelated local identity/fingerprint changes.

## Verification strategy

Backend unit tests cover strict parsing, retry bounds, candidate exclusion,
lease contention, compare-before-restore, and every non-mutating failure path.
Repository/handler tests cover migrations, authorization, stable enums, and
cursor pagination. Frontend tests cover validation, destructive-action copy,
unknown rendering, and pagination state. Finish with targeted Go tests,
frontend tests/build, and existing Codex forwarding/fingerprint regressions.
