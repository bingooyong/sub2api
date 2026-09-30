# Sub2API Account Quality Operations Handoff

## Session

- Date: 2026-09-30
- Scope: `sub2api` account quality operations and intelligence-test support
- Branch: `v0.2.10-hotfix1`
- Base/upstream sync: local branch includes upstream v0.2.10 sync commits before this feature
- HEAD: `4996ae1ed chore: record journal`
- Worktree: clean for tracked files; `.omc/` remains intentionally untracked and must not be committed
- Push/deploy: not pushed, tagged, released, or deployed

## Result

Implemented and verified an opt-in account quality operation path for detecting
degraded account responses. It reuses the existing account forwarding path and
keeps local Codex identity/fingerprint, WebSocket, account isolation, and
connectivity-test behavior intact.

The quality plan supports a probe question and expected answer, model, Cron,
parallel count, independent judge group/model/prompt, bounded judge retries,
selected isolation action, and opt-in automatic restore. Only an explicit
`incorrect` judge verdict can mutate account scheduling or group membership.
Transport errors, timeouts, auth failures, unavailable judges, malformed JSON,
and `unknown` remain non-mutating.

Isolation and restore use leases, account/group version checks, snapshots, and
transactional scheduler outbox invalidation. Durable operation history is stored
in `account_quality_operations`; its API payload redacts probe and judge
prompts, while retained response bodies remain separately prunable.

Admin APIs and `/admin/account-quality` provide plan lifecycle, manual trigger,
latest results, cursor-paginated operation history, operation details, and
localized validation. The existing account intelligence-test UI and scheduled
Pelican test surfaces were extended with dedicated request handling and tests.

This work does not implement private `292/312` state injection and does not
claim to control official provider routing to Luna, Astra, or another model
family.

## Commits

- `c11317269 feat: add account quality operations`
- `89f62a321 chore(task): archive 09-29-account-quality-ops`
- `4996ae1ed chore: record journal`

The Trellis task was archived at:
`.trellis/tasks/archive/2026-09/09-29-account-quality-ops/`.

## Important Paths

- Backend quality service: `backend/internal/service/account_quality.go`
- Judge and strict JSON parsing: `backend/internal/service/quality_judge.go`
- Scheduled quality runner: `backend/internal/service/pelican_scheduled.go`
- Transactional quality repository: `backend/internal/repository/account_quality.go`
- Migrations: `backend/migrations/243_pelican_scheduled_tests.sql`,
  `247_account_quality_ops.sql`, `248_account_quality_judgment.sql`,
  `249_account_quality_operation_history.sql`
- Admin view: `frontend/src/views/admin/AccountQualityView.vue`
- Admin API: `frontend/src/api/admin/accountQuality.ts`
- Implementation spec: `.trellis/spec/backend/account-quality-operations.md`

## Validation Evidence

Passed locally:

- Targeted Go quality, Pelican, repository, handler, and route tests
- Go compile checks for service, repository, admin handler, routes, and migrations
- Frontend `pnpm run typecheck`
- 13 focused frontend/i18n tests
- Frontend production `pnpm run build`
- `git diff --check`
- Trellis context validation before archiving

The integration-tag repository test was attempted but could not start its
PostgreSQL/Redis containers because the local Colima Docker socket failed to
create its mount source. Do not interpret that as a code failure; rerun after a
working Docker/Colima runtime is available.

## Next Session Startup

1. Read `HANDOFF.md`, then this document.
2. Run `git status --short --branch` and confirm only the intentionally ignored
   or untracked `.omc/` state remains.
3. If validating database behavior, start a working PostgreSQL/Redis Docker
   runtime and run:

   ```bash
   cd /Users/bingooyong/Code/01Code/github.com/bingooyong/sub2api/backend
   go test -tags=integration ./internal/repository -count=1
   ```

4. Before any release action, review the current branch against `origin` and
   obtain explicit authorization for push, tag, deployment, or publication.

## Known Concurrent Work

Trellis record mode still lists older unrelated tasks, including account-list
pagination/type filters and upstream hotfix synchronization. Do not archive or
modify those tasks as part of this handoff.
