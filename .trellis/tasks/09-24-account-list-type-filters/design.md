# Design: account list page size and plan-tier filter

## Boundaries

### Pagination default

Change the shared defaults only:

- `frontend/src/utils/tablePreferences.ts`: `DEFAULT_TABLE_PAGE_SIZE` `20` → `50`
- `backend/internal/service/setting_parse.go` (and any mirrored default map):
  `table_default_page_size` `"20"` → `"50"`

Do not add an accounts-only pageSize override. Existing behavior stays:

1. Explicit `__APP_CONFIG__.table_default_page_size` wins.
2. Else persisted `localStorage['table-page-size']` wins.
3. Else the new default 50.

Update tests that hard-code the factory default of 20.

### Plan-tier filter (separate from account type)

Add a new list query parameter `plan_tier` (not reuse `type`). Frontend keeps
`params.type` as oauth / apikey / setup-token.

UI on `AccountsView`:

- New Select labeled for subscription plan tier.
- Options computed from `params.platform`.
- When `platform === ''`, the control is disabled, value cleared, and no
  `plan_tier` is sent.
- Changing platform clears an incompatible selected plan tier.

## Platform option catalogs

Canonical option values are stable filter IDs. Labels match the badge copy.

### OpenAI

| Filter ID | Label | Stored aliases (normalized) |
| --- | --- | --- |
| `free` | Free | `free` |
| `plus` | Plus | `plus` |
| `pro_5x` | Pro 5x | `prolite` |
| `pro_20x` | Pro 20x | `pro`, `chatgptpro` |
| `business_standard` | Business Standard | `team` |
| `business_premium` | Business Premium | `selfservebusinessprolite` |

Normalization matches `normalizePlanType`: trim, lower-case, strip spaces /
underscores / hyphens.

### Antigravity

| Filter ID | Label | Aliases |
| --- | --- | --- |
| `free` | Free | `free`, empty treated as free only if product already stores Free |
| `pro` | Pro | `pro` |
| `ultra` | Ultra | `ultra` |

### Grok

| Filter ID | Label | Aliases |
| --- | --- | --- |
| `free` | Grok Free | `free`, `basic`, `xbasic` |
| `supergrok` | SuperGrok | `supergrok`, `supergrokpro` (ambiguous paid bucket) |
| `supergrok_lite` | SuperGrok Lite | `supergroklite` |
| `supergrok_plus` | SuperGrok Plus | `supergrokplus` |
| `supergrok_heavy` | SuperGrok Heavy | `supergrokheavy`, `heavy` |

## Backend contract

Extend admin account list:

`GET /admin/accounts?...&plan_tier=<filter_id>`

Rules:

- `plan_tier` without `platform` is ignored (or rejected with 400). Prefer
  ignore + frontend disabled control so old clients stay safe.
- Filtering is platform-scoped alias expansion only. Never apply OpenAI aliases
  when `platform=grok`.
- Match sources:
  - OpenAI / Antigravity: `credentials->>'plan_type'` (and parent account’s
    `credentials->>'plan_type'` when `parent_account_id` is set and the child
    value is empty, so spark shadows stay findable).
  - Grok: `credentials->>'subscription_tier'` OR `credentials->>'plan_type'`,
    compared with the same normalization. Do not attempt SQL replication of
    usage-snapshot Heavy upgrades in this task; document that residual mismatch
    as an accepted MVP limit.
- Push the predicate into `accountListFilteredQuery` so count and page share it.
- Thread `plan_tier` through handler → admin service → repo `ListWithFilters` /
  `ListAllWithFilters` / scheduler-score filter pool helpers that clone the same
  filter set.

Share alias tables between frontend option builders and backend matching via
mirrored constants (TS util + Go helper), tested on both sides against the same
cases as `PlatformTypeBadge.openaiPlans.spec.ts`.

## Compatibility

- Bulk edit / select-all filtered snapshots must include `plan_tier` when present.
- ETag / lite list paths that hash filter inputs should include `plan_tier`.
- i18n: add zh / en labels for the new control; option labels may stay as today
  for plan names that are already English product terms.

## Rollback

Revert the default page-size constants and the `plan_tier` API/UI wiring. No
schema migration is required if filtering uses existing JSON credentials.
