# Implementation plan

1. Change shared page-size defaults from 20 to 50 in
   `frontend/src/utils/tablePreferences.ts` and
   `backend/internal/service/setting_parse.go` (plus any setting default map /
   tests that assert 20).
2. Add shared plan-tier catalogs and alias helpers:
   - frontend next to `utils/planType.ts`
   - backend helper used by account list filtering
3. Extend repo `accountListFilteredQuery` + service/handler list paths with
   `plan_tier`, including parent plan fallback for empty child OpenAI plan_type
   and Grok credential/tier OR match. Update ETag and scheduler-score filter pool
   call sites that forward the same filters.
4. Update `frontend/src/api/admin/accounts.ts` list filters to pass `plan_tier`.
5. Update `AccountsView.vue`: new plan-tier Select, platform-driven options,
   disable/clear when platform empty, clear on incompatible platform change,
   include `plan_tier` in bulk-filter snapshots.
6. Add i18n strings for the control.
7. Tests:
   - table default page size 50
   - OpenAI alias filter cases (Free / Plus / Pro 5x / Pro 20x / Business *)
   - platform empty ignores plan_tier
   - Antigravity / Grok option catalogs do not use ChatGPT labels
   - AccountsView wiring (options disabled without platform)
8. Run focused frontend unit tests and backend account list / setting tests.

## Validation commands

- `cd frontend && pnpm exec vitest run src/utils/... src/views/admin/__tests__/AccountsView... src/components/common/__tests__/PlatformTypeBadge...`
- `cd backend && go test -count=1 ./internal/handler/admin/ ./internal/repository/ ./internal/service/ -run 'Account|TableDefault|PlanTier|ListAccounts'`
- Manual: `/admin/accounts` with cleared site data shows 50; OpenAI + Free
  returns only Free-badged rows.

## Rollback points

- After defaults change: revert constants if unrelated tables regress.
- After API wiring: feature-flag not required; omit `plan_tier` to restore old
  list behavior.
- After UI: remove Select; list API remains backward compatible.

## Context for implement / check

- `.trellis/spec/frontend/index.md`
- `.trellis/spec/backend/index.md`
- `.trellis/tasks/09-24-account-list-type-filters/prd.md`
- `.trellis/tasks/09-24-account-list-type-filters/design.md`
