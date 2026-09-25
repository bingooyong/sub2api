# Account list pagination and plan-tier filters

## Goal

Make admin tables default to 50 rows per page, and let operators filter the
account list by subscription plan tier without confusing that with the existing
oauth / API-key type filter.

## Background

- Page: `/admin/accounts` (`frontend/src/views/admin/AccountsView.vue`).
- Pagination uses shared `useTableLoader` → `getPersistedPageSize()`. The code
  default is `DEFAULT_TABLE_PAGE_SIZE = 20`, and the system setting
  `table_default_page_size` also defaults to `"20"`. When
  `__APP_CONFIG__.table_default_page_size` is set, it overrides localStorage.
  The UI already offers 10 / 20 / 50 / 100.
- The current “类型” filter is `accounts.type` (`oauth` / `apikey` / …). It is
  not the Free / Plus / Business Premium badge.
- Those badges come mainly from `credentials.plan_type` (and parent plan for
  spark shadows; Grok also reads subscription_tier / billing snapshots),
  rendered by `PlatformTypeBadge` via `frontend/src/utils/planType.ts`.
- OpenAI, Antigravity, and Grok use different plan vocabularies. Backend
  `ListAccounts` / `ListWithFilters` has no plan-tier filter today.

## Decisions

- Plan-tier UX is option **A**: add a separate “订阅档位” control. Options
  depend on the selected platform. When platform is “全部”, disable the
  plan-tier filter. Keep the existing oauth / API-key type filter unchanged.
- OpenAI options: Free, Plus, Pro 5x, Pro 20x, Business Standard, Business
  Premium, using the same aliases as `openAIPlanTypeLabel`.
- Default page size **50** is the shared table default: change
  `DEFAULT_TABLE_PAGE_SIZE` and the `table_default_page_size` setting default
  from 20 to 50. Existing localStorage or an explicit configured setting still
  wins.

## Requirements

- Shared first-load table page size is 50 when no localStorage value and no
  explicit `__APP_CONFIG__.table_default_page_size` override is present.
- Operators can filter account plan tier for the selected platform.
- OpenAI filters cover Free, Plus, Pro 5x, Pro 20x, Business Standard, and
  Business Premium with alias matching.
- Antigravity and Grok expose that platform’s known tiers, not ChatGPT labels.
- Empty platform disables plan-tier filtering and sends no plan query param.
- Plan-tier filter combines with type / status / group / privacy / search and
  stays server-paginated.

## Acceptance Criteria

- [ ] With cleared localStorage and no configured override, admin tables
      including `/admin/accounts` request `page_size=50`.
- [ ] Changing page size still persists to `table-page-size` and reloads.
- [ ] Explicit `table_default_page_size` in app config still overrides the code
      default.
- [ ] With platform = OpenAI, operators can filter Free, Plus, Pro 5x, Pro 20x,
      Business Standard, and Business Premium with correct aliases.
- [ ] With platform = Grok or Antigravity, options are that platform’s tiers.
- [ ] With platform empty, plan-tier filtering is disabled and does not narrow
      results by plan.
- [ ] oauth / API-key type filter and other existing filters still work with the
      plan-tier filter.
- [ ] Results remain server-paginated for the combined filters.

## Out of Scope

- Redesigning plan detection / sync or badge labeling beyond filter consistency.
- Replicating Grok’s snapshot-based Heavy inference in SQL for every edge case.
- Client-only filtering of the current page as the long-term solution.
