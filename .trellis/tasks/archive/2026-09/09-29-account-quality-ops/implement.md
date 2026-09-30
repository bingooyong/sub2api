# Implementation plan: account quality operations

## Ordered work

1. Run `trellis-before-dev`; read this task, the shared guides, backend/frontend
   indexes, and the existing scheduled-test/account isolation implementations.
2. Map current schema, Ent/repository conventions, cron dispatch, admin route
   registration, API typing, and scheduled-test UI seams. Add no new behavior
   until the existing contracts are covered by focused tests where needed.
3. Add migrations and domain persistence for quality plans, leases, snapshots,
   and operation history. Generate/update only the repository's normal derived
   artifacts.
4. Implement strict judge parsing, candidate selection, bounded retries, and
   the quality runner. Reuse the normal account forwarding path and keep
   connectivity `auto_recover` separate.
5. Implement compare-before-write isolation and opt-in snapshot restoration;
   add operation history and cursor pagination.
6. Add admin handlers/routes, request/response types, API client functions,
   plan form, manual-run action, latest result, history table, and localized
   validation/error copy.
7. Add focused backend and frontend tests from the design verification list.
8. Run Trellis quality check, fix findings, update specs with any durable
   conventions learned, and perform final validation.

## Validation commands

- `python3 ./.trellis/scripts/task.py validate account-quality-ops`
- Targeted Go tests for scheduled-test, quality, account repository, and admin
  handler packages.
- Targeted frontend tests for scheduled-test/account administration components.
- Frontend production build.
- Existing Codex forwarding, identity/fingerprint, WebSocket, account plan-tier,
  and scheduled connectivity tests.
- `git diff --check` and a final `git status --short` confirming `.omc/` is not
  included in the commit.

## Risk points and rollback checkpoints

- Schema/generated-code changes: verify migrations against an existing database
  before proceeding to service changes.
- Runner integration: verify connectivity-only plans before enabling quality
  dispatch.
- Isolation/restore: verify compare-before-write tests before exposing the UI
  toggle for automatic restore.
- API/UI: verify stable enum/nullability payloads before wiring pagination.
- At every checkpoint, preserve the current branch and local fingerprint files;
  rollback by disabling the new quality plan path rather than reverting
  unrelated local changes.
